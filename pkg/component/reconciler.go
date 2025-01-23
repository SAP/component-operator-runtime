/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/sap/component-operator-runtime/internal/backoff"
	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/internal/metrics"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
	"github.com/sap/component-operator-runtime/pkg/status"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: in general add more retry to overcome 409 update errors (also etcd storage errors because of missed precondition on delete)
// TODO: emitting events to deployment target may fail if corresponding rbac privileges are missing; either this should be pre-discovered or we
// should stop emitting events to remote targets at all; howerver pre-discovering is difficult (may vary from object to object); one option could
// be to send events only if we are cluster-admin
// TODO: allow to override namespace auto-creation on a per-component level
// that is: consider adding them to the PolicyConfiguration interface?
// TODO: allow to override namespace auto-creation on a per-object level
// TODO: run admission webhooks (if present) in reconcile (e.g. as post-read hook)
// TODO: improve overall log output
// TODO: finalizer and fieldowner should be made more configurable (instead of just using the reconciler name)
// TODO: finalizer should have the standard format prefix/finalizer
// TODO: currently, the reconciler always claims/owns dependent objects entirely; but due to server-side-apply it can happen that
// only parts of an object are managed: other parts/fiels might be managed by other actors (or even other components); how to handle such cases?

const (
	readyConditionReasonNew                = "FirstSeen"
	readyConditionReasonPending            = "Pending"
	readyConditionReasonProcessing         = "Processing"
	readyConditionReasonReady              = "Ready"
	readyConditionReasonError              = "Error"
	readyConditionReasonTimeout            = "Timeout"
	readyConditionReasonDeletionPending    = "DeletionPending"
	readyConditionReasonDeletionBlocked    = "DeletionBlocked"
	readyConditionReasonDeletionProcessing = "DeletionProcessing"

	triggerBufferSize = 1024
)

// TODO: should we pass cluster.Client to hooks instead of just client.Client?

// HookFunc is the function signature that can be used to
// establish callbacks at certain points in the reconciliation logic.
// Hooks will be passed a local client (to be precise, the one belonging to the owning
// manager), and the current (potentially unsaved) state of the component.
// Post-hooks will only be called if the according operation (read, reconcile, delete)
// has been successful.
type HookFunc[T Component] func(ctx context.Context, clnt client.Client, component T) error

// ReconcilerOptions are creation options for a Reconciler.
type ReconcilerOptions struct {
	// Which field manager to use in API calls.
	// If unspecified, the reconciler name is used.
	FieldOwner *string
	// Which finalizer to use.
	// If unspecified, the reconciler name is used.
	Finalizer *string
	// Default service account used for impersonation of the target client.
	// If set this service account (in the namespace of the reconciled component) will be used
	// to default the impersonation of the target client (that is, the client used to manage dependents);
	// otherwise no impersonation happens by default, and the controller's own service account is used.
	// Of course, components can still customize impersonation by implementing the ImpersonationConfiguration interface.
	DefaultServiceAccount *string
	// Whether namespaces are auto-created if missing.
	// If unspecified, true is assumed.
	CreateMissingNamespaces *bool
	// How to react if a dependent object exists but has no or a different owner.
	// If unspecified, AdoptionPolicyIfUnowned is assumed.
	// Can be overridden by annotation on object level.
	AdoptionPolicy *reconciler.AdoptionPolicy
	// How to perform updates to dependent objects.
	// If unspecified, UpdatePolicyReplace is assumed.
	// Can be overridden by annotation on object level.
	UpdatePolicy *reconciler.UpdatePolicy
	// How to perform deletion of dependent objects.
	// If unspecified, DeletePolicyDelete is assumed.
	// Can be overridden by annotation on object level.
	DeletePolicy *reconciler.DeletePolicy
	// SchemeBuilder allows to define additional schemes to be made available in the
	// target client.
	SchemeBuilder types.SchemeBuilder
}

// Reconciler provides the implementation of controller-runtime's Reconciler interface, for a given Component type T.
type Reconciler[T Component] struct {
	name               string
	id                 string
	groupVersionKind   schema.GroupVersionKind
	controllerName     string
	client             cluster.Client
	resourceGenerator  manifests.Generator
	statusAnalyzer     status.StatusAnalyzer
	options            ReconcilerOptions
	clients            *cluster.ClientFactory
	backoff            *backoff.Backoff
	postReadHooks      []HookFunc[T]
	preReconcileHooks  []HookFunc[T]
	postReconcileHooks []HookFunc[T]
	preDeleteHooks     []HookFunc[T]
	postDeleteHooks    []HookFunc[T]
	triggerCh          chan event.TypedGenericEvent[apitypes.NamespacedName]
	setupMutex         sync.Mutex
	setupComplete      bool
}

// Create a new Reconciler.
// Here, name should be a meaningful and unique name identifying this reconciler within the Kubernetes cluster; it will be used in annotations, finalizers, and so on;
// resourceGenerator must be an implementation of the manifests.Generator interface.
func NewReconciler[T Component](name string, resourceGenerator manifests.Generator, options ReconcilerOptions) *Reconciler[T] {
	// TOOD: validate options
	// TODO: currently, the defaulting here is identical to the defaulting in the underlying reconciler.Reconciler;
	// under the assumption that these attributes are not used here, we could skip the defaulting here, and let it happen in the underlying implementation only
	if options.FieldOwner == nil {
		options.FieldOwner = &name
	}
	if options.Finalizer == nil {
		options.Finalizer = &name
	}
	if options.CreateMissingNamespaces == nil {
		options.CreateMissingNamespaces = ref(true)
	}
	if options.AdoptionPolicy == nil {
		options.AdoptionPolicy = ref(reconciler.AdoptionPolicyIfUnowned)
	}
	if options.UpdatePolicy == nil {
		options.UpdatePolicy = ref(reconciler.UpdatePolicyReplace)
	}
	if options.DeletePolicy == nil {
		options.DeletePolicy = ref(reconciler.DeletePolicyDelete)
	}

	return &Reconciler[T]{
		name:              name,
		resourceGenerator: resourceGenerator,
		// TODO: make statusAnalyzer specifiable via options?
		statusAnalyzer: status.NewStatusAnalyzer(name),
		options:        options,
		// TODO: make backoff configurable via options?
		backoff:       backoff.NewBackoff(10 * time.Second),
		postReadHooks: []HookFunc[T]{resolveReferences[T]},
		triggerCh:     make(chan event.TypedGenericEvent[apitypes.NamespacedName], triggerBufferSize),
	}
}

// Reconcile contains the actual reconciliation logic.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	r.setupMutex.Lock()
	if !r.setupComplete {
		defer r.setupMutex.Unlock()
		panic("usage error: setup must be called first")
	}
	r.setupMutex.Unlock()

	log := log.FromContext(ctx)
	log.V(1).Info("running reconcile")

	metrics.Reconciles.WithLabelValues(r.controllerName).Inc()

	now := metav1.Now()

	// fetch reconciled object
	component := newComponent[T]()
	if err := r.client.Get(ctx, req.NamespacedName, component); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("not found; ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "unexpected get error")
	}
	component.GetObjectKind().SetGroupVersionKind(r.groupVersionKind)

	// fetch requeue interval, retry interval and timeout
	requeueInterval := time.Duration(0)
	if requeueConfiguration, ok := assertRequeueConfiguration(component); ok {
		requeueInterval = requeueConfiguration.GetRequeueInterval()
	}
	if requeueInterval == 0 {
		requeueInterval = 10 * time.Minute
	}
	retryInterval := time.Duration(0)
	if retryConfiguration, ok := assertRetryConfiguration(component); ok {
		retryInterval = retryConfiguration.GetRetryInterval()
	}
	if retryInterval == 0 {
		retryInterval = requeueInterval
	}
	timeout := time.Duration(0)
	if timeoutConfiguration, ok := assertTimeoutConfiguration(component); ok {
		timeout = timeoutConfiguration.GetTimeout()
	}
	if timeout == 0 {
		timeout = requeueInterval
	}

	// convenience accessors
	status := component.GetStatus()
	savedStatus := status.DeepCopy()

	// always attempt to update the status
	skipStatusUpdate := false
	defer func() {
		if r := recover(); r != nil {
			log.Error(fmt.Errorf("panic occurred during reconcile"), "panic", r)
			// re-panic in order skip the remaining steps
			panic(r)
		}

		status.ObservedGeneration = component.GetGeneration()

		if status.State == StateReady || err != nil {
			// clear backoff if state is ready (obviously) or if there is an error;
			// even is the error is a RetriableError which will be turned into a non-error;
			// this is correct, because in that case, the RequeueAfter will be determined through the RetriableError
			r.backoff.Forget(req)
		}
		if status.State != StateProcessing || err != nil {
			// clear ProcessingDigest and ProcessingSince in all non-error cases where state is StateProcessing
			status.ProcessingDigest = ""
			status.ProcessingSince = nil
		}
		if status.State == StateProcessing && err == nil && now.Sub(status.ProcessingSince.Time) >= timeout {
			// TODO: maybe it would be better to have a dedicated StateTimeout?
			// note: it is guaranteed that status.ProcessingSince is not nil here because
			// - it was not cleared above because of the mutually exclusive clauses on status.State and err
			// - it was set during reconcile when state was set to StateProcessing
			status.SetState(StateError, readyConditionReasonTimeout, "Reconcilation of dependent resources timed out")
		}

		if err != nil {
			// convert retriable errors into non-errors (Pending or DeletionPending state), and return specified or default backoff
			retriableError := &types.RetriableError{}
			if errors.As(err, retriableError) {
				retryAfter := retriableError.RetryAfter()
				if retryAfter == nil || *retryAfter == 0 {
					retryAfter = &retryInterval
				}
				// TODO: allow RetriableError to provide custom reason and message
				if component.GetDeletionTimestamp().IsZero() {
					status.SetState(StatePending, readyConditionReasonPending, capitalize(retriableError.Error()))
				} else {
					status.SetState(StateDeletionPending, readyConditionReasonDeletionPending, capitalize(retriableError.Error()))
				}
				result = ctrl.Result{RequeueAfter: *retryAfter}
				err = nil
			} else {
				status.SetState(StateError, readyConditionReasonError, err.Error())
			}
		}

		if result.RequeueAfter > 0 {
			// add jitter of 1-5 percent to RequeueAfter
			addJitter(&result.RequeueAfter, 1, 5)
		}

		log.V(1).Info("reconcile done", "withError", err != nil, "requeue", result.Requeue || result.RequeueAfter > 0, "requeueAfter", result.RequeueAfter.String())
		if err != nil {
			if status, ok := err.(apierrors.APIStatus); ok || errors.As(err, &status) {
				metrics.ReconcileErrors.WithLabelValues(r.controllerName, strconv.Itoa(int(status.Status().Code))).Inc()
			} else {
				metrics.ReconcileErrors.WithLabelValues(r.controllerName, "other").Inc()
			}
		}

		// TODO: should we move this behind the DeepEqual check below to avoid noise?
		// also note: it seems that no events will be written if the component's namespace is in deletion
		state, reason, message := status.GetState()
		if state == StateError {
			r.client.EventRecorder().Event(component, corev1.EventTypeWarning, reason, message)
		} else {
			r.client.EventRecorder().Event(component, corev1.EventTypeNormal, reason, message)
		}

		if skipStatusUpdate {
			return
		}
		if reflect.DeepEqual(status, savedStatus) {
			return
		}

		// note: it's crucial to set the following timestamps late (otherwise the DeepEqual() check above would always be false)
		// on the other hand it's a bit weird, because LastObservedAt will not be updated if no other changes have happened to the status;
		// and same for the conditions' LastTransitionTime timestamps;
		// maybe we should remove this optimization, and always do the Update() call
		status.LastObservedAt = &now
		for i := 0; i < len(status.Conditions); i++ {
			cond := &status.Conditions[i]
			if savedCond := savedStatus.getCondition(cond.Type); savedCond == nil || cond.Status != savedCond.Status {
				cond.LastTransitionTime = &now
			}
		}
		if updateErr := r.client.Status().Update(ctx, component, client.FieldOwner(*r.options.FieldOwner)); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// set a first status (and requeue, because the status update itself will not trigger another reconciliation because of the event filter set)
	if status.ObservedGeneration <= 0 {
		status.SetState(StatePending, readyConditionReasonNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// run post-read hooks
	// note: it's important that this happens after deferring the status handler
	// TODO: enhance ctx with tailored logger and event recorder
	// TODO: enhance ctx  with the local client
	hookCtx := NewContext(ctx).WithReconcilerName(r.name)
	for hookOrder, hook := range r.postReadHooks {
		if err := hook(hookCtx, r.client, component); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "error running post-read hook (%d)", hookOrder)
		}
	}

	// setup target
	targetClient, err := r.getClientForComponent(component)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error getting client for component")
	}
	targetOptions := r.getOptionsForComponent(component)
	target := newReconcileTarget[T](r.name, r.id, targetClient, r.resourceGenerator, targetOptions)
	// TODO: enhance ctx with tailored logger and event recorder
	// TODO: enhance ctx  with the local client
	hookCtx = NewContext(ctx).WithReconcilerName(r.name).WithClient(targetClient)

	// do the reconciliation
	if component.GetDeletionTimestamp().IsZero() {
		// create/update case
		// TODO: optionally (to be completely consistent) set finalizer through a mutating webhook
		if added := controllerutil.AddFinalizer(component, *r.options.Finalizer); added {
			if err := r.client.Update(ctx, component, client.FieldOwner(*r.options.FieldOwner)); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "error adding finalizer")
			}
			// trigger another round trip
			// this is necessary because the update call invalidates potential changes done to the component by the post-read
			// hook above; this means, not to the object itself, but for example to loaded secrets or config maps;
			// in the following round trip, the finalizer will already be there, and the update will not happen again
			return ctrl.Result{Requeue: true}, nil
		}

		log.V(2).Info("reconciling dependent resources")
		for hookOrder, hook := range r.preReconcileHooks {
			if err := hook(hookCtx, r.client, component); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "error running pre-reconcile hook (%d)", hookOrder)
			}
		}
		ok, digest, err := target.Apply(ctx, component)
		if err != nil {
			log.V(1).Info("error while reconciling dependent resources")
			return ctrl.Result{}, errors.Wrap(err, "error reconciling dependent resources")
		}
		if ok {
			for hookOrder, hook := range r.postReconcileHooks {
				if err := hook(hookCtx, r.client, component); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "error running post-reconcile hook (%d)", hookOrder)
				}
			}
			log.V(1).Info("all dependent resources successfully reconciled")
			status.AppliedGeneration = component.GetGeneration()
			status.LastAppliedAt = &now
			status.SetState(StateReady, readyConditionReasonReady, "Dependent resources successfully reconciled")
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		} else {
			log.V(1).Info("not all dependent resources successfully reconciled")
			if digest != status.ProcessingDigest {
				status.ProcessingDigest = digest
				status.ProcessingSince = &now
				r.backoff.Forget(req)
			}
			if !reflect.DeepEqual(status.Inventory, savedStatus.Inventory) {
				r.backoff.Forget(req)
			}
			status.SetState(StateProcessing, readyConditionReasonProcessing, "Reconcilation of dependent resources triggered; waiting until all dependent resources are ready")
			return ctrl.Result{RequeueAfter: r.backoff.Next(req, readyConditionReasonProcessing)}, nil
		}
	} else {
		for hookOrder, hook := range r.preDeleteHooks {
			if err := hook(hookCtx, r.client, component); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "error running pre-delete hook (%d)", hookOrder)
			}
		}
		allowed, msg, err := target.IsDeletionAllowed(ctx, component)
		if err != nil {
			log.V(1).Info("error while checking if deletion is allowed")
			return ctrl.Result{}, errors.Wrap(err, "error checking whether deletion is possible")
		}
		if !allowed {
			// deletion is blocked because of existing managed CROs and so on
			log.V(1).Info("deletion not allowed")
			// TODO: have an additional StateDeletionBlocked?
			// TODO: eliminate this msg logic
			r.client.EventRecorder().Event(component, corev1.EventTypeNormal, readyConditionReasonDeletionBlocked, "Deletion blocked: "+msg)
			status.SetState(StateDeleting, readyConditionReasonDeletionBlocked, "Deletion blocked: "+msg)
			return ctrl.Result{RequeueAfter: 1*time.Second + r.backoff.Next(req, readyConditionReasonDeletionBlocked)}, nil
		}
		if len(slices.Remove(component.GetFinalizers(), *r.options.Finalizer)) > 0 {
			// deletion is blocked because of foreign finalizers
			log.V(1).Info("deleted blocked due to existence of foreign finalizers")
			// TODO: have an additional StateDeletionBlocked?
			r.client.EventRecorder().Event(component, corev1.EventTypeNormal, readyConditionReasonDeletionBlocked, "Deletion blocked due to existing foreign finalizers")
			status.SetState(StateDeleting, readyConditionReasonDeletionBlocked, "Deletion blocked due to existing foreign finalizers")
			return ctrl.Result{RequeueAfter: 1*time.Second + r.backoff.Next(req, readyConditionReasonDeletionBlocked)}, nil
		}
		// deletion case
		log.V(2).Info("deleting dependent resources")
		ok, err := target.Delete(ctx, component)
		if err != nil {
			log.V(1).Info("error while deleting dependent resources")
			return ctrl.Result{}, errors.Wrap(err, "error deleting dependent resources")
		}
		if ok {
			for hookOrder, hook := range r.postDeleteHooks {
				if err := hook(hookCtx, r.client, component); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "error running post-delete hook (%d)", hookOrder)
				}
			}
			// all dependent resources are already gone, so that's it
			log.V(1).Info("all dependent resources are successfully deleted; removing finalizer")
			if removed := controllerutil.RemoveFinalizer(component, *r.options.Finalizer); removed {
				if err := r.client.Update(ctx, component, client.FieldOwner(*r.options.FieldOwner)); err != nil {
					return ctrl.Result{}, errors.Wrap(err, "error removing finalizer")
				}
			}
			// skip status update, since the instance will anyway deleted timely by the API server
			// this will avoid unnecessary ugly 409'ish error messages in the logs
			// (occurring in the case that API server would delete the resource in the course of the subsequent reconciliation)
			skipStatusUpdate = true
			return ctrl.Result{}, nil
		} else {
			// deletion triggered for dependent resources, but some are not yet gone
			log.V(1).Info("not all dependent resources are successfully deleted")
			if !reflect.DeepEqual(status.Inventory, savedStatus.Inventory) {
				r.backoff.Forget(req)
			}
			status.SetState(StateDeleting, readyConditionReasonDeletionProcessing, "Deletion of dependent resources triggered; waiting until dependent resources are deleted")
			return ctrl.Result{RequeueAfter: r.backoff.Next(req, readyConditionReasonDeletionProcessing)}, nil
		}
	}
}

// Trigger ad-hoc reconcilation for specified component.
func (r *Reconciler[T]) Trigger(namespace string, name string) error {
	select {
	case r.triggerCh <- event.TypedGenericEvent[apitypes.NamespacedName]{Object: apitypes.NamespacedName{Namespace: namespace, Name: name}}:
		return nil
	default:
		return fmt.Errorf("error triggering reconcile for component %s/%s (buffer full)", namespace, name)
	}
}

// Register post-read hook with reconciler.
// This hook will be called after the reconciled component object has been retrieved from the Kubernetes API.
func (r *Reconciler[T]) WithPostReadHook(hook HookFunc[T]) *Reconciler[T] {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: hooks can only be registered before setup was called")
	}
	r.postReadHooks = append(r.postReadHooks, hook)
	return r
}

// Register pre-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right before the reconcilation of the dependent objects starts.
func (r *Reconciler[T]) WithPreReconcileHook(hook HookFunc[T]) *Reconciler[T] {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: hooks can only be registered before setup was called")
	}
	r.preReconcileHooks = append(r.preReconcileHooks, hook)
	return r
}

// Register post-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right after the reconcilation of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostReconcileHook(hook HookFunc[T]) *Reconciler[T] {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: hooks can only be registered before setup was called")
	}
	r.postReconcileHooks = append(r.postReconcileHooks, hook)
	return r
}

// Register pre-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right before the deletion of the dependent objects starts.
func (r *Reconciler[T]) WithPreDeleteHook(hook HookFunc[T]) *Reconciler[T] {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: hooks can only be registered before setup was called")
	}
	r.preDeleteHooks = append(r.preDeleteHooks, hook)
	return r
}

// Register post-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right after the deletion of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostDeleteHook(hook HookFunc[T]) *Reconciler[T] {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: hooks can only be registered before setup was called")
	}
	r.postDeleteHooks = append(r.postDeleteHooks, hook)
	return r
}

// Register the reconciler with a given controller-runtime Manager and Builder.
// This will call For() and Complete() on the provided builder.
// It populates the recnciler's client with an enhnanced client derived from mgr.GetClient() and mgr.GetConfig().
// That client is used for three purposes:
// - reading/updating the reconciled component, sending events for this component
// - it is passed to hooks
// - it is passed to the factory for target clients as a default local client
func (r *Reconciler[T]) SetupWithManagerAndBuilder(mgr ctrl.Manager, blder *ctrl.Builder) error {
	r.setupMutex.Lock()
	defer r.setupMutex.Unlock()
	if r.setupComplete {
		panic("usage error: setup must not be called more than once")
	}

	kubeSystemNamespace := &corev1.Namespace{}
	if err := mgr.GetAPIReader().Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, kubeSystemNamespace); err != nil {
		return errors.Wrap(err, "error retrieving uid of kube-system namespace")
	}
	r.id = string(kubeSystemNamespace.UID)

	discoveryClient, err := discovery.NewDiscoveryClientForConfigAndClient(mgr.GetConfig(), mgr.GetHTTPClient())
	if err != nil {
		return errors.Wrap(err, "error creating discovery client")
	}
	r.client = cluster.NewClient(mgr.GetClient(), discoveryClient, mgr.GetEventRecorderFor(r.name))

	component := newComponent[T]()
	r.groupVersionKind, err = apiutil.GVKForObject(component, r.client.Scheme())
	if err != nil {
		return errors.Wrap(err, "error getting type metadata for component")
	}
	// TODO: should this be more fully qualified, or configurable?
	// for now we reproduce the controller-runtime default (the lowercase kind of the reconciled type)
	r.controllerName = strings.ToLower(r.groupVersionKind.Kind)

	var schemeBuilders []types.SchemeBuilder
	if schemeBuilder, ok := r.resourceGenerator.(types.SchemeBuilder); ok {
		schemeBuilders = append(schemeBuilders, schemeBuilder)
	}
	if r.options.SchemeBuilder != nil {
		schemeBuilders = append(schemeBuilders, r.options.SchemeBuilder)
	}
	r.clients, err = cluster.NewClientFactory(r.name, r.controllerName, mgr.GetConfig(), schemeBuilders)
	if err != nil {
		return errors.Wrap(err, "error creating client factory")
	}

	if err := blder.
		For(component, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))).
		WatchesRawSource(source.Channel(
			r.triggerCh,
			handler.TypedFuncs[apitypes.NamespacedName, reconcile.Request]{GenericFunc: func(ctx context.Context, e event.TypedGenericEvent[apitypes.NamespacedName], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				q.Add(reconcile.Request{NamespacedName: e.Object})
			}},
			source.WithBufferSize[apitypes.NamespacedName, reconcile.Request](triggerBufferSize))).
		Named(r.controllerName).
		Complete(r); err != nil {
		return errors.Wrap(err, "error creating controller")
	}

	r.setupComplete = true
	return nil
}

// Register the reconciler with a given controller-runtime Manager.
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	return r.SetupWithManagerAndBuilder(
		mgr,
		ctrl.NewControllerManagedBy(mgr).
			WithOptions(controller.Options{MaxConcurrentReconciles: 5}),
	)
}

func (r *Reconciler[T]) getClientForComponent(component T) (cluster.Client, error) {
	placementConfiguration, havePlacementConfiguration := assertPlacementConfiguration(component)
	clientConfiguration, haveClientConfiguration := assertClientConfiguration(component)
	impersonationConfiguration, haveImpersonationConfiguration := assertImpersonationConfiguration(component)

	var kubeConfig []byte
	var impersonationUser string
	var impersonationGroups []string
	if haveClientConfiguration {
		kubeConfig = clientConfiguration.GetKubeConfig()
	}
	if haveImpersonationConfiguration {
		impersonationUser = impersonationConfiguration.GetImpersonationUser()
		impersonationGroups = impersonationConfiguration.GetImpersonationGroups()
		if m := regexp.MustCompile(`^(system:serviceaccount):(.*):(.+)$`).FindStringSubmatch(impersonationUser); m != nil {
			if m[2] == "" {
				namespace := ""
				if havePlacementConfiguration {
					namespace = placementConfiguration.GetDeploymentNamespace()
				}
				if namespace == "" {
					namespace = component.GetNamespace()
				}
				impersonationUser = fmt.Sprintf("%s:%s:%s", m[1], namespace, m[3])
			}
		}
	}
	if len(kubeConfig) == 0 && impersonationUser == "" && len(impersonationGroups) == 0 && r.options.DefaultServiceAccount != nil {
		impersonationUser = fmt.Sprintf("system:serviceaccount:%s:%s", component.GetNamespace(), *r.options.DefaultServiceAccount)
	}
	clnt, err := r.clients.Get(kubeConfig, impersonationUser, impersonationGroups)
	if err != nil {
		return nil, errors.Wrap(err, "error getting remote or impersonated client")
	}
	return clnt, nil
}

func (r *Reconciler[T]) getOptionsForComponent(component T) reconciler.ReconcilerOptions {
	options := reconciler.ReconcilerOptions{
		FieldOwner:              r.options.FieldOwner,
		Finalizer:               r.options.Finalizer,
		CreateMissingNamespaces: r.options.CreateMissingNamespaces,
		AdoptionPolicy:          r.options.AdoptionPolicy,
		UpdatePolicy:            r.options.UpdatePolicy,
		DeletePolicy:            r.options.DeletePolicy,
		StatusAnalyzer:          r.statusAnalyzer,
		Metrics: reconciler.ReconcilerMetrics{
			ReadCounter:   metrics.Operations.WithLabelValues(r.controllerName, "read"),
			CreateCounter: metrics.Operations.WithLabelValues(r.controllerName, "create"),
			UpdateCounter: metrics.Operations.WithLabelValues(r.controllerName, "update"),
			DeleteCounter: metrics.Operations.WithLabelValues(r.controllerName, "delete"),
		},
	}
	if policyConfiguration, ok := assertPolicyConfiguration(component); ok {
		// TODO: check the values returned by the PolicyConfiguration
		if adoptionPolicy := policyConfiguration.GetAdoptionPolicy(); adoptionPolicy != "" {
			options.AdoptionPolicy = &adoptionPolicy
		}
		if updatePolicy := policyConfiguration.GetUpdatePolicy(); updatePolicy != "" {
			options.UpdatePolicy = &updatePolicy
		}
		if deletePolicy := policyConfiguration.GetDeletePolicy(); deletePolicy != "" {
			options.DeletePolicy = &deletePolicy
		}
	}
	return options
}

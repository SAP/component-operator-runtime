/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/sap/component-operator-runtime/internal/backoff"
	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: in general add more retry to overcome 409 update errors (also etcd storage errors because of missed precondition on delete)
// TODO: optionally allow to use server-side-apply instead of create/update (e.g. through a component annotation)
// TODO: make a can-i check before emitting events to deployment target (e.g. in the client factory when creating the client)
// TODO: allow to override namespace auto-creation and adoption policy settings on a per-component level
// (e.g. through annotations or another interface that components could optionally implement)
// TODO: run admission webhooks (if present) in reconcile (e.g. as post-read hook)

const (
	readyConditionReasonNew                = "FirstSeen"
	readyConditionPending                  = "Pending"
	readyConditionReasonProcessing         = "Processing"
	readyConditionReasonReady              = "Ready"
	readyConditionReasonError              = "Error"
	readyConditionReasonDeletionBlocked    = "DeletionBlocked"
	readyConditionReasonDeletionProcessing = "DeletionProcessing"
)

// HookFunc is the function signature that can be used to
// establish callbacks at certain points in the reconciliation logic.
// Hooks will be passed the current (potentially unsaved) state of the component.
// Post-hooks will only be called if the according operation (read, reconcile, delete)
// has been successful.
type HookFunc[T Component] func(ctx context.Context, client client.Client, component T) error

// ReconcilerOptions are creation options for a Reconciler.
type ReconcilerOptions struct {
	// Whether namespaces are auto-created if missing.
	// If unspecified, true is assumed.
	CreateMissingNamespaces *bool
	// How to react if a dependent object exists but has no or a different owner.
	// If unspecified, AdoptionPolicyAdoptUnowned is assumed.
	AdoptionPolicy *AdoptionPolicy
	// Schemebuilder allows to define additional schemes to be made available in the
	// target client.
	SchemeBuilder types.SchemeBuilder
}

// AdoptionPolicy defines how the reconciler reacts if a dependent object exists but has no or a different owner.
type AdoptionPolicy string

const (
	// Fail if the dependent object exists but has no or a different owner.
	AdoptionPolicyFail AdoptionPolicy = "Fail"
	// Adopt existing dependent objects if they have no owner set.
	AdoptionPolicyAdoptUnowned AdoptionPolicy = "AdoptUnowned"
	// Adopt all existing dependent objects, even if they have a conflicting owner.
	AdoptionPolicyAdoptAll AdoptionPolicy = "AdoptAll"
)

// Reconciler provides the implementation of controller-runtime's Reconciler interface, for a given Component type T.
type Reconciler[T Component] struct {
	name               string
	id                 string
	client             cluster.Client
	resourceGenerator  manifests.Generator
	options            ReconcilerOptions
	clients            *cluster.ClientFactory
	backoff            *backoff.Backoff
	postReadHooks      []HookFunc[T]
	preReconcileHooks  []HookFunc[T]
	postReconcileHooks []HookFunc[T]
	preDeleteHooks     []HookFunc[T]
	postDeleteHooks    []HookFunc[T]
	setupMutex         sync.Mutex
	setupComplete      bool
}

// Create a new Reconciler.
// Here, name should be a meaningful and unique name identifying this reconciler within the Kubernetes cluster; it will be used in annotations, finalizers, and so on;
// resourceGenerator must be an implementation of the manifests.Generator interface.
func NewReconciler[T Component](name string, resourceGenerator manifests.Generator, options ReconcilerOptions) *Reconciler[T] {
	if options.CreateMissingNamespaces == nil {
		options.CreateMissingNamespaces = &[]bool{true}[0]
	}
	if options.AdoptionPolicy == nil {
		options.AdoptionPolicy = &[]AdoptionPolicy{AdoptionPolicyAdoptUnowned}[0]
	}
	// TOOD: validate adoption policy

	return &Reconciler[T]{
		name:              name,
		resourceGenerator: resourceGenerator,
		options:           options,
		backoff:           backoff.NewBackoff(5 * time.Second),
		postReadHooks:     []HookFunc[T]{resolveReferences[T]},
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

	// convenience accessors
	status := component.GetStatus()
	savedStatus := status.DeepCopy()

	// requeue/retry interval
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

	// always attempt to update the status
	skipStatusUpdate := false
	defer func() {
		log.V(1).Info("reconcile done", "withError", err != nil, "requeue", result.Requeue || result.RequeueAfter > 0, "requeueAfter", result.RequeueAfter.String())
		if status.State == StateReady || err != nil {
			r.backoff.Forget(req)
		}
		status.ObservedGeneration = component.GetGeneration()
		if err != nil {
			retriableError := &types.RetriableError{}
			if errors.As(err, retriableError) {
				retryAfter := retriableError.RetryAfter()
				if retryAfter == nil || *retryAfter == 0 {
					retryAfter = &retryInterval
				}
				status.SetState(StatePending, readyConditionPending, err.Error())
				result = ctrl.Result{RequeueAfter: *retryAfter}
				err = nil
			} else {
				status.SetState(StateError, readyConditionReasonError, err.Error())
			}
		}
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
		// note: it's crucial to set the following timestamp late (otherwise the DeepEqual() check before would always be false)
		status.LastObservedAt = &now
		if updateErr := r.client.Status().Update(ctx, component, client.FieldOwner(r.name)); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// run post-read hooks
	// note: it's important that this happens after deferring the status handler
	for hookOrder, hook := range r.postReadHooks {
		if err := hook(ctx, r.client, component); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "error running post-read hook (%d)", hookOrder)
		}
	}

	// set a first status (and requeue, because the status update itself will not trigger another reconciliation because of the event filter set)
	if status.ObservedGeneration <= 0 {
		status.SetState(StateProcessing, readyConditionReasonNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// setup target
	targetClient, err := r.getClientForComponent(component)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error getting client for component")
	}
	target := newReconcileTarget[T](r.name, r.id, targetClient, r.resourceGenerator, *r.options.CreateMissingNamespaces, *r.options.AdoptionPolicy)
	hookCtx := newContext(ctx).WithClient(targetClient)

	// do the reconciliation
	if component.GetDeletionTimestamp().IsZero() {
		// create/update case
		// TODO: optionally (to be completely consistent) set finalizer through a mutating webhook
		if added := controllerutil.AddFinalizer(component, r.name); added {
			if err := r.client.Update(ctx, component, client.FieldOwner(r.name)); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "error adding finalizer")
			}
			// trigger another round trip
			// this is necessary because the update call invalidates potential changes done by the post-read hook above
			// in the following round trip, the finalizer will already be there, and the update will not happen again
			return ctrl.Result{Requeue: true}, nil
		}

		log.V(2).Info("reconciling dependent resources")
		for hookOrder, hook := range r.preReconcileHooks {
			if err := hook(hookCtx, r.client, component); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "error running pre-reconcile hook (%d)", hookOrder)
			}
		}
		ok, err := target.Reconcile(ctx, component)
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
			status.SetState(StateReady, readyConditionReasonReady, "Dependent resources successfully reconciled")
			status.AppliedGeneration = component.GetGeneration()
			status.LastAppliedAt = &now
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		} else {
			log.V(1).Info("not all dependent resources successfully reconciled")
			status.SetState(StateProcessing, readyConditionReasonProcessing, "Reconcilation of dependent resources triggered; waiting until all dependent resources are ready")
			if !reflect.DeepEqual(status.Inventory, savedStatus.Inventory) {
				r.backoff.Forget(req)
			}
			return ctrl.Result{RequeueAfter: r.backoff.Next(req, readyConditionReasonProcessing)}, nil
		}
	} else if allowed, msg, err := target.IsDeletionAllowed(ctx, component); err != nil || !allowed {
		// deletion is blocked because of existing managed CROs and so on
		// TODO: eliminate this msg logic
		if err != nil {
			log.V(1).Info("error while checking if deletion is allowed")
			return ctrl.Result{}, errors.Wrap(err, "error checking whether deletion is possible")
		}
		log.V(1).Info("deletion not allowed")
		status.SetState(StateDeleting, readyConditionReasonDeletionBlocked, "Deletion blocked: "+msg)
		return ctrl.Result{RequeueAfter: 1*time.Second + r.backoff.Next(req, readyConditionReasonDeletionBlocked)}, nil
	} else if len(slices.Remove(component.GetFinalizers(), r.name)) > 0 {
		// deletion is blocked because of foreign finalizers
		log.V(1).Info("deleted blocked due to existence of foreign finalizers")
		status.SetState(StateDeleting, readyConditionReasonDeletionBlocked, "Deletion blocked due to existing foreign finalizers")
		return ctrl.Result{RequeueAfter: 1*time.Second + r.backoff.Next(req, readyConditionReasonDeletionBlocked)}, nil
	} else {
		// deletion case
		log.V(2).Info("deleting dependent resources")
		for hookOrder, hook := range r.preDeleteHooks {
			if err := hook(hookCtx, r.client, component); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "error running pre-delete hook (%d)", hookOrder)
			}
		}
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
			if removed := controllerutil.RemoveFinalizer(component, r.name); removed {
				if err := r.client.Update(ctx, component, client.FieldOwner(r.name)); err != nil {
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
			status.SetState(StateDeleting, readyConditionReasonDeletionProcessing, "Deletion of dependent resources triggered; waiting until dependent resources are deleted")
			if !reflect.DeepEqual(status.Inventory, savedStatus.Inventory) {
				r.backoff.Forget(req)
			}
			return ctrl.Result{RequeueAfter: r.backoff.Next(req, readyConditionReasonDeletionProcessing)}, nil
		}
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

	var schemeBuilders []types.SchemeBuilder
	if schemeBuilder, ok := r.resourceGenerator.(types.SchemeBuilder); ok {
		schemeBuilders = append(schemeBuilders, schemeBuilder)
	}
	if r.options.SchemeBuilder != nil {
		schemeBuilders = append(schemeBuilders, r.options.SchemeBuilder)
	}
	r.clients, err = cluster.NewClientFactory(r.name, mgr.GetConfig(), schemeBuilders)
	if err != nil {
		return errors.Wrap(err, "error creating client factory")
	}

	component := newComponent[T]()
	if err := blder.
		For(component, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))).
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
			WithOptions(controller.Options{MaxConcurrentReconciles: 3}),
	)
}

func (r *Reconciler[T]) getClientForComponent(component T) (cluster.Client, error) {
	placementConfiguration, havePlacementConfiguration := assertPlacementConfiguration(component)
	clientConfiguration, haveClientConfiguration := assertClientConfiguration(component)
	impersonationConfiguration, haveImpersonationConfiguration := assertImpersonationConfiguration(component)
	haveCustomScheme := func() bool { _, ok := r.resourceGenerator.(types.SchemeBuilder); return ok }() || r.options.SchemeBuilder != nil
	// TODO: we should always return a factory client, even in the default case;
	// however this would be an incompatible change; people who previously supplied a custom scheme via the manager's client
	// would now have to do the same by adding AddToScheme() to the used generator.
	if !haveClientConfiguration && !haveImpersonationConfiguration && !haveCustomScheme {
		return r.client, nil
	}
	var kubeconfig []byte
	var impersonationUser string
	var impersonationGroups []string
	if haveClientConfiguration {
		kubeconfig = clientConfiguration.GetKubeConfig()
	}
	if haveImpersonationConfiguration {
		impersonationUser = impersonationConfiguration.GetImpersonationUser()
		impersonationGroups = impersonationConfiguration.GetImpersonationGroups()
		r := regexp.MustCompile(`^(system:serviceaccount):(.*):(.+)$`)
		if m := r.FindStringSubmatch(impersonationUser); m != nil {
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
	client, err := r.clients.Get(kubeconfig, impersonationUser, impersonationGroups)
	if err != nil {
		return nil, errors.Wrap(err, "error getting remote or impersonated client")
	}
	return client, nil
}

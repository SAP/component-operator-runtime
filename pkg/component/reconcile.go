/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/sap/component-operator-runtime/internal/backoff"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: simplify the manager's client creation (do not force it to uncache T and the apiextensions/apiregistration types)
// TODO: the default client does not cache unstructured objects; from the perspective of this package it should be fine to cache unstructured ...

// TODO: we should disallow generators to produce manifests using metadata.generatName (it would break the whole inventory logic)

// TODO: in general add more retry to overcome 409 update errors (also etcd storage errors because of missed precondition on delete)

// TODO: it might be desirable to run the component operator outside the target cluster;
// maybe even in a 1:n layout (where one component operator manages multiple target clusters);
// in this case, we would need a way to specify a kubeconfig per operator CRO;
// the difficulty in implementing this lies mostly with the complex generators (helm, kustomize)
// which currently expect a client/discoveryClient at creation time (instead of generation time)

const (
	readyConditionReasonNew                = "FirstSeen"
	readyConditionReasonProcessing         = "Processing"
	readyConditionReasonReady              = "Ready"
	readyConditionReasonError              = "Error"
	readyConditionReasonDeletionBlocked    = "DeletionBlocked"
	readyConditionReasonDeletionProcessing = "DeletionProcessing"
)

const (
	objectReasonCreated     = "Created"
	objectReasonUpdated     = "Updated"
	objectReasonUpdateError = "UpdateError"
	objectReasonDeleted     = "Deleted"
	objectReasonDeleteError = "DeleteError"
)

const (
	reconcilePolicyOnObjectChange            = "on-object-change"
	reconcilePolicyOnObjectOrComponentChange = "on-object-or-component-change"
	reconcilePolicyOnce                      = "once"
)

const (
	updatePolicyDefault  = "default"
	updatePolicyRecreate = "recreate"
)

const (
	scopeUnknown = iota
	scopeNamespaced
	scopeCluster
)

// HookFunc is the function signature that can be used to
// establish callbacks at certain points in the reconciliation logic.
// Hooks will be passed the current (potentially unsaved) state of the component.
// Post-hooks will only be called if the previous operator (read, reconcile, delete)
// has been successful.
type HookFunc[T Component] func(ctx context.Context, client client.Client, component T) error

// Reconciler provides the implementation of controller-runtime's Reconciler interface, for a given Component type T.
type Reconciler[T Component] struct {
	name               string
	client             client.Client
	discoveryClient    discovery.DiscoveryInterface
	eventRecorder      record.EventRecorder
	scheme             *runtime.Scheme
	resourceGenerator  manifests.Generator
	backoff            *backoff.Backoff
	postReadHooks      []HookFunc[T]
	preReconcileHooks  []HookFunc[T]
	postReconcileHooks []HookFunc[T]
	preDeleteHooks     []HookFunc[T]
	postDeleteHooks    []HookFunc[T]
}

// Create a new Reconciler. Here:
//   - name should be a meaningful and unique name identifying this reconciler with the Kubernetes cluster; it will be used in annotations, finalizers, and so on
//   - client should be a controller-runtime client for the current cluster, typically it is mgr.GetClient();
//     it should have informer caching disabled for the reconciled type T, as well as for apiextensionsv1.CustomResourceDefinition and apiregistrationv1.APIService
//   - discoveryClient should be a discovery client for the current cluster, typically built from mgr.GetConfig() and mgr.GetHTTPClient()
//   - eventRecorder should be an event recorder for the current cluster, typically it is mgr.GetEventRecorderFor(name)
//   - scheme is required to recognize the core group (corev1), the api group containing T, and apiextensionsv1 and apiregistrationv1;
//     in addition, scheme must know about all concrete (i.e. non-unstructured) types returned by the given resource generator
//   - resourceGenerator must be an implementation of the manifests.Generator interface.
//
// Deprecation warning: the parameters client, discoveryClient, eventRecorder, scheme will be removed; the deprecation will happen in three phases:
//   - Phase 1: the current behavior remains unchanged, but consumers should ensure to pass the deprecated parameters exactly as follows:
//     client: mgr.GetClient()
//     discoveryClient: a discovery client instantiated through mgr.GetConfig() and mgr.GetHTTPClient()
//     eventRecorder: mgr.GetEventRecorderFor(name)
//     scheme: mgr.GetScheme()
//     here, mgr is the owning manager;
//     as an alternative the above parameters can be passed as nil, in which case they will be defaulted as just described;
//     the returned Reconciler object must not be used before SetupWithManager() was called.
//   - Phase 2: the parameters client, discoveryClient, eventRecorder, scheme will be ignored (can be passed as nil);
//     the reconcile logic will infer client, discoveryClient, eventRecorder and scheme from the Manager instance passed to SetupWithManager(), as described above.
//   - Phase 3: the deprecated parameters will be removed.
func NewReconciler[T Component](name string, client client.Client, discoveryClient discovery.DiscoveryInterface, eventRecorder record.EventRecorder, scheme *runtime.Scheme, resourceGenerator manifests.Generator) *Reconciler[T] {
	return &Reconciler[T]{
		name:              name,
		client:            client,
		discoveryClient:   discoveryClient,
		eventRecorder:     eventRecorder,
		scheme:            scheme,
		resourceGenerator: resourceGenerator,
		backoff:           backoff.NewBackoff(5 * time.Second),
	}
}

// Reconcile contains the actual reconciliation logic.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
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

	// always attempt to update the status
	skipStatusUpdate := false
	defer func() {
		log.V(1).Info("reconcile done", "withError", err != nil, "requeue", result.Requeue || result.RequeueAfter > 0, "requeueAfter", result.RequeueAfter.String())
		if status.State == StateReady || err != nil {
			r.backoff.Forget(req)
		}
		status.ObservedGeneration = component.GetGeneration()
		if err != nil {
			status.SetState(StateError, readyConditionReasonError, err.Error())
		}
		state, reason, message := status.GetState()
		if state == StateError {
			r.eventRecorder.Event(component, corev1.EventTypeWarning, reason, message)
		} else {
			r.eventRecorder.Event(component, corev1.EventTypeNormal, reason, message)
		}
		if skipStatusUpdate {
			return
		}
		if reflect.DeepEqual(status, savedStatus) {
			return
		}
		// note: it's crucial to set the following timestamp late (otherwise the DeepEqual() check before would always be false)
		status.LastObservedAt = &now
		if updateErr := r.client.Status().Update(ctx, component); updateErr != nil {
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
	target := newReconcileTarget[T](r.name, r.client, r.discoveryClient, r.eventRecorder, r.scheme, r.resourceGenerator)

	// do the reconciliation
	if component.GetDeletionTimestamp().IsZero() {
		// create/update case
		if added := controllerutil.AddFinalizer(component, r.name); added {
			if err := r.client.Update(ctx, component); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "error adding finalizer")
			}
			// trigger another round trip
			// this is necessary because the update call invalidates potential changes done by the post-read hook above
			// in the following round trip, the finalizer will already be there, and the update will not happen again
			return ctrl.Result{Requeue: true}, nil
		}

		log.V(2).Info("reconciling dependent resources")
		for hookOrder, hook := range r.preReconcileHooks {
			if err := hook(ctx, r.client, component); err != nil {
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
				if err := hook(ctx, r.client, component); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "error running post-reconcile hook (%d)", hookOrder)
				}
			}
			log.V(1).Info("all dependent resources successfully reconciled")
			status.SetState(StateReady, readyConditionReasonReady, "Dependent resources successfully reconciled")
			status.AppliedGeneration = component.GetGeneration()
			status.LastAppliedAt = &now
			return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
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
			if err := hook(ctx, r.client, component); err != nil {
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
				if err := hook(ctx, r.client, component); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "error running post-delete hook (%d)", hookOrder)
				}
			}
			// all dependent resources are already gone, so that's it
			log.V(1).Info("all dependent resources are successfully deleted; removing finalizer")
			if removed := controllerutil.RemoveFinalizer(component, r.name); removed {
				if err := r.client.Update(ctx, component); err != nil {
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
	r.postReadHooks = append(r.postReadHooks, hook)
	return r
}

// Register pre-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right before the reconcilation of the dependent objects starts.
func (r *Reconciler[T]) WithPreReconcileHook(hook HookFunc[T]) *Reconciler[T] {
	r.preReconcileHooks = append(r.preReconcileHooks, hook)
	return r
}

// Register post-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right after the reconcilation of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostReconcileHook(hook HookFunc[T]) *Reconciler[T] {
	r.postReconcileHooks = append(r.postReconcileHooks, hook)
	return r
}

// Register pre-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right before the deletion of the dependent objects starts.
func (r *Reconciler[T]) WithPreDeleteHook(hook HookFunc[T]) *Reconciler[T] {
	r.preDeleteHooks = append(r.preDeleteHooks, hook)
	return r
}

// Register post-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right after the deletion of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostDeleteHook(hook HookFunc[T]) *Reconciler[T] {
	r.postDeleteHooks = append(r.postDeleteHooks, hook)
	return r
}

// Register the reconciler with a given controller-runtime Manager.
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: add some check (and lock?) to prevent SetupWithManager() being called more than once
	// TODO: proceed with the deprecation of NewReconciler() arguments (see comments there); that is, remove the below if statements ...
	if r.client == nil {
		r.client = mgr.GetClient()
	}
	if r.discoveryClient == nil {
		discoveryClient, err := discovery.NewDiscoveryClientForConfigAndClient(mgr.GetConfig(), mgr.GetHTTPClient())
		if err != nil {
			return errors.Wrap(err, "error creating discovery client")
		}
		r.discoveryClient = discoveryClient
	}
	if r.eventRecorder == nil {
		r.eventRecorder = mgr.GetEventRecorderFor(r.name)
	}
	if r.scheme == nil {
		r.scheme = mgr.GetScheme()
	}

	component := newComponent[T]()
	return ctrl.NewControllerManagedBy(mgr).
		For(component).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		Complete(r)
}

// ReconcileTarget describes the deployment target of a reconcile run.
type reconcileTarget[T Component] struct {
	name                         string
	client                       client.Client
	discoveryClient              discovery.DiscoveryInterface
	eventRecorder                record.EventRecorder
	scheme                       *runtime.Scheme
	resourceGenerator            manifests.Generator
	labelKeyOwnerId              string
	annotationKeyDigest          string
	annotationKeyReconcilePolicy string
	annotationKeyUpdatePolicy    string
	annotationKeyOrder           string
	annotationKeyPurgeOrder      string
	annotationKeyOwnerId         string
}

func newReconcileTarget[T Component](
	name string,
	client client.Client,
	discoveryClient discovery.DiscoveryInterface,
	eventRecorder record.EventRecorder,
	scheme *runtime.Scheme, resourceGenerator manifests.Generator,
) *reconcileTarget[T] {
	return &reconcileTarget[T]{
		name:                         name,
		client:                       client,
		discoveryClient:              discoveryClient,
		eventRecorder:                eventRecorder,
		scheme:                       scheme,
		resourceGenerator:            resourceGenerator,
		labelKeyOwnerId:              name + "/owner-id",
		annotationKeyDigest:          name + "/digest",
		annotationKeyReconcilePolicy: name + "/reconcile-policy",
		annotationKeyUpdatePolicy:    name + "/update-policy",
		annotationKeyOrder:           name + "/order",
		annotationKeyPurgeOrder:      name + "/purge-order",
		annotationKeyOwnerId:         name + "/owner-id",
	}
}

func (t *reconcileTarget[T]) Reconcile(ctx context.Context, component T) (bool, error) {
	namespace := component.GetDeploymentNamespace()
	name := component.GetDeploymentName()
	ownerId := component.GetNamespace() + "/" + component.GetName()
	status := component.GetStatus()

	// render manifests
	objects, err := t.resourceGenerator.Generate(namespace, name, component.GetSpec())
	if err != nil {
		return false, errors.Wrap(err, "error rendering manifests")
	}

	// normalize objects; that means:
	// - check that unstructured objects have valid type information set, and convert them to their concrete type if known to the scheme
	// - check that non-unstructured types are known to the scheme, and validate/set their type information
	normalizedObjects := make([]client.Object, len(objects))
	for i, object := range objects {
		gvk := object.GetObjectKind().GroupVersionKind()
		if unstructuredObject, ok := object.(*unstructured.Unstructured); ok {
			if gvk.Version == "" || gvk.Kind == "" {
				return false, fmt.Errorf("unstructured object %s is missing type information", types.ObjectKeyToString(object))
			}
			if t.scheme.Recognizes(gvk) {
				typedObject, err := t.scheme.New(gvk)
				if err != nil {
					return false, errors.Wrapf(err, "error instantiating type for object %s", types.ObjectKeyToString(object))
				}
				if typedObject, ok := typedObject.(client.Object); ok {
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, typedObject); err != nil {
						return false, errors.Wrapf(err, "error converting object %s", types.ObjectKeyToString(object))
					}
					normalizedObjects[i] = typedObject
				} else {
					return false, errors.Wrapf(err, "error instantiating type for object %s", types.ObjectKeyToString(object))
				}
			} else if isCrd(object) || isApiService(object) {
				return false, fmt.Errorf("scheme does not recognize type of object %s", types.ObjectKeyToString(object))
			} else {
				normalizedObjects[i] = object
			}
		} else {
			_gvk, err := apiutil.GVKForObject(object, t.scheme)
			if err != nil {
				return false, errors.Wrapf(err, "error retrieving scheme type information for object %s", types.ObjectKeyToString(object))
			}
			if gvk.Version == "" || gvk.Kind == "" {
				object.GetObjectKind().SetGroupVersionKind(_gvk)
			} else if gvk != _gvk {
				return false, fmt.Errorf("object %s specifies inconsistent type information (expected: %s)", types.ObjectKeyToString(object), _gvk)
			}
			normalizedObjects[i] = object
		}
	}
	objects = normalizedObjects

	// validate type and set namespace for namespaced objects which have no namespace set
	for _, object := range objects {
		// note: due to the normalization done before, every object will now have a valid object kind set
		gvk := object.GetObjectKind().GroupVersionKind()

		scope := scopeUnknown
		restMapping, err := t.client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err == nil {
			scope = scopeFromRestMapping(restMapping)
		} else if !meta.IsNoMatchError(err) {
			return false, errors.Wrapf(err, "error getting rest mapping for object %s", types.ObjectKeyToString(object))
		}
		for _, crd := range getCrds(objects) {
			if crd.Spec.Group == gvk.Group && crd.Spec.Names.Kind == gvk.Kind {
				scope = scopeFromCrd(crd)
				err = nil
				break
			}
		}
		for _, apiService := range getApiServices(objects) {
			if apiService.Spec.Group == gvk.Group && apiService.Spec.Version == gvk.Version {
				err = nil
				break
			}
		}
		if err != nil {
			return false, errors.Wrapf(err, "error getting rest mapping for object %s", types.ObjectKeyToString(object))
		}

		if object.GetNamespace() == "" && scope == scopeNamespaced {
			object.SetNamespace(namespace)
		}
	}

	// validate order annotations and define getter functions for later usage
	getAnnotationInt := func(obj client.Object, key string, minValue int, maxValue int, defaultValue int) (int, error) {
		if value, ok := obj.GetAnnotations()[key]; ok {
			value, err := strconv.Atoi(value)
			if err != nil {
				return 0, err
			}
			if value < minValue || value > maxValue {
				return 0, fmt.Errorf("value %d not in allowed range [%d,%d]", value, minValue, maxValue)
			}
			return value, nil
		} else {
			return defaultValue, nil
		}
	}
	for _, object := range objects {
		if _, err := getAnnotationInt(object, t.annotationKeyOrder, math.MinInt16, math.MaxInt16, 0); err != nil {
			return false, errors.Wrapf(err, "invalid value for annotation %s", t.annotationKeyOrder)
		}
		if _, err := getAnnotationInt(object, t.annotationKeyPurgeOrder, math.MinInt16, math.MaxInt16, math.MaxInt); err != nil {
			return false, errors.Wrapf(err, "invalid value for annotation %s", t.annotationKeyPurgeOrder)
		}
	}
	getOrder := func(object client.Object) int {
		order, err := getAnnotationInt(object, t.annotationKeyOrder, math.MinInt16, math.MaxInt16, 0)
		if err != nil {
			panic("this cannot happen")
		}
		return order
	}
	getPurgeOrder := func(object client.Object) int {
		order, err := getAnnotationInt(object, t.annotationKeyPurgeOrder, math.MinInt16, math.MaxInt16, math.MaxInt)
		if err != nil {
			panic("this cannot happen")
		}
		return order
	}

	// add/update inventory with target objects
	numAdded := 0
	for _, object := range objects {
		// retrieve inventory item belonging to this object (if existing)
		item := getItem(status.Inventory, object)

		// calculate object digest
		raw, err := json.Marshal(object)
		if err != nil {
			return false, errors.Wrapf(err, "error serializing object %s", types.ObjectKeyToString(object))
		}
		digest := sha256hash(raw)

		reconcilePolicy := object.GetAnnotations()[t.annotationKeyReconcilePolicy]
		switch reconcilePolicy {
		case reconcilePolicyOnObjectChange, "":
			reconcilePolicy = reconcilePolicyOnObjectChange
		case reconcilePolicyOnObjectOrComponentChange:
			digest = fmt.Sprintf("%s@%d", digest, component.GetGeneration())
		case reconcilePolicyOnce:
			// note: if the object already existed with a different reconcile policy, then it will get reconciled one (and only one) more time
			digest = "__once__"
		default:
			return false, fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyReconcilePolicy, reconcilePolicy)
		}

		// if item was not found, append an empty item
		if item == nil {
			// fetch object (if existing)
			existingObject, err := t.readObject(ctx, object)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", types.ObjectKeyToString(object))
			}
			// check ownership
			if existingObject != nil {
				existingOwnerId := existingObject.GetAnnotations()[t.annotationKeyOwnerId]
				if existingOwnerId == "" {
					// TODO: make this configurable by some switch on Reconciler (or even per Component instance)
					processForeign := true
					if !processForeign {
						return false, fmt.Errorf("found existing object %s without owner", types.ObjectKeyToString(object))
					}
				} else if existingOwnerId != ownerId {
					return false, fmt.Errorf("owner conflict; object %s is owned by %s", types.ObjectKeyToString(object), existingOwnerId)
				}
			}
			status.Inventory = append(status.Inventory, &InventoryItem{})
			item = status.Inventory[len(status.Inventory)-1]
			numAdded++
		}

		// update item
		if digest != item.Digest {
			gvk := object.GetObjectKind().GroupVersionKind()
			item.Group = gvk.Group
			item.Version = gvk.Version
			item.Kind = gvk.Kind
			item.Namespace = object.GetNamespace()
			item.Name = object.GetName()
			item.ManagedTypes = getManagedTypes(object)
			item.Digest = digest
			item.Phase = PhaseScheduledForApplication
			item.Status = kstatus.InProgressStatus.String()
		}
	}

	// mark obsolete inventory items (clear digest)
	for _, item := range status.Inventory {
		found := false
		for _, object := range objects {
			if item.Matches(object) {
				found = true
				break
			}
		}
		if !found && item.Digest != "" {
			item.Digest = ""
			item.Phase = PhaseScheduledForDeletion
			item.Status = kstatus.TerminatingStatus.String()
		}
	}

	// trigger another reconcile
	if numAdded > 0 {
		// put inventory into right order for future deletion
		status.Inventory = sortObjectsForDelete(status.Inventory)
		return false, nil
	}

	// note: after this point it is guaranteed that the persisted inventory reflects the target state
	// now it is about to synchronize the cluster state with the inventory

	// count instances of managed types which are about to be deleted
	numManagedToBeDeleted := 0
	for _, item := range status.Inventory {
		if item.Phase == PhaseScheduledForDeletion || item.Phase == PhaseScheduledForCompletion || item.Phase == PhaseDeleting || item.Phase == PhaseCompleting {
			if t.isManaged(item, component) {
				numManagedToBeDeleted++
			}
		}
	}

	// delete redundant objects and maintain inventory
	numToBeDeleted := 0
	var inventory []*InventoryItem
	for _, item := range status.Inventory {
		if item.Phase == PhaseScheduledForDeletion || item.Phase == PhaseScheduledForCompletion || item.Phase == PhaseDeleting || item.Phase == PhaseCompleting {
			// fetch object (if existing)
			existingObject, err := t.readObject(ctx, item)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", item)
			}

			switch item.Phase {
			case PhaseScheduledForDeletion:
				if numManagedToBeDeleted == 0 || t.isManaged(item, component) {
					// note: here is a theoretical risk that we delete an existing foreign object, because informers are not yet synced
					// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
					if err := t.deleteObject(ctx, item, existingObject); err != nil {
						return false, errors.Wrapf(err, "error deleting object %s", item)
					}
					item.Phase = PhaseDeleting
					item.Status = kstatus.TerminatingStatus.String()
				}
				numToBeDeleted++
			case PhaseScheduledForCompletion:
				if numManagedToBeDeleted == 0 || t.isManaged(item, component) {
					// note: here is a theoretical risk that we delete an existing foreign object, because informers are not yet synced
					// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
					if err := t.deleteObject(ctx, item, existingObject); err != nil {
						return false, errors.Wrapf(err, "error deleting object %s", item)
					}
					item.Phase = PhaseCompleting
					item.Status = kstatus.TerminatingStatus.String()
				}
				numToBeDeleted++
			case PhaseDeleting:
				// if object is gone, we can remove it from inventory
				if existingObject == nil {
					continue
				} else {
					numToBeDeleted++
				}
			case PhaseCompleting:
				// if object is gone, it is set to completed, and kept in inventory
				if existingObject == nil {
					item.Phase = PhaseCompleted
					item.Status = ""
				} else {
					numToBeDeleted++
				}
			default:
				panic("this cannot happen")
			}
		}
		inventory = append(inventory, item)
	}

	status.Inventory = inventory

	// trigger another reconcile
	if numToBeDeleted > 0 {
		return false, nil
	}

	// note: after this point, PhaseScheduledForDeletion, PhaseScheduledForCompletion, PhaseDeleting, PhaseCompleting cannot occur anymore in status.Inventory
	// in other words: status.Inventory and objects contains the same resources

	// create missing namespaces
	// TODO: make this more configurable
	for _, namespace := range findMissingNamespaces(objects) {
		if err := t.client.Get(ctx, apitypes.NamespacedName{Name: namespace}, &corev1.Namespace{}); err != nil {
			if !apierrors.IsNotFound(err) {
				return false, errors.Wrapf(err, "error reading namespace %s", namespace)
			}
			if err := t.client.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
				return false, errors.Wrapf(err, "error creating namespace %s", namespace)
			}
		}
	}

	// put objects into right order for applying
	objects = sortObjectsForApply(objects, getOrder)

	// apply new objects and maintain inventory
	numUnready := 0
	numNotManagedToBeApplied := 0
	for k, object := range objects {
		// retreive update policy
		updatePolicy := object.GetAnnotations()[t.annotationKeyUpdatePolicy]
		switch updatePolicy {
		case updatePolicyDefault, "":
			updatePolicy = updatePolicyDefault
		case updatePolicyRecreate:
		default:
			return false, fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyUpdatePolicy, updatePolicy)
		}

		// retrieve object order
		order := getOrder(object)

		// retrieve inventory item corresponding to this object
		item := mustGetItem(status.Inventory, object)

		// if this is the first object of an order, then
		// count instances of managed types in this order which are about to be applied
		if k == 0 || getOrder(objects[k-1]) < order {
			numNotManagedToBeApplied = 0
			for j := k; j < len(objects) && getOrder(objects[j]) == order; j++ {
				_object := objects[j]
				_item := mustGetItem(status.Inventory, _object)
				if _item.Phase != PhaseReady && _item.Phase != PhaseCompleted && !t.isManaged(_object, component) {
					// that means: _item.Phase is one of PhaseScheduledForApplication, PhaseCreating, PhaseUpdating
					numNotManagedToBeApplied++
				}
			}
		}

		// for non-completed objects, compute and update status, and apply (create or update) the object if necessary
		if item.Phase != PhaseCompleted {
			if numNotManagedToBeApplied == 0 || !t.isManaged(object, component) {
				// fetch object (if existing)
				existingObject, err := t.readObject(ctx, item)
				if err != nil {
					return false, errors.Wrapf(err, "error reading object %s", item)
				}

				setLabel(object, t.labelKeyOwnerId, strings.Replace(ownerId, "/", "_", -1))
				setAnnotation(object, t.annotationKeyOwnerId, ownerId)
				setAnnotation(object, t.annotationKeyDigest, item.Digest)

				if existingObject == nil {
					if err := t.createObject(ctx, object); err != nil {
						return false, errors.Wrapf(err, "error creating object %s", item)
					}
					item.Phase = PhaseCreating
					item.Status = kstatus.InProgressStatus.String()
					numUnready++
				} else if existingObject.GetAnnotations()[t.annotationKeyDigest] != item.Digest {
					switch updatePolicy {
					case updatePolicyDefault:
						if err := t.updateObject(ctx, object, existingObject); err != nil {
							return false, errors.Wrapf(err, "error creating object %s", item)
						}
					case updatePolicyRecreate:
						if err := t.deleteObject(ctx, object, existingObject); err != nil {
							return false, errors.Wrapf(err, "error deleting (while recreating) object %s", item)
						}
					default:
						panic("this cannot happen")
					}
					item.Phase = PhaseUpdating
					item.Status = kstatus.InProgressStatus.String()
					numUnready++
				} else {
					res, err := computeStatus(existingObject)
					if err != nil {
						return false, errors.Wrapf(err, "error checking status of object %s", item)
					}
					if res.Status == kstatus.CurrentStatus {
						item.Phase = PhaseReady
					} else {
						numUnready++
					}
					item.Status = res.Status.String()
				}
			} else {
				numUnready++
			}
		}

		// note: after this point, when numUnready is zero, then this and all previous objects are either in PhaseReady or PhaseCompleted

		// if this is the last object of an order, then
		// - if everything so far is ready, trigger due completions and trigger another reconcile if any completion was triggered
		// - otherwise trigger another reconcile
		if k == len(objects)-1 || getOrder(objects[k+1]) > order {
			if numUnready == 0 {
				numPurged := 0
				for j := 0; j <= k; j++ {
					_object := objects[j]
					_item := mustGetItem(status.Inventory, _object)
					_purgeOrder := getPurgeOrder(_object)
					if (k == len(objects)-1 && _purgeOrder < math.MaxInt || _purgeOrder <= order) && _item.Phase != PhaseCompleted {
						_item.Phase = PhaseScheduledForCompletion
						numPurged++
					}
				}
				if numPurged > 0 {
					return false, nil
				}
			} else {
				return false, nil
			}
		}
	}

	return numUnready == 0, nil
}

func (t *reconcileTarget[T]) Delete(ctx context.Context, component T) (bool, error) {
	status := component.GetStatus()

	// count instances of managed types
	numManaged := 0
	for _, item := range status.Inventory {
		if t.isManaged(item, component) {
			numManaged++
		}
	}

	// delete objects and maintain inventory
	var inventory []*InventoryItem
	for _, item := range status.Inventory {
		// fetch object (if existing)
		existingObject, err := t.readObject(ctx, item)
		if err != nil {
			return false, errors.Wrapf(err, "error reading object %s", item)
		}

		// if object is gone, we can remove it from inventory
		if existingObject == nil && item.Phase == PhaseDeleting {
			continue
		}

		if numManaged == 0 || t.isManaged(item, component) {
			// delete the object
			// note: here is a theoretical risk that we delete an existing (foreign) object, because informers are not yet synced
			// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
			if err := t.deleteObject(ctx, item, existingObject); err != nil {
				return false, errors.Wrapf(err, "error deleting object %s", item)
			}
			item.Phase = PhaseDeleting
			item.Status = kstatus.TerminatingStatus.String()
		}
		inventory = append(inventory, item)
	}
	status.Inventory = inventory

	return len(status.Inventory) == 0, nil
}

func (t *reconcileTarget[T]) IsDeletionAllowed(ctx context.Context, component T) (bool, string, error) {
	status := component.GetStatus()

	for _, item := range status.Inventory {
		switch {
		case isCrd(item):
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := t.client.Get(ctx, apitypes.NamespacedName{Name: item.GetName()}, crd); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				} else {
					return false, "", errors.Wrapf(err, "error retrieving crd %s", item.GetName())
				}
			}
			used, err := t.isCrdUsed(ctx, crd, true)
			if err != nil {
				return false, "", errors.Wrapf(err, "error checking usage of crd %s", item.GetName())
			}
			if used {
				return false, fmt.Sprintf("crd %s is still in use (instances exist)", item.GetName()), nil
			}
		case isApiService(item):
			apiService := &apiregistrationv1.APIService{}
			if err := t.client.Get(ctx, apitypes.NamespacedName{Name: item.GetName()}, apiService); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				} else {
					return false, "", errors.Wrapf(err, "error retrieving api service %s", item.GetName())
				}
			}
			used, err := t.isApiServiceUsed(ctx, apiService, true)
			if err != nil {
				return false, "", errors.Wrapf(err, "error checking usage of api service %s", item.GetName())
			}
			if used {
				// TODO: other than with CRDs it is not clear for which types there are instances existing
				// we should improve the error message somehow
				return false, fmt.Sprintf("api service %s is still in use (instances exist)", item.GetName()), nil
			}
		}
	}
	return true, "", nil
}

func (t *reconcileTarget[T]) readObject(ctx context.Context, key types.ObjectKey) (*unstructured.Unstructured, error) {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(key.GetObjectKind().GroupVersionKind())
	if err := t.client.Get(ctx, apitypes.NamespacedName{Namespace: key.GetNamespace(), Name: key.GetName()}, object); err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			object = nil
		} else {
			return nil, err
		}
	}
	return object, nil
}

func (t *reconcileTarget[T]) createObject(ctx context.Context, object client.Object) (err error) {
	defer func() {
		if err == nil {
			t.eventRecorder.Event(object, corev1.EventTypeNormal, objectReasonCreated, "Object successfully created")
		}
	}()
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return err
	}
	object = &unstructured.Unstructured{Object: data}
	if isCrd(object) || isApiService(object) {
		controllerutil.AddFinalizer(object, t.name)
	}
	return t.client.Create(ctx, object)
}

func (t *reconcileTarget[T]) updateObject(ctx context.Context, object client.Object, existingObject *unstructured.Unstructured) (err error) {
	defer func() {
		if err == nil {
			t.eventRecorder.Event(object, corev1.EventTypeNormal, objectReasonUpdated, "Object successfully updated")
		} else {
			t.eventRecorder.Eventf(existingObject, corev1.EventTypeWarning, objectReasonUpdateError, "Error updating object: %s", err)
		}
	}()
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return err
	}
	object = &unstructured.Unstructured{Object: data}
	if isCrd(object) || isApiService(object) {
		controllerutil.AddFinalizer(object, t.name)
	}
	object.SetResourceVersion((existingObject.GetResourceVersion()))
	return t.client.Update(ctx, object)
}

func (t *reconcileTarget[T]) deleteObject(ctx context.Context, key types.ObjectKey, existingObject *unstructured.Unstructured) (err error) {
	defer func() {
		if existingObject == nil {
			return
		}
		if err == nil {
			t.eventRecorder.Event(existingObject, corev1.EventTypeNormal, objectReasonDeleted, "Object successfully deleted")
		} else {
			t.eventRecorder.Eventf(existingObject, corev1.EventTypeWarning, objectReasonDeleteError, "Error deleting object: %s", err)
		}
	}()
	log := log.FromContext(ctx)

	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(key.GetObjectKind().GroupVersionKind())
	object.SetNamespace(key.GetNamespace())
	object.SetName(key.GetName())
	deleteOptions := &client.DeleteOptions{PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationBackground}[0]}
	if existingObject != nil {
		deleteOptions.Preconditions = &metav1.Preconditions{
			ResourceVersion: &[]string{existingObject.GetResourceVersion()}[0],
		}
	}
	if err := t.client.Delete(ctx, object, deleteOptions); err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	switch {
	case isCrd(key):
		for i := 1; i <= 2; i++ {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := t.client.Get(ctx, apitypes.NamespacedName{Name: key.GetName()}, crd); err != nil {
				return client.IgnoreNotFound(err)
			}
			used, err := t.isCrdUsed(ctx, crd, false)
			if err != nil {
				return err
			}
			if used {
				return fmt.Errorf("error deleting custom resource definition %s, existing instances found", types.ObjectKeyToString(key))
			}
			if ok := controllerutil.RemoveFinalizer(crd, t.name); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the API server); this is why we retry once
				if err := t.client.Update(ctx, crd); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating CustomResourcedefinition (409 conflict); doing one retry", "name", t.name, "error", err.Error())
						continue
					}
					return err
				}
			}
			break
		}
	case isApiService(key):
		for i := 1; i <= 2; i++ {
			apiService := &apiregistrationv1.APIService{}
			if err := t.client.Get(ctx, apitypes.NamespacedName{Name: key.GetName()}, apiService); err != nil {
				return client.IgnoreNotFound(err)
			}
			used, err := t.isApiServiceUsed(ctx, apiService, false)
			if err != nil {
				return err
			}
			if used {
				return fmt.Errorf("error deleting api service %s, existing instances found", types.ObjectKeyToString(key))
			}
			if ok := controllerutil.RemoveFinalizer(apiService, t.name); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the API server); this is why we retry once
				if err := t.client.Update(ctx, apiService); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating APIService (409 conflict); doing one retry", "name", t.name, "error", err.Error())
						continue
					}
					return err
				}
			}
			break
		}
	}
	return nil
}

func (t *reconcileTarget[T]) isCrdUsed(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, onlyForeign bool) (bool, error) {
	gvk := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Versions[0].Name,
		Kind:    crd.Spec.Names.Kind,
	}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	labelSelector := labels.Everything()
	if onlyForeign {
		labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + "!=" + crd.Labels[t.labelKeyOwnerId])
	}
	if err := t.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
		return false, err
	}
	return len(list.Items) > 0, nil
}

func (t *reconcileTarget[T]) isApiServiceUsed(ctx context.Context, apiService *apiregistrationv1.APIService, onlyForeign bool) (bool, error) {
	gv := schema.GroupVersion{Group: apiService.Spec.Group, Version: apiService.Spec.Version}
	resList, err := t.discoveryClient.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		return false, err
	}
	var kinds []string
	for _, res := range resList.APIResources {
		if !slices.Contains(kinds, res.Kind) {
			kinds = append(kinds, res.Kind)
		}
	}
	labelSelector := labels.Everything()
	if onlyForeign {
		labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + "!=" + apiService.Labels[t.labelKeyOwnerId])
	}
	for _, kind := range kinds {
		gvk := schema.GroupVersionKind{
			Group:   apiService.Spec.Group,
			Version: apiService.Spec.Version,
			Kind:    kind,
		}
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := t.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
			return false, err
		}
		if len(list.Items) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (t *reconcileTarget[T]) isManaged(key types.ObjectKey, component T) bool {
	status := component.GetStatus()
	gvk := key.GetObjectKind().GroupVersionKind()
	for _, item := range status.Inventory {
		for _, t := range item.ManagedTypes {
			if (t.Group == "*" || t.Group == gvk.Group) && (t.Version == "*" || t.Version == gvk.Version) && (t.Kind == "*" || t.Kind == gvk.Kind) {
				return true
			}
		}
	}
	return false
}

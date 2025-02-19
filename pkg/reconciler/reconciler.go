/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sap/go-generics/sets"
	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/status"
	"github.com/sap/component-operator-runtime/pkg/types"
)

const (
	objectReasonCreated     = "Created"
	objectReasonUpdated     = "Updated"
	objectReasonUpdateError = "UpdateError"
	objectReasonDeleted     = "Deleted"
	objectReasonDeleteError = "DeleteError"
)

const (
	scopeUnknown = iota
	scopeNamespaced
	scopeCluster
)

const (
	minOrder = math.MinInt16
	maxOrder = math.MaxInt16
)

const (
	forceReapplyPeriod = 60 * time.Minute
)

var adoptionPolicyByAnnotation = map[string]AdoptionPolicy{
	types.AdoptionPolicyNever:     AdoptionPolicyNever,
	types.AdoptionPolicyIfUnowned: AdoptionPolicyIfUnowned,
	types.AdoptionPolicyAlways:    AdoptionPolicyAlways,
}

var reconcilePolicyByAnnotation = map[string]ReconcilePolicy{
	types.ReconcilePolicyOnObjectChange:            ReconcilePolicyOnObjectChange,
	types.ReconcilePolicyOnObjectOrComponentChange: ReconcilePolicyOnObjectOrComponentChange,
	types.ReconcilePolicyOnce:                      ReconcilePolicyOnce,
}

var updatePolicyByAnnotation = map[string]UpdatePolicy{
	types.UpdatePolicyRecreate:    UpdatePolicyRecreate,
	types.UpdatePolicyReplace:     UpdatePolicyReplace,
	types.UpdatePolicySsaMerge:    UpdatePolicySsaMerge,
	types.UpdatePolicySsaOverride: UpdatePolicySsaOverride,
}

var deletePolicyByAnnotation = map[string]DeletePolicy{
	types.DeletePolicyDelete:         DeletePolicyDelete,
	types.DeletePolicyOrphan:         DeletePolicyOrphan,
	types.DeletePolicyOrphanOnApply:  DeletePolicyOrphanOnApply,
	types.DeletePolicyOrphanOnDelete: DeletePolicyOrphanOnDelete,
}

// ReconcilerOptions are creation options for a Reconciler.
type ReconcilerOptions struct {
	// Which field manager to use in API calls.
	// If unspecified, the reconciler name is used.
	FieldOwner *string
	// Which finalizer to use.
	// If unspecified, the reconciler name is used.
	Finalizer *string
	// How to react if a dependent object exists but has no or a different owner.
	// If unspecified, AdoptionPolicyIfUnowned is assumed.
	// Can be overridden by annotation on object level.
	AdoptionPolicy *AdoptionPolicy
	// How to perform updates to dependent objects.
	// If unspecified, UpdatePolicyReplace is assumed.
	// Can be overridden by annotation on object level.
	UpdatePolicy *UpdatePolicy
	// How to perform deletion of dependent objects.
	// If unspecified, DeletePolicyDelete is assumed.
	// Can be overridden by annotation on object level.
	DeletePolicy *DeletePolicy
	// Whether namespaces are auto-created if missing.
	// If unspecified, MissingNamespacesPolicyCreate is assumed.
	MissingNamespacesPolicy *MissingNamespacesPolicy
	// Additional managed types. Instances of these types are handled differently during
	// apply and delete; foreign instances of these types will block deletion of the component;
	// a typical example of such additional managed types are CRDs which are implicitly created
	// by the workloads of the component, but not part of the manifests.
	AdditionalManagedTypes []TypeInfo
	// How to analyze the state of the dependent objects.
	// If unspecified, an optimized kstatus based implementation is used.
	StatusAnalyzer status.StatusAnalyzer
	// Prometheus metrics to be populated by the reconciler.
	Metrics ReconcilerMetrics
}

// ReconcilerMetrics defines metrics that the reconciler can populate.
// Metrics specified as nil will be ignored.
type ReconcilerMetrics struct {
	ReadCounter   prometheus.Counter
	CreateCounter prometheus.Counter
	UpdateCounter prometheus.Counter
	DeleteCounter prometheus.Counter
}

// Reconciler manages specified objects in the given target cluster.
type Reconciler struct {
	fieldOwner                   string
	finalizer                    string
	client                       cluster.Client
	statusAnalyzer               status.StatusAnalyzer
	metrics                      ReconcilerMetrics
	adoptionPolicy               AdoptionPolicy
	reconcilePolicy              ReconcilePolicy
	updatePolicy                 UpdatePolicy
	deletePolicy                 DeletePolicy
	missingNamespacesPolicy      MissingNamespacesPolicy
	additionalManagedTypes       []TypeInfo
	labelKeyOwnerId              string
	annotationKeyOwnerId         string
	annotationKeyDigest          string
	annotationKeyAdoptionPolicy  string
	annotationKeyReconcilePolicy string
	annotationKeyUpdatePolicy    string
	annotationKeyDeletePolicy    string
	annotationKeyApplyOrder      string
	annotationKeyPurgeOrder      string
	annotationKeyDeleteOrder     string
}

// Create new reconciler.
// The passed name should be fully qualified; by default it will be used as field owner and finalizer.
// The passed client's scheme must recognize at least the core group (v1) and apiextensions.k8s.io/v1 and apiregistration.k8s.io/v1.
func NewReconciler(name string, clnt cluster.Client, options ReconcilerOptions) *Reconciler {
	// TOOD: validate options
	if options.FieldOwner == nil {
		options.FieldOwner = &name
	}
	if options.Finalizer == nil {
		options.Finalizer = &name
	}
	if options.AdoptionPolicy == nil {
		options.AdoptionPolicy = ref(AdoptionPolicyIfUnowned)
	}
	if options.UpdatePolicy == nil {
		options.UpdatePolicy = ref(UpdatePolicyReplace)
	}
	if options.DeletePolicy == nil {
		options.DeletePolicy = ref(DeletePolicyDelete)
	}
	if options.MissingNamespacesPolicy == nil {
		options.MissingNamespacesPolicy = ref(MissingNamespacesPolicyCreate)
	}
	if options.StatusAnalyzer == nil {
		options.StatusAnalyzer = status.NewStatusAnalyzer(name)
	}

	return &Reconciler{
		fieldOwner:                   *options.FieldOwner,
		finalizer:                    *options.Finalizer,
		client:                       clnt,
		statusAnalyzer:               options.StatusAnalyzer,
		metrics:                      options.Metrics,
		adoptionPolicy:               *options.AdoptionPolicy,
		reconcilePolicy:              ReconcilePolicyOnObjectChange,
		updatePolicy:                 *options.UpdatePolicy,
		deletePolicy:                 *options.DeletePolicy,
		missingNamespacesPolicy:      *options.MissingNamespacesPolicy,
		additionalManagedTypes:       options.AdditionalManagedTypes,
		labelKeyOwnerId:              name + "/" + types.LabelKeySuffixOwnerId,
		annotationKeyOwnerId:         name + "/" + types.AnnotationKeySuffixOwnerId,
		annotationKeyDigest:          name + "/" + types.AnnotationKeySuffixDigest,
		annotationKeyAdoptionPolicy:  name + "/" + types.AnnotationKeySuffixAdoptionPolicy,
		annotationKeyReconcilePolicy: name + "/" + types.AnnotationKeySuffixReconcilePolicy,
		annotationKeyUpdatePolicy:    name + "/" + types.AnnotationKeySuffixUpdatePolicy,
		annotationKeyDeletePolicy:    name + "/" + types.AnnotationKeySuffixDeletePolicy,
		annotationKeyApplyOrder:      name + "/" + types.AnnotationKeySuffixApplyOrder,
		annotationKeyPurgeOrder:      name + "/" + types.AnnotationKeySuffixPurgeOrder,
		annotationKeyDeleteOrder:     name + "/" + types.AnnotationKeySuffixDeleteOrder,
	}
}

// Apply given object manifests to the target cluster and maintain inventory. That means:
//   - non-existent objects will be created
//   - existing objects will be updated if there is a drift (see below)
//   - redundant objects will be removed.
//
// Existing objects will only be updated or deleted if the owner id check is successful; that means:
//   - the object's owner id matches the specified ownerId or
//   - the object's owner id does not match the specified ownerId, and the effective adoption policy is AdoptionPolicyAlways or
//   - the object has no or empty owner id set, and the effective adoption policy is AdoptionPolicyAlways or AdoptionPolicyIfUnowned.
//
// Objects which are instances of namespaced types will be placed into the namespace passed to Apply(), if they have no namespace defined in their manifest.
// An update of an existing object will be performed if it is considered to be out of sync; that means:
//   - the object's manifest has changed, and the effective reconcile policy is ReconcilePolicyOnObjectChange or ReconcilePolicyOnObjectOrComponentChange or
//   - the specified component revision has changed and the effective reconcile policy is ReconcilePolicyOnObjectOrComponentChange or
//   - periodically after forceReapplyPeriod.
//
// The update itself will be done as follows:
//   - if the effective update policy is UpdatePolicyReplace, a http PUT request will be sent to the Kubernetes API
//   - if the effective update policy is UpdatePolicySsaMerge or UpdatePolicySsaOverride, a server-side-apply http PATCH request will be sent;
//     while UpdatePolicySsaMerge just implements the Kubernetes standard behavior (leaving foreign non-conflicting fields untouched), UpdatePolicySsaOverride
//     will re-claim (and therefore potentially drop) fields owned by certain field managers, such as kubectl and helm
//   - if the effective update policy is UpdatePolicyRecreate, the object will be deleted and recreated.
//
// Objects will be applied and deleted in waves, according to their apply/delete order. Objects which specify a purge order will be deleted from the cluster at the
// end of the wave specified as purge order; other than redundant objects, a purged object will remain as Completed in the inventory;
// and it might be re-applied/re-purged in case it runs out of sync. Within a wave, objects are processed following a certain internal order;
// in particular, instances of types which are part of the wave are processed only if all other objects in that wave have a ready state.
//
// Redundant objects will be removed; that means, a http DELETE request will be sent to the Kubernetes API.
//
// This method will change the passed inventory (add or remove elements, change elements). If Apply() returns true, then all objects are successfully reconciled;
// otherwise, if it returns false, the caller should re-call it periodically, until it returns true. In any case, the passed inventory should match the state of the
// inventory after the previous invocation of Apply(); usually, the caller saves the inventory after calling Apply(), and loads it before calling Apply().
// The namespace and ownerId arguments should not be changed across subsequent invocations of Apply(); the componentRevision should be incremented only.
func (r *Reconciler) Apply(ctx context.Context, inventory *[]*InventoryItem, objects []client.Object, namespace string, ownerId string, componentRevision int64) (bool, error) {
	var err error
	log := log.FromContext(ctx)

	hashedOwnerId := sha256base32([]byte(ownerId))

	// perform some initial validation
	for _, object := range objects {
		if object.GetGenerateName() != "" {
			// TODO: the object key string representation below will probably be incomplete because of missing metadata.name
			return false, fmt.Errorf("object %s specifies metadata.generateName (but dependent objects are not allowed to do so)", types.ObjectKeyToString(object))
		}
	}

	// normalize objects; that means:
	// - check that unstructured objects have valid type information set, and convert them to their concrete type if known to the scheme
	// - check that non-unstructured types are known to the scheme, and validate/set their type information
	objects, err = normalizeObjects(objects, r.client.Scheme())
	if err != nil {
		return false, errors.Wrap(err, "error normalizing objects")
	}

	// perform cleanup on object manifests
	for _, object := range objects {
		removeLabel(object, r.labelKeyOwnerId)
		removeAnnotation(object, r.annotationKeyOwnerId)
		removeAnnotation(object, r.annotationKeyDigest)
	}

	// validate type and set namespace for namespaced objects which have no namespace set
	for _, object := range objects {
		// note: due to the normalization done before, every object will now have a valid object kind set
		gvk := object.GetObjectKind().GroupVersionKind()

		// TODO: client now has a method IsObjectNamespaced(); can we use this instead?
		scope := scopeUnknown
		restMapping, err := r.client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err == nil {
			scope = scopeFromRestMapping(restMapping)
		} else if !apimeta.IsNoMatchError(err) {
			return false, errors.Wrapf(err, "error getting rest mapping for object %s", types.ObjectKeyToString(object))
		}
		for _, crd := range getCrds(objects) {
			if crd.Spec.Group == gvk.Group && crd.Spec.Names.Kind == gvk.Kind {
				// TODO: validate that scope obtained from crd matches scope from rest mapping (if one was found there)
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
		if object.GetNamespace() != "" && scope == scopeCluster {
			object.SetNamespace("")
		}
	}
	// note: after this point there still can be objects in the list which
	// - have a namespace set although they are not namespaced
	// - do not have a namespace set although they are namespaced
	// which exactly happens if
	// 1. the object is incorrectly specified and
	// 2. calling RESTMapping() above returned a NoMatchError (i.e. the type is currently not known to the api server) and
	// 3. the type belongs to a (new) api service which is part of the inventory
	// such entries can cause trouble, e.g. because the duplicate check, or InventoryItem.Match() might not work reliably ...
	// TODO: should we allow at all that api services and according instances are deployed together?

	// check that there are no duplicate objects
	objectKeys := sets.New[string]()
	for _, object := range objects {
		objectKey := types.ObjectKeyToString(object)
		if sets.Contains(objectKeys, objectKey) {
			return false, fmt.Errorf("duplicate object %s", objectKey)
		}
		sets.Add(objectKeys, objectKey)
	}

	// validate annotations
	for _, object := range objects {
		if _, err := r.getAdoptionPolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getReconcilePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getUpdatePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getDeletePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getApplyOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getPurgeOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := r.getDeleteOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		// TODO: should status-hint be validated here as well?
	}

	// define getter functions for later usage
	getAdoptionPolicy := func(object client.Object) AdoptionPolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getAdoptionPolicy(object))
	}
	getReconcilePolicy := func(object client.Object) ReconcilePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getReconcilePolicy(object))
	}
	getUpdatePolicy := func(object client.Object) UpdatePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getUpdatePolicy(object))
	}
	getDeletePolicy := func(object client.Object) DeletePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getDeletePolicy(object))
	}
	getApplyOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getApplyOrder(object))
	}
	getPurgeOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getPurgeOrder(object))
	}
	getDeleteOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(r.getDeleteOrder(object))
	}

	// perform further validations of object set
	for _, object := range objects {
		switch {
		case isNamespace(object):
			if getPurgeOrder(object) <= maxOrder {
				return false, errors.Wrapf(fmt.Errorf("namespaces must not define a purge order"), "error validating object %s", types.ObjectKeyToString(object))
			}
		case isCrd(object):
			if getPurgeOrder(object) <= maxOrder {
				return false, errors.Wrapf(fmt.Errorf("custom resource definitions must not define a purge order"), "error validating object %s", types.ObjectKeyToString(object))
			}
		case isApiService(object):
			if getPurgeOrder(object) <= maxOrder {
				return false, errors.Wrapf(fmt.Errorf("api services must not define a purge order"), "error validating object %s", types.ObjectKeyToString(object))
			}
		}
	}

	// prepare (add/update) new inventory with target objects
	// TODO: review this; it would be cleaner to use a DeepCopy method for a []*InventoryItem type (if there would be such a type)
	newInventory := slices.Collect(*inventory, func(item *InventoryItem) *InventoryItem { return item.DeepCopy() })
	numAdded := 0
	for _, object := range objects {
		// retrieve inventory item belonging to this object (if existing)
		item := getItem(newInventory, object)

		// calculate object digest
		// note: if the effective reconcile policy of an object changes, it will always be reconciled at least one more time;
		// this is in particular the case if the policy changes from or to ReconcilePolicyOnce.
		digest, err := calculateObjectDigest(object, componentRevision, getReconcilePolicy(object))
		if err != nil {
			return false, errors.Wrapf(err, "error calculating digest for object %s", types.ObjectKeyToString(object))
		}

		// if item was not found, append an empty item
		if item == nil {
			// TODO: should the owner id check happen always (not only if the object is unknown to the inventory)?
			// TODO: since deletion handling now happens late, it can happen that, when an object is moved from its previous compoment into a new one,
			// and the previous one gets deleted at the same time, applying the new one runs stuck because of the owner id check;
			// so we might add some logic to skip the owner id check in that particular case

			// fetch object (if existing)
			existingObject, err := r.readObject(ctx, object)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", types.ObjectKeyToString(object))
			}
			// check ownership
			// note: failing already here in case of a conflict prevents problems during apply and, in particular, during deletion
			if existingObject != nil {
				adoptionPolicy := getAdoptionPolicy(object)
				existingOwnerId := existingObject.GetLabels()[r.labelKeyOwnerId]
				if existingOwnerId == "" {
					if adoptionPolicy != AdoptionPolicyIfUnowned && adoptionPolicy != AdoptionPolicyAlways {
						return false, fmt.Errorf("found existing object %s without owner", types.ObjectKeyToString(object))
					}
				} else if existingOwnerId != hashedOwnerId {
					if adoptionPolicy != AdoptionPolicyAlways {
						return false, fmt.Errorf("owner conflict; object %s is owned by %s", types.ObjectKeyToString(object), existingObject.GetAnnotations()[r.annotationKeyOwnerId])
					}
				}
			}
			newInventory = append(newInventory, &InventoryItem{})
			item = newInventory[len(newInventory)-1]
			numAdded++
		}

		// update item
		gvk := object.GetObjectKind().GroupVersionKind()
		item.Group = gvk.Group
		item.Version = gvk.Version
		item.Kind = gvk.Kind
		item.Namespace = object.GetNamespace()
		item.Name = object.GetName()
		item.AdoptionPolicy = getAdoptionPolicy(object)
		item.ReconcilePolicy = getReconcilePolicy(object)
		item.UpdatePolicy = getUpdatePolicy(object)
		item.DeletePolicy = getDeletePolicy(object)
		item.ApplyOrder = getApplyOrder(object)
		item.DeleteOrder = getDeleteOrder(object)
		item.ManagedTypes = getManagedTypes(object)
		if digest != item.Digest {
			item.Digest = digest
			item.Phase = PhaseScheduledForApplication
			item.Status = status.InProgressStatus
		}
	}

	// mark obsolete items (clear digest) in new inventory
	for _, item := range newInventory {
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
			item.Status = status.TerminatingStatus
		}
	}

	// validate new inventory:
	// - check that all managed instances have apply-order greater than or equal to the according managed type
	// - check that all managed instances have delete-order less than or equal to the according managed type
	// - check that no managed types are about to be deleted (empty digest) unless all related managed instances are as well
	// - check that all contained objects have apply-order greater than or equal to the according namespace
	// - check that all contained objects have delete-order less than or equal to the according namespace
	// - check that no namespaces are about to be deleted (empty digest) unless all contained objects are as well
	for _, item := range newInventory {
		if isCrd(item) || isApiService(item) {
			for _, _item := range newInventory {
				if isManagedByTypeVersions(item.ManagedTypes, _item) {
					if _item.ApplyOrder < item.ApplyOrder {
						return false, fmt.Errorf("error valdidating object set (%s): managed instance must not have an apply order lesser than the one of its type", _item)
					}
					if _item.DeleteOrder > item.DeleteOrder {
						return false, fmt.Errorf("error valdidating object set (%s): managed instance must not have a delete order greater than the one of its type", _item)
					}
					if _item.Digest != "" && item.Digest == "" {
						return false, fmt.Errorf("error valdidating object set (%s): managed instance is not being deleted, but the managing type is", _item)
					}
				}
			}
		}
		if isNamespace(item) {
			for _, _item := range newInventory {
				if _item.Namespace == item.Name {
					if _item.ApplyOrder < item.ApplyOrder {
						return false, fmt.Errorf("error valdidating object set (%s): namespaced object must not have an apply order lesser than the one of its namespace", _item)
					}
					if _item.DeleteOrder > item.DeleteOrder {
						return false, fmt.Errorf("error valdidating object set (%s): namespaced object must not have a delete order greater than the one of its namespace", _item)
					}
					if _item.Digest != "" && item.Digest == "" {
						return false, fmt.Errorf("error valdidating object set (%s): namespaced object is not being deleted, but the namespace is", _item)
					}
				}
			}
		}
	}

	// accept new inventory for further processing, put into right order for future deletion
	*inventory = sortObjectsForDelete(newInventory)

	// trigger another reconcile if something was added (to be sure that it is persisted)
	if numAdded > 0 {
		return false, nil
	}

	// note: after this point it is guaranteed that
	// - the in-memory inventory reflects the target state
	// - the persisted inventory at least has the same object keys as the in-memory inventory
	// now it is about to synchronize the cluster state with the inventory

	// note: after this point, it is also guaranteed that objects is contained in the persisted inventory;
	// the inventory therefore consists of two parts:
	// - items which are contained in objects
	//   these items can have one of the following phases:
	//   - PhaseScheduledForApplication
	//   - PhaseCreating
	//   - PhaseUpdating
	//   - PhaseReady
	//   - PhaseScheduledForCompletion
	//   - PhaseCompleting
	//   - PhaseCompleted
	// - items which are not contained in objects
	//   their phase is one of the following:
	//   - PhaseScheduledForDeletion
	//   - PhaseDeleting

	// create missing namespaces
	if r.missingNamespacesPolicy == MissingNamespacesPolicyCreate {
		for _, namespace := range findMissingNamespaces(objects) {
			if err := r.client.Get(ctx, apitypes.NamespacedName{Name: namespace}, &corev1.Namespace{}); err != nil {
				if !apierrors.IsNotFound(err) {
					return false, errors.Wrapf(err, "error reading namespace %s", namespace)
				}
				if err := r.client.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, client.FieldOwner(r.fieldOwner)); err != nil {
					return false, errors.Wrapf(err, "error creating namespace %s", namespace)
				}
			}
		}
	}

	// put objects into right order for applying
	objects = sortObjectsForApply(objects, getApplyOrder)

	// finish due completions
	// note that completions do not honor delete-order or delete-policy
	// however, due to the way how PhaseScheduledForCompletion is set, the affected objects will
	// always be in one and the same apply order
	// in addition deletions are triggered in the canonical deletion order (but not waited for)
	numToBeCompleted := 0
	for _, item := range *inventory {
		if item.Phase == PhaseScheduledForCompletion || item.Phase == PhaseCompleting {
			existingObject, err := r.readObject(ctx, item)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", item)
			}

			switch item.Phase {
			case PhaseScheduledForCompletion:
				if err := r.deleteObject(ctx, item, existingObject, hashedOwnerId); err != nil {
					return false, errors.Wrapf(err, "error deleting object %s", item)
				}
				item.Phase = PhaseCompleting
				item.Status = status.TerminatingStatus
				numToBeCompleted++
			case PhaseCompleting:
				if existingObject == nil {
					item.Phase = PhaseCompleted
					item.Status = ""
				} else {
					// TODO: should we (similar to the delete cases) check deletion timestamp and ownership to void deadlocks if object
					// was recreated by someone else
					numToBeCompleted++
				}
			}
		}
	}

	// trigger another reconcile if any to-be-completed objects are left
	if numToBeCompleted > 0 {
		return false, nil
	}

	// note: after this point, PhaseScheduledForCompletion, PhaseCompleting cannot occur anymore in inventory

	// apply objects and maintain inventory;
	// objects are applied (i.e. created/updated) in waves according to their apply order;
	// that means, only if all objects of a wave are ready or completed, the next wave
	// will be procesed; within each wave, objects which are instances of managed types
	// will be applied after all other objects
	isManaged := func(key types.ObjectKey) bool {
		return isManagedInstance(r.additionalManagedTypes, *inventory, key)
	}
	isLate := func(key types.ObjectKey) bool {
		return isApiService(key)
	}
	isRegular := func(key types.ObjectKey) bool {
		return !isLate(key) && !isManaged(key)
	}
	isUsedNamespace := func(key types.ObjectKey) bool {
		return isNamespace(key) && isNamespaceUsed(*inventory, key.GetName())
	}
	numRegularToBeApplied := 0
	numLateToBeApplied := 0
	numUnready := 0
	for k, object := range objects {
		// retrieve inventory item corresponding to this object
		item := mustGetItem(*inventory, object)

		// retrieve object order
		applyOrder := getApplyOrder(object)

		// within each apply order, objects are deployed to readiness in three sub stages
		// - regular objects (all 'normal' objects)
		// - late objects (currently, this is only APIService objects)
		// - instances of managed types (that is instances of types which are added in this component as CRD or through an APIService)
		// within each of these sub groups, the static ordering defined in sortObjectsForApply() is effective

		// if this is the first object of an order, then
		// count instances of managed types in this order which are about to be applied
		if k == 0 || getApplyOrder(objects[k-1]) < applyOrder {
			log.V(2).Info("begin of apply wave", "order", applyOrder)
			numRegularToBeApplied = 0
			numLateToBeApplied = 0
			for j := k; j < len(objects) && getApplyOrder(objects[j]) == applyOrder; j++ {
				_object := objects[j]
				_item := mustGetItem(*inventory, _object)
				if _item.Phase != PhaseReady && _item.Phase != PhaseCompleted {
					// that means: _item.Phase is one of PhaseScheduledForApplication, PhaseCreating, PhaseUpdating
					if isRegular(_object) {
						numRegularToBeApplied++
					} // (same as) else (because isRegular() and isLate() are mutually exclusive)
					if isLate(_object) {
						numLateToBeApplied++
					}
				}
			}
		}

		// for non-completed objects, compute and update status, and apply (create or update) the object if necessary
		if item.Phase != PhaseCompleted {
			// reconcile all instances of managed types after remaining objects
			// this ensures that everything is running what is needed for the reconciliation of the managed instances,
			// such as webhook servers, api servers, ...
			// note: here, phase is one of PhaseScheduledForApplication, PhaseCreating, PhaseUpdating, PhaseReady
			if isRegular(object) || isLate(object) && numRegularToBeApplied == 0 || isManaged(object) && numRegularToBeApplied == 0 && numLateToBeApplied == 0 {
				// fetch object (if existing)
				existingObject, err := r.readObject(ctx, item)
				if err != nil {
					return false, errors.Wrapf(err, "error reading object %s", item)
				}

				setLabel(object, r.labelKeyOwnerId, hashedOwnerId)
				setAnnotation(object, r.annotationKeyOwnerId, ownerId)
				setAnnotation(object, r.annotationKeyDigest, item.Digest)

				updatePolicy := getUpdatePolicy(object)
				now := time.Now()
				if existingObject == nil {
					if err := r.createObject(ctx, object, nil, updatePolicy); err != nil {
						return false, errors.Wrapf(err, "error creating object %s", item)
					}
					item.Phase = PhaseCreating
					item.Status = status.InProgressStatus
					item.LastAppliedAt = &metav1.Time{Time: now}
					numUnready++
				} else if existingObject.GetDeletionTimestamp().IsZero() &&
					// TODO: make force-reconcile period (60 minutes as of now) configurable
					(existingObject.GetAnnotations()[r.annotationKeyDigest] != item.Digest || item.LastAppliedAt == nil || item.LastAppliedAt.Time.Before(now.Add(-forceReapplyPeriod))) {
					switch updatePolicy {
					case UpdatePolicyRecreate:
						if err := r.deleteObject(ctx, object, existingObject, hashedOwnerId); err != nil {
							return false, errors.Wrapf(err, "error deleting (while recreating) object %s", item)
						}
					default:
						// TODO: perform an additional owner id check
						if err := r.updateObject(ctx, object, existingObject, nil, updatePolicy); err != nil {
							return false, errors.Wrapf(err, "error updating object %s", item)
						}
					}
					item.Phase = PhaseUpdating
					item.Status = status.InProgressStatus
					item.LastAppliedAt = &metav1.Time{Time: now}
					numUnready++
				} else {
					existingStatus, err := r.statusAnalyzer.ComputeStatus(existingObject)
					if err != nil {
						return false, errors.Wrapf(err, "error checking status of object %s", item)
					}
					if existingObject.GetDeletionTimestamp().IsZero() && existingStatus == status.CurrentStatus {
						item.Phase = PhaseReady
					} else {
						numUnready++
					}
					item.Status = existingStatus
				}
			} else {
				numUnready++
			}
		}

		// note: after this point, when numUnready is zero, then this and all previous objects are either in PhaseReady or PhaseCompleted

		// if this is the last object of an order, then
		// - if everything so far is ready, trigger due completions and trigger another reconcile if any completion was triggered
		// - otherwise trigger another reconcile
		if k == len(objects)-1 || getApplyOrder(objects[k+1]) > applyOrder {
			log.V(2).Info("end of apply wave", "order", applyOrder)
			if numUnready == 0 {
				numPurged := 0
				for j := 0; j <= k; j++ {
					_object := objects[j]
					_item := mustGetItem(*inventory, _object)
					_purgeOrder := getPurgeOrder(_object)
					if (k == len(objects)-1 && _purgeOrder <= maxOrder || _purgeOrder <= applyOrder) && _item.Phase != PhaseCompleted {
						_item.Phase = PhaseScheduledForCompletion
						numPurged++
					}
				}
				if numPurged > 0 {
					return false, nil
				}
				// TODO: we should do deletion of redundant objects (in the current apply stage) here
				// maybe it would even make sense to introduce something like a delete-on-apply-policy (Early,Regular,Late)
				// and maybe even a delete-on-apply-order ...
			} else {
				return false, nil
			}
		}
	}

	// trigger another reconcile if any unready objects are left
	if numUnready > 0 {
		return false, nil
	}

	// delete redundant objects and maintain inventory;
	// objects are deleted in waves according to their delete order;
	// that means, only if all redundant objects of a wave are gone , the next
	// wave will be processed; within each wave, objects which are instances of managed
	// types are deleted before all other objects, and namespaces will only be deleted
	// if they are not used by any object in the inventory (note that this may cause deadlocks)
	numManagedToBeDeleted := 0
	numToBeDeleted := 0
	for k, item := range *inventory {
		// if this is the first object of an order, then
		// count instances of managed types in this wave which are about to be deleted
		if k == 0 || (*inventory)[k-1].DeleteOrder < item.DeleteOrder {
			log.V(2).Info("begin of deletion wave", "order", item.DeleteOrder)
			numManagedToBeDeleted = 0
			for j := k; j < len(*inventory) && (*inventory)[j].DeleteOrder == item.DeleteOrder; j++ {
				_item := (*inventory)[j]
				if (_item.Phase == PhaseScheduledForDeletion || _item.Phase == PhaseDeleting) && isManaged(_item) {
					numManagedToBeDeleted++
				}
			}
		}

		if item.Phase == PhaseScheduledForDeletion || item.Phase == PhaseDeleting {
			// fetch object (if existing)
			existingObject, err := r.readObject(ctx, item)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", item)
			}

			// note: the effective deletion policy is always the last known one of the dependent object,
			// that is, the one determined when the object was contained in the manifests the last time;
			// just-in-time changes of the default deletion policy on the component thus have no impact on the
			// deletion policy of redundant objects; dependent objects are orphaned if they have an effective
			// Orphan or OrphanOnApply deletion policy.

			orphan := item.DeletePolicy == DeletePolicyOrphan || item.DeletePolicy == DeletePolicyOrphanOnApply

			switch item.Phase {
			case PhaseScheduledForDeletion:
				// delete namespaces after all contained inventory items
				// delete all instances of managed types before remaining objects; this ensures that no objects are prematurely
				// deleted which are needed for the deletion of the managed instances, such as webhook servers, api servers, ...
				if !isUsedNamespace(item) && (numManagedToBeDeleted == 0 || isManaged(item)) {
					if orphan {
						item.Phase = ""
					} else {
						// note: here is a theoretical risk that we delete an existing foreign object, because informers are not yet synced
						// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
						if err := r.deleteObject(ctx, item, existingObject, hashedOwnerId); err != nil {
							return false, errors.Wrapf(err, "error deleting object %s", item)
						}
						item.Phase = PhaseDeleting
						item.Status = status.TerminatingStatus
						numToBeDeleted++
					}
				} else {
					numToBeDeleted++
				}
			case PhaseDeleting:
				if existingObject == nil {
					// if object is gone, we can remove it from inventory
					item.Phase = ""
				} else if !existingObject.GetDeletionTimestamp().IsZero() {
					// object is still there and deleting, waiting until it goes away
					numToBeDeleted++
				} else if existingObject.GetLabels()[r.labelKeyOwnerId] != hashedOwnerId {
					// object is there but not deleting; if we are not owning it that means that somebody else has
					// recreated it in the meantime; so we consider this as not our problem and remove it from inventory
					log.V(1).Info("orphaning resurrected object (probably it was recreated by someone else)", "key", types.ObjectKeyToString(item))
					item.Phase = ""
				} else {
					// object is there, not deleting, but we own it; that is really strange and should actually not happen
					return false, fmt.Errorf("object %s was already deleted but has no deletion timestamp", types.ObjectKeyToString(item))
				}
			default:
				// note: any other phase value would indicate a severe code problem, so we want to see the panic in that case
				panic("this cannot happen")
			}
		}

		// trigger another reconcile if this is the last object of the wave, and some deletions are not yet finished
		if k == len(*inventory)-1 || (*inventory)[k+1].DeleteOrder > item.DeleteOrder {
			log.V(2).Info("end of deletion wave", "order", item.DeleteOrder)
			if numToBeDeleted > 0 {
				break
			}
		}
	}

	*inventory = slices.Select(*inventory, func(item *InventoryItem) bool { return item.Phase != "" })

	// trigger another reconcile if any to-be-deleted objects are left
	if numToBeDeleted > 0 {
		return false, nil
	}

	return true, nil
}

// Delete objects stored in the inventory from the target cluster and maintain inventory.
// Objects will be deleted in waves, according to their delete order (as stored in the inventory); that means, the deletion of
// objects having a certain delete order will only start if all objects with lower delete order are gone. Within a wave, objects are
// deleted following a certain internal ordering; in particular, if there are instances of types which are part of the wave, then these
// instances will be deleted first; only if all such instances are gone, the remaining objects of the wave will be deleted.
// Objects which have an effective Orphan or OrphanOnDelete deletion policy will not be touched (remain in the cluster),
// but will no longer appear in the inventory.
//
// This method will change the passed inventory (remove elements, change elements). If Delete() returns true, then all objects are gone; otherwise,
// if it returns false, the caller should recall it timely, until it returns true. In any case, the passed inventory should match the state of the
// inventory after the previous invocation of Delete(); usually, the caller saves the inventory after calling Delete(), and loads it before calling Delete().
func (r *Reconciler) Delete(ctx context.Context, inventory *[]*InventoryItem, ownerId string) (bool, error) {
	log := log.FromContext(ctx)

	hashedOwnerId := sha256base32([]byte(ownerId))

	// delete objects and maintain inventory;
	// objects are deleted in waves according to their delete order;
	// that means, only if all objects of a wave are gone, the next wave will be processed;
	// within each wave, objects which are instances of managed types are deleted before all
	// other objects, and namespaces will only be deleted if they are not used by any
	// object in the inventory (note that this may cause deadlocks)
	numManagedToBeDeleted := 0
	numToBeDeleted := 0
	for k, item := range *inventory {
		// if this is the first object of an order, then
		// count instances of managed types in this wave which are about to be deleted
		if k == 0 || (*inventory)[k-1].DeleteOrder < item.DeleteOrder {
			log.V(2).Info("begin of deletion wave", "order", item.DeleteOrder)
			numManagedToBeDeleted = 0
			for j := k; j < len(*inventory) && (*inventory)[j].DeleteOrder == item.DeleteOrder; j++ {
				_item := (*inventory)[j]
				if isManagedInstance(r.additionalManagedTypes, *inventory, _item) {
					numManagedToBeDeleted++
				}
			}
		}

		// fetch object (if existing)
		existingObject, err := r.readObject(ctx, item)
		if err != nil {
			return false, errors.Wrapf(err, "error reading object %s", item)
		}

		orphan := item.DeletePolicy == DeletePolicyOrphan || item.DeletePolicy == DeletePolicyOrphanOnDelete

		switch item.Phase {
		case PhaseDeleting:
			if existingObject == nil {
				// if object is gone, we can remove it from inventory
				item.Phase = ""
			} else if !existingObject.GetDeletionTimestamp().IsZero() {
				// object is still there and deleting, waiting until it goes away
				numToBeDeleted++
			} else if existingObject.GetLabels()[r.labelKeyOwnerId] != hashedOwnerId {
				// object is there but not deleting; if we are not owning it that means that somebody else has
				// recreated it in the meantime; so we consider this as not our problem and remove it from inventory
				log.V(1).Info("orphaning resurrected object (probably it was recreated by someone else)", "key", types.ObjectKeyToString(item))
				item.Phase = ""
			} else {
				// object is there, not deleting, but we own it; that is really strange and should actually not happen
				return false, fmt.Errorf("object %s was already deleted but has no deletion timestamp", types.ObjectKeyToString(item))
			}
		default:
			// delete namespaces after all contained inventory items
			// delete all instances of managed types before remaining objects; this ensures that no objects are prematurely
			// deleted which are needed for the deletion of the managed instances, such as webhook servers, api servers, ...
			if (!isNamespace(item) || !isNamespaceUsed(*inventory, item.Name)) && (numManagedToBeDeleted == 0 || isManagedInstance(r.additionalManagedTypes, *inventory, item)) {
				if orphan {
					item.Phase = ""
				} else {
					// delete the object
					// note: here is a theoretical risk that we delete an existing (foreign) object, because informers are not yet synced
					// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
					if err := r.deleteObject(ctx, item, existingObject, hashedOwnerId); err != nil {
						return false, errors.Wrapf(err, "error deleting object %s", item)
					}
					item.Phase = PhaseDeleting
					item.Status = status.TerminatingStatus
					numToBeDeleted++
				}
			} else {
				numToBeDeleted++
			}
		}

		// trigger another reconcile if this is the last object of the wave, and some deletions are not yet completed
		if k == len(*inventory)-1 || (*inventory)[k+1].DeleteOrder > item.DeleteOrder {
			log.V(2).Info("end of deletion wave", "order", item.DeleteOrder)
			if numToBeDeleted > 0 {
				break
			}
		}
	}

	*inventory = slices.Select(*inventory, func(item *InventoryItem) bool { return item.Phase != "" })

	return len(*inventory) == 0, nil
}

// Check if the object set defined by inventory is ready for deletion; that means: check if the inventory contains
// types (as custom resource definition or from an api service), while there exist instances of these types in the cluster,
// which are not contained in the inventory. There is one exception of this rule: if all objects in the inventory have their
// deletion policy set to Orphan or OrphanOnDelete, then the deletion of the component is immediately allowed.
func (r *Reconciler) IsDeletionAllowed(ctx context.Context, inventory *[]*InventoryItem, ownerId string) (bool, string, error) {
	hashedOwnerId := sha256base32([]byte(ownerId))

	for _, t := range r.additionalManagedTypes {
		gk := schema.GroupKind(t)
		used, err := r.isTypeUsed(ctx, gk, hashedOwnerId, true)
		if err != nil {
			return false, "", errors.Wrapf(err, "error checking usage of type %s", gk)
		}
		if used {
			return false, fmt.Sprintf("type %s is still in use (instances exist)", gk), nil
		}
	}

	if slices.All(*inventory, func(item *InventoryItem) bool {
		return item.DeletePolicy == DeletePolicyOrphan || item.DeletePolicy == DeletePolicyOrphanOnDelete
	}) {
		return true, "", nil
	}

	for _, item := range *inventory {
		switch {
		case isCrd(item):
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := r.client.Get(ctx, apitypes.NamespacedName{Name: item.GetName()}, crd); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				} else {
					return false, "", errors.Wrapf(err, "error retrieving crd %s", item.GetName())
				}
			}
			used, err := r.isCrdUsed(ctx, crd, hashedOwnerId, true)
			if err != nil {
				return false, "", errors.Wrapf(err, "error checking usage of crd %s", item.GetName())
			}
			if used {
				return false, fmt.Sprintf("crd %s is still in use (instances exist)", item.GetName()), nil
			}
		case isApiService(item):
			apiService := &apiregistrationv1.APIService{}
			if err := r.client.Get(ctx, apitypes.NamespacedName{Name: item.GetName()}, apiService); err != nil {
				if apierrors.IsNotFound(err) {
					continue
				} else {
					return false, "", errors.Wrapf(err, "error retrieving api service %s", item.GetName())
				}
			}
			used, err := r.isApiServiceUsed(ctx, apiService, hashedOwnerId, true)
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

// reaad object and return as unstructured
func (r *Reconciler) readObject(ctx context.Context, key types.ObjectKey) (*unstructured.Unstructured, error) {
	if counter := r.metrics.ReadCounter; counter != nil {
		counter.Inc()
	}

	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(key.GetObjectKind().GroupVersionKind())
	if err := r.client.Get(ctx, apitypes.NamespacedName{Namespace: key.GetNamespace(), Name: key.GetName()}, object); err != nil {
		if apimeta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			object = nil
		} else {
			return nil, err
		}
	}
	return object, nil
}

// create object; object may be a concrete type or unstructured; in any case, type meta must be populated;
// createdObject is optional; if non-nil, it will be populated with the created object; the same variable can be supplied as object and createObject;
// if object is a crd or an api services, the reconciler's name will be added as finalizer
func (r *Reconciler) createObject(ctx context.Context, object client.Object, createdObject any, updatePolicy UpdatePolicy) (err error) {
	if counter := r.metrics.CreateCounter; counter != nil {
		counter.Inc()
	}

	defer func() {
		if err == nil && createdObject != nil {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.(*unstructured.Unstructured).Object, createdObject)
		}
		if err == nil {
			r.client.EventRecorder().Event(object, corev1.EventTypeNormal, objectReasonCreated, "Object successfully created")
		}
	}()

	// log := log.FromContext(ctx).WithValues("object", types.ObjectKeyToString(object))

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return err
	}
	object = &unstructured.Unstructured{Object: data}
	if isCrd(object) || isApiService(object) {
		controllerutil.AddFinalizer(object, r.finalizer)
	}
	// note: clearing managedFields is anyway required for ssa; but also in the create (post) case it does not harm
	object.SetManagedFields(nil)
	// create the object right from the start with the right managed fields operation (Apply or Update), in order to avoid
	// having to patch the managed fields during future update calls
	switch updatePolicy {
	case UpdatePolicySsaMerge, UpdatePolicySsaOverride:
		// set the target resource version to an impossible value; this will produce a 409 conflict in case the object already exists
		object.SetResourceVersion("1")
		return r.client.Patch(ctx, object, client.Apply, client.FieldOwner(r.fieldOwner))
	default:
		return r.client.Create(ctx, object, client.FieldOwner(r.fieldOwner))
	}
}

// update object; object may be a concrete type or unstructured; in any case, type meta must be populated;
// existingObject is required, and should represent the last-read state of the object; it must not have a deletionTimestamp set;
// updatedObject is optional; if non-nil, it will be populated with the updated object; the same variable can be supplied as object and updatedObject;
// if object is a crd or an api services, the reconciler's name will be added as finalizer;
// object may have a resourceVersion; if it does not, the resourceVersion of existingObject will be used for conflict checks during put/patch;
// if updatePolicy equals UpdatePolicyReplace, an update (put) will be performed; finalizers of existingObject will be copied;
// if updatePolicy equals UpdatePolicySsaMerge, a conflict-forcing server-side-apply patch will be performed;
// if updatePolicy equals UpdatePolicySsaOverride, then in addition, a preparation patch request will be performed before doing the conflict-forcing
// server-side-apply patch; this preparation patch will adjust managedFields, reclaiming fields/values previously owned by kubectl or helm
func (r *Reconciler) updateObject(ctx context.Context, object client.Object, existingObject *unstructured.Unstructured, updatedObject any, updatePolicy UpdatePolicy) (err error) {
	if counter := r.metrics.UpdateCounter; counter != nil {
		counter.Inc()
	}

	defer func() {
		if err == nil && updatedObject != nil {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.(*unstructured.Unstructured).Object, updatedObject)
		}
		if err == nil {
			r.client.EventRecorder().Event(object, corev1.EventTypeNormal, objectReasonUpdated, "Object successfully updated")
		} else {
			r.client.EventRecorder().Eventf(existingObject, corev1.EventTypeWarning, objectReasonUpdateError, "Error updating object: %s", err)
		}
	}()

	log := log.FromContext(ctx).WithValues("object", types.ObjectKeyToString(object))

	// TODO: validate (by panic) that existingObject fits to object (i.e. have same object key)
	if !existingObject.GetDeletionTimestamp().IsZero() {
		// note: we must not update objects which are in deletion (e.g. to avoid unintentionally clearing finalizers), so we want to see the panic in that case
		panic("this cannot happen")
	}

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return err
	}
	object = &unstructured.Unstructured{Object: data}
	if isCrd(object) || isApiService(object) {
		controllerutil.AddFinalizer(object, r.finalizer)
	}
	// it is allowed that target object contains a resource version; otherwise, we set the resource version to the one of the existing object,
	// in order to ensure that we do not unintentionally overwrite a state different from the one we have read;
	// note that the api server performs a resource version conflict check not only in case of update (put), but also for ssa (patch)
	if object.GetResourceVersion() == "" {
		object.SetResourceVersion((existingObject.GetResourceVersion()))
	}
	// note: clearing managedFields is anyway required for ssa; but also in the replace (put, update) case it does not harm;
	// because replace will only claim fields which are new or which have changed; the field owner of declared (but unmodified)
	// fields will not be touched
	object.SetManagedFields(nil)
	switch updatePolicy {
	case UpdatePolicySsaMerge, UpdatePolicySsaOverride:
		var replacedFieldManagerPrefixes []string
		if updatePolicy == UpdatePolicySsaOverride {
			// TODO: add ways (per reconciler, per component, per object) to configure the list of field manager (prefixes) which are reclaimed
			replacedFieldManagerPrefixes = []string{"kubectl", "helm"}
		}
		// note: even if replacedFieldManagerPrefixes is empty, replaceFieldManager() will reclaim fields created by us through an Update operation,
		// that is through a create or update call; this may be necessary, if the update policy for the object changed (globally or per-object)
		if managedFields, changed, err := replaceFieldManager(existingObject.GetManagedFields(), replacedFieldManagerPrefixes, r.fieldOwner); err != nil {
			return err
		} else if changed {
			log.V(1).Info("adjusting field managers as preparation of ssa")
			// TODO: add a metric to count if this happens
			gvk := object.GetObjectKind().GroupVersionKind()
			obj := &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: object.GetNamespace(),
					Name:      object.GetName(),
				},
			}
			preparePatch := []map[string]any{
				{"op": "replace", "path": "/metadata/managedFields", "value": managedFields},
				{"op": "replace", "path": "/metadata/resourceVersion", "value": object.GetResourceVersion()},
			}
			// note: this must() is ok because marshalling the patch should always work
			if err := r.client.Patch(ctx, obj, client.RawPatch(apitypes.JSONPatchType, must(json.Marshal(preparePatch))), client.FieldOwner(r.fieldOwner)); err != nil {
				return err
			}
			object.SetResourceVersion(obj.GetResourceVersion())
		}
		return r.client.Patch(ctx, object, client.Apply, client.FieldOwner(r.fieldOwner), client.ForceOwnership)
	default:
		for _, finalizer := range existingObject.GetFinalizers() {
			controllerutil.AddFinalizer(object, finalizer)
		}
		return r.client.Update(ctx, object, client.FieldOwner(r.fieldOwner))
	}
}

// delete object; existingObject is optional; if present, its resourceVersion will be used as a delete precondition;
// deletion will always be performed with background propagation; if the object is a crd or an api service which is still in use,
// then an error will be returned after issuing the delete call; otherwise, if the crd or api service is not in use, then our
// finalizer (i.e. the finalizer equal to the reconciler name) will be cleared, such that the object can be physically
// removed (unless other finalizers prevent this)
func (r *Reconciler) deleteObject(ctx context.Context, key types.ObjectKey, existingObject *unstructured.Unstructured, hashedOwnerId string) (err error) {
	if counter := r.metrics.DeleteCounter; counter != nil {
		counter.Inc()
	}

	defer func() {
		if existingObject == nil {
			return
		}
		if err == nil {
			r.client.EventRecorder().Event(existingObject, corev1.EventTypeNormal, objectReasonDeleted, "Object successfully deleted")
		} else {
			r.client.EventRecorder().Eventf(existingObject, corev1.EventTypeWarning, objectReasonDeleteError, "Error deleting object: %s", err)
		}
	}()

	log := log.FromContext(ctx).WithValues("object", types.ObjectKeyToString(key))

	// TODO: validate (by panic) that existingObject (if present) fits to key

	if existingObject != nil && existingObject.GetLabels()[r.labelKeyOwnerId] != hashedOwnerId {
		return fmt.Errorf("owner conflict; object %s has no or different owner", types.ObjectKeyToString(key))
	}

	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(key.GetObjectKind().GroupVersionKind())
	object.SetNamespace(key.GetNamespace())
	object.SetName(key.GetName())
	deleteOptions := &client.DeleteOptions{PropagationPolicy: ref(metav1.DeletePropagationBackground)}
	if existingObject != nil {
		deleteOptions.Preconditions = &metav1.Preconditions{
			ResourceVersion: ref(existingObject.GetResourceVersion()),
		}
	}
	if err := r.client.Delete(ctx, object, deleteOptions); err != nil {
		if apimeta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	switch {
	case isCrd(key):
		for i := 1; i <= 2; i++ {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := r.client.Get(ctx, apitypes.NamespacedName{Name: key.GetName()}, crd); err != nil {
				return client.IgnoreNotFound(err)
			}
			used, err := r.isCrdUsed(ctx, crd, hashedOwnerId, false)
			if err != nil {
				return err
			}
			if used {
				return fmt.Errorf("error deleting custom resource definition %s, existing instances found", types.ObjectKeyToString(key))
			}
			if ok := controllerutil.RemoveFinalizer(crd, r.finalizer); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the api server); this is why we retry once
				if err := r.client.Update(ctx, crd, client.FieldOwner(r.fieldOwner)); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating CustomResourcedefinition (409 conflict); doing one retry", "error", err.Error())
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
			if err := r.client.Get(ctx, apitypes.NamespacedName{Name: key.GetName()}, apiService); err != nil {
				return client.IgnoreNotFound(err)
			}
			used, err := r.isApiServiceUsed(ctx, apiService, hashedOwnerId, false)
			if err != nil {
				return err
			}
			if used {
				return fmt.Errorf("error deleting api service %s, existing instances found", types.ObjectKeyToString(key))
			}
			if ok := controllerutil.RemoveFinalizer(apiService, r.finalizer); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the api server); this is why we retry once
				if err := r.client.Update(ctx, apiService, client.FieldOwner(r.fieldOwner)); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating APIService (409 conflict); doing one retry", "error", err.Error())
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

func (r *Reconciler) getAdoptionPolicy(object client.Object) (AdoptionPolicy, error) {
	adoptionPolicy := strcase.ToKebab(object.GetAnnotations()[r.annotationKeyAdoptionPolicy])
	switch adoptionPolicy {
	case "":
		return r.adoptionPolicy, nil
	case types.AdoptionPolicyNever, types.AdoptionPolicyIfUnowned, types.AdoptionPolicyAlways:
		return adoptionPolicyByAnnotation[adoptionPolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", r.annotationKeyAdoptionPolicy, adoptionPolicy)
	}
}

func (r *Reconciler) getReconcilePolicy(object client.Object) (ReconcilePolicy, error) {
	reconcilePolicy := strcase.ToKebab(object.GetAnnotations()[r.annotationKeyReconcilePolicy])
	switch reconcilePolicy {
	case "":
		return r.reconcilePolicy, nil
	case types.ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnce:
		return reconcilePolicyByAnnotation[reconcilePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", r.annotationKeyReconcilePolicy, reconcilePolicy)
	}
}

func (r *Reconciler) getUpdatePolicy(object client.Object) (UpdatePolicy, error) {
	updatePolicy := strcase.ToKebab(object.GetAnnotations()[r.annotationKeyUpdatePolicy])
	switch updatePolicy {
	case "", types.UpdatePolicyDefault:
		return r.updatePolicy, nil
	case types.UpdatePolicyRecreate, types.UpdatePolicyReplace, types.UpdatePolicySsaMerge, types.UpdatePolicySsaOverride:
		return updatePolicyByAnnotation[updatePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", r.annotationKeyUpdatePolicy, updatePolicy)
	}
}

func (r *Reconciler) getDeletePolicy(object client.Object) (DeletePolicy, error) {
	deletePolicy := strcase.ToKebab(object.GetAnnotations()[r.annotationKeyDeletePolicy])
	switch deletePolicy {
	case "", types.DeletePolicyDefault:
		return r.deletePolicy, nil
	case types.DeletePolicyDelete, types.DeletePolicyOrphan, types.DeletePolicyOrphanOnApply, types.DeletePolicyOrphanOnDelete:
		return deletePolicyByAnnotation[deletePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", r.annotationKeyDeletePolicy, deletePolicy)
	}
}

func (r *Reconciler) getApplyOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[r.annotationKeyApplyOrder]
	if !ok {
		return 0, nil
	}
	applyOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyApplyOrder, value)
	}
	if err := checkRange(applyOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyApplyOrder, value)
	}
	return applyOrder, nil
}

func (r *Reconciler) getPurgeOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[r.annotationKeyPurgeOrder]
	if !ok {
		return maxOrder + 1, nil
	}
	purgeOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyPurgeOrder, value)
	}
	if err := checkRange(purgeOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyPurgeOrder, value)
	}
	return purgeOrder, nil
}

func (r *Reconciler) getDeleteOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[r.annotationKeyDeleteOrder]
	if !ok {
		return 0, nil
	}
	deleteOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyDeleteOrder, value)
	}
	if err := checkRange(deleteOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", r.annotationKeyDeleteOrder, value)
	}
	return deleteOrder, nil
}

func (r *Reconciler) isTypeUsed(ctx context.Context, gk schema.GroupKind, hashedOwnerId string, onlyForeign bool) (bool, error) {
	resLists, err := r.client.DiscoveryClient().ServerPreferredResources()
	if err != nil {
		return false, err
	}
	var gvks []schema.GroupVersionKind
	for _, resList := range resLists {
		gv, err := schema.ParseGroupVersion(resList.GroupVersion)
		if err != nil {
			return false, err
		}
		if matches(gv.Group, gk.Group) {
			for _, res := range resList.APIResources {
				if gk.Kind == "*" || gk.Kind == res.Kind {
					gvks = append(gvks, schema.GroupVersionKind{
						Group:   gv.Group,
						Version: gv.Version,
						Kind:    res.Kind,
					})
				}
			}
		}
	}
	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		labelSelector := labels.Everything()
		if onlyForeign {
			// note: this must() is ok because the label selector string is static, and correct
			labelSelector = must(labels.Parse(r.labelKeyOwnerId + "!=" + hashedOwnerId))
		}
		if err := r.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
			return false, err
		}
		if len(list.Items) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (r *Reconciler) isCrdUsed(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, hashedOwnerId string, onlyForeign bool) (bool, error) {
	gvk := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Versions[0].Name,
		Kind:    crd.Spec.Names.Kind,
	}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	labelSelector := labels.Everything()
	if onlyForeign {
		// note: this must() is ok because the label selector string is static, and correct
		labelSelector = must(labels.Parse(r.labelKeyOwnerId + "!=" + hashedOwnerId))
	}
	if err := r.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
		return false, err
	}
	return len(list.Items) > 0, nil
}

func (r *Reconciler) isApiServiceUsed(ctx context.Context, apiService *apiregistrationv1.APIService, hashedOwnerId string, onlyForeign bool) (bool, error) {
	gv := schema.GroupVersion{Group: apiService.Spec.Group, Version: apiService.Spec.Version}
	resList, err := r.client.DiscoveryClient().ServerResourcesForGroupVersion(gv.String())
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
		// note: this must() is ok because the label selector string is static, and correct
		labelSelector = must(labels.Parse(r.labelKeyOwnerId + "!=" + hashedOwnerId))
	}
	for _, kind := range kinds {
		gvk := schema.GroupVersionKind{
			Group:   apiService.Spec.Group,
			Version: apiService.Spec.Version,
			Kind:    kind,
		}
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := r.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
			return false, err
		}
		if len(list.Items) > 0 {
			return true, nil
		}
	}
	return false, nil
}

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
	"strconv"
	"strings"

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
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/pkg/manifests"
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
	types.DeletePolicyDelete: DeletePolicyDelete,
	types.DeletePolicyOrphan: DeletePolicyOrphan,
}

type reconcileTarget[T Component] struct {
	reconcilerName               string
	reconcilerId                 string
	client                       cluster.Client
	resourceGenerator            manifests.Generator
	createMissingNamespaces      bool
	adoptionPolicy               AdoptionPolicy
	reconcilePolicy              ReconcilePolicy
	updatePolicy                 UpdatePolicy
	deletePolicy                 DeletePolicy
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

func newReconcileTarget[T Component](reconcilerName string, reconcilerId string, clnt cluster.Client, resourceGenerator manifests.Generator, createMissingNamespaces bool, adoptionPolicy AdoptionPolicy, updatePolicy UpdatePolicy) *reconcileTarget[T] {
	return &reconcileTarget[T]{
		reconcilerName:               reconcilerName,
		reconcilerId:                 reconcilerId,
		client:                       clnt,
		resourceGenerator:            resourceGenerator,
		createMissingNamespaces:      createMissingNamespaces,
		adoptionPolicy:               adoptionPolicy,
		reconcilePolicy:              ReconcilePolicyOnObjectChange,
		updatePolicy:                 updatePolicy,
		deletePolicy:                 DeletePolicyDelete,
		labelKeyOwnerId:              reconcilerName + "/" + types.LabelKeySuffixOwnerId,
		annotationKeyOwnerId:         reconcilerName + "/" + types.AnnotationKeySuffixOwnerId,
		annotationKeyDigest:          reconcilerName + "/" + types.AnnotationKeySuffixDigest,
		annotationKeyAdoptionPolicy:  reconcilerName + "/" + types.AnnotationKeySuffixAdoptionPolicy,
		annotationKeyReconcilePolicy: reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy,
		annotationKeyUpdatePolicy:    reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy,
		annotationKeyDeletePolicy:    reconcilerName + "/" + types.AnnotationKeySuffixDeletePolicy,
		annotationKeyApplyOrder:      reconcilerName + "/" + types.AnnotationKeySuffixApplyOrder,
		annotationKeyPurgeOrder:      reconcilerName + "/" + types.AnnotationKeySuffixPurgeOrder,
		annotationKeyDeleteOrder:     reconcilerName + "/" + types.AnnotationKeySuffixDeleteOrder,
	}
}

func (t *reconcileTarget[T]) Reconcile(ctx context.Context, component T) (bool, error) {
	log := log.FromContext(ctx)
	namespace := ""
	name := ""
	if placementConfiguration, ok := assertPlacementConfiguration(component); ok {
		namespace = placementConfiguration.GetDeploymentNamespace()
		name = placementConfiguration.GetDeploymentName()
	}
	if namespace == "" {
		namespace = component.GetNamespace()
	}
	if name == "" {
		name = component.GetName()
	}
	ownerId := t.reconcilerId + "/" + component.GetNamespace() + "/" + component.GetName()
	hashedOwnerId := sha256base32([]byte(ownerId))
	// TODO: remove the legacyOwnerId check (needed until we are sure that all owner id labels have the new format)
	legacyOwnerId := component.GetNamespace() + "_" + component.GetName()
	status := component.GetStatus()
	componentDigest := calculateComponentDigest(component)

	// render manifests
	generateCtx := newContext(ctx).
		WithReconcilerName(t.reconcilerName).
		WithClient(t.client).
		WithComponent(component).
		WithComponentDigest(componentDigest)
	objects, err := t.resourceGenerator.Generate(generateCtx, namespace, name, component.GetSpec())
	if err != nil {
		return false, errors.Wrap(err, "error rendering manifests")
	}

	// perform some validation and cleanup on rendered manifests
	for _, object := range objects {
		if object.GetGenerateName() != "" {
			return false, fmt.Errorf("object %s specifies metadata.generateName (but dependent objects are not allowed to do so)", types.ObjectKeyToString(object))
		}
		removeLabel(object, t.labelKeyOwnerId)
		removeAnnotation(object, t.annotationKeyOwnerId)
		removeAnnotation(object, t.annotationKeyDigest)
	}

	/*
		// if desired: write generated manifests to secret for debugging
		if debug := component.GetAnnotations()["component-operator-runtime.cs.sap.com/debug"]; debug == "true" {
			var buf bytes.Buffer
			for _, object := range objects {
				rawObject, err := kyaml.Marshal(object)
				if err != nil {
					return false, errors.Wrapf(err, "error serializing object %s", types.ObjectKeyToString(object))
				}
				buf.WriteString("---\n")
				buf.Write(rawObject)
			}
			debugSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "com.sap.cs.component-operator-runtime.release." + name,
				},
				Data: map[string][]byte{
					"objects": buf.Bytes(),
				},
			}
			objects = append(objects, debugSecret)
		}
	*/

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
			if t.client.Scheme().Recognizes(gvk) {
				typedObject, err := t.client.Scheme().New(gvk)
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
			_gvk, err := apiutil.GVKForObject(object, t.client.Scheme())
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

		// TODO: client now has a method IsObjectNamespaced(); can we use this instead?
		scope := scopeUnknown
		restMapping, err := t.client.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err == nil {
			scope = scopeFromRestMapping(restMapping)
		} else if !meta.IsNoMatchError(err) {
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
	// 1. the generator provided wrong information and
	// 2. calling RESTMapping() above returned a NoMatchError (i.e. the type is currently not known to the api server) and
	// 3. the type belongs to a (new) api service which is part of this component
	// such entries can cause trouble, e.g. because InventoryItem.Match() might not work reliably ...
	// TODO: should we allow at all that api services and according instances are part of the same component?

	// validate annotations
	for _, object := range objects {
		if _, err := t.getAdoptionPolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getReconcilePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getUpdatePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getDeletePolicy(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getApplyOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getPurgeOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getDeleteOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
	}

	// define getter functions for later usage
	getAdoptionPolicy := func(object client.Object) AdoptionPolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getAdoptionPolicy(object))
	}
	getReconcilePolicy := func(object client.Object) ReconcilePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getReconcilePolicy(object))
	}
	getUpdatePolicy := func(object client.Object) UpdatePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getUpdatePolicy(object))
	}
	getDeletePolicy := func(object client.Object) DeletePolicy {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getDeletePolicy(object))
	}
	getApplyOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getApplyOrder(object))
	}
	getPurgeOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getPurgeOrder(object))
	}
	getDeleteOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getDeleteOrder(object))
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

	// add/update inventory with target objects
	// TODO: review this; it would be cleaner to use a DeepCopy method for a []*InventoryItem type (if there would be such a type)
	inventory := slices.Collect(status.Inventory, func(item *InventoryItem) *InventoryItem { return item.DeepCopy() })
	numAdded := 0
	for _, object := range objects {
		// retrieve inventory item belonging to this object (if existing)
		item := getItem(inventory, object)

		// calculate object digest
		digest, err := calculateObjectDigest(object)
		if err != nil {
			return false, errors.Wrapf(err, "error calculating digest for object %s", types.ObjectKeyToString(object))
		}
		switch getReconcilePolicy(object) {
		case ReconcilePolicyOnObjectOrComponentChange:
			digest = fmt.Sprintf("%s@%d", digest, component.GetGeneration())
		case ReconcilePolicyOnce:
			// note: if the object already existed with a different reconcile policy, then it will get reconciled one (and only one) more time
			digest = "__once__"
		}

		// if item was not found, append an empty item
		if item == nil {
			// fetch object (if existing)
			existingObject, err := t.readObject(ctx, object)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", types.ObjectKeyToString(object))
			}
			// check ownership
			// note: failing already here in case of a conflict prevents problems during apply and, in particular, during deletion
			if existingObject != nil {
				adoptionPolicy := getAdoptionPolicy(object)
				existingOwnerId := existingObject.GetLabels()[t.labelKeyOwnerId]
				if existingOwnerId == "" {
					if adoptionPolicy != AdoptionPolicyIfUnowned && adoptionPolicy != AdoptionPolicyAlways {
						return false, fmt.Errorf("found existing object %s without owner", types.ObjectKeyToString(object))
					}
				} else if existingOwnerId != hashedOwnerId && existingOwnerId != legacyOwnerId {
					if adoptionPolicy != AdoptionPolicyAlways {
						return false, fmt.Errorf("owner conflict; object %s is owned by %s", types.ObjectKeyToString(object), existingObject.GetAnnotations()[t.annotationKeyOwnerId])
					}
				}
			}
			inventory = append(inventory, &InventoryItem{})
			item = inventory[len(inventory)-1]
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
			item.Status = kstatus.InProgressStatus.String()
		}
	}

	// mark obsolete inventory items (clear digest)
	for _, item := range inventory {
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

	// validate object set:
	// - check that all managed instances have apply-order greater than or equal to the according managed type
	// - check that all managed instances have delete-order less than or equal to the according managed type
	// - check that no managed types are about to be deleted (empty digest) unless all related managed instances are as well
	// - check that all contained objects have apply-order greater than or equal to the according namespace
	// - check that all contained objects have delete-order less than or equal to the according namespace
	// - check that no namespaces are about to be deleted (empty digest) unless all contained objects are as well
	for _, item := range inventory {
		if isCrd(item) || isApiService(item) {
			for _, _item := range inventory {
				if isManagedBy(item, _item) {
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
			for _, _item := range inventory {
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

	// accept inventory for further processing, put into right order for future deletion
	status.Inventory = sortObjectsForDelete(inventory)

	// trigger another reconcile if something was added (to be sure that it is persisted)
	if numAdded > 0 {
		return false, nil
	}

	// note: after this point it is guaranteed that
	// - the in-memory inventory reflects the target state
	// - the persisted inventory at least has the same object keys as the in-memory inventory
	// now it is about to synchronize the cluster state with the inventory

	// delete redundant objects and maintain inventory;
	// objects are deleted in waves according to their delete order;
	// that means, only if all redundant objects of a wave are gone or comppleted, the next
	// wave will be processed; within each wave, objects which are instances of managed
	// types are deleted before all other objects, and namespaces will only be deleted
	// if they are not used by any object in the inventory (note that this may cause deadlocks)
	numManagedToBeDeleted := 0
	numToBeDeleted := 0
	for k, item := range status.Inventory {
		// if this is the first object of an order, then
		// count instances of managed types in this wave which are about to be deleted
		if k == 0 || status.Inventory[k-1].DeleteOrder < item.DeleteOrder {
			log.V(2).Info("begin of deletion wave", "order", item.DeleteOrder)
			numManagedToBeDeleted = 0
			for j := k; j < len(status.Inventory) && status.Inventory[j].DeleteOrder == item.DeleteOrder; j++ {
				_item := status.Inventory[j]
				if (_item.Phase == PhaseScheduledForDeletion || _item.Phase == PhaseScheduledForCompletion || _item.Phase == PhaseDeleting || _item.Phase == PhaseCompleting) && isInstanceOfManagedType(status.Inventory, _item) {
					numManagedToBeDeleted++
				}
			}
		}

		if item.Phase == PhaseScheduledForDeletion || item.Phase == PhaseScheduledForCompletion || item.Phase == PhaseDeleting || item.Phase == PhaseCompleting {
			// fetch object (if existing)
			existingObject, err := t.readObject(ctx, item)
			if err != nil {
				return false, errors.Wrapf(err, "error reading object %s", item)
			}

			orphan := item.DeletePolicy == DeletePolicyOrphan

			switch item.Phase {
			case PhaseScheduledForDeletion:
				// delete namespaces after all contained inventory items
				// delete all instances of managed types before remaining objects; this ensures that no objects are prematurely
				// deleted which are needed for the deletion of the managed instances, such as webhook servers, api servers, ...
				if (!isNamespace(item) || !isNamespaceUsed(status.Inventory, item.Name)) && (numManagedToBeDeleted == 0 || isInstanceOfManagedType(status.Inventory, item)) {
					if orphan {
						item.Phase = ""
					} else {
						// note: here is a theoretical risk that we delete an existing foreign object, because informers are not yet synced
						// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
						// TODO: perform an additional owner id check
						if err := t.deleteObject(ctx, item, existingObject); err != nil {
							return false, errors.Wrapf(err, "error deleting object %s", item)
						}
						item.Phase = PhaseDeleting
						item.Status = kstatus.TerminatingStatus.String()
						numToBeDeleted++
					}
				} else {
					numToBeDeleted++
				}
			case PhaseScheduledForCompletion:
				// delete namespaces after all contained inventory items
				// delete all instances of managed types before remaining objects; this ensures that no objects are prematurely
				// deleted which are needed for the deletion of the managed instances, such as webhook servers, api servers, ...
				if (!isNamespace(item) || !isNamespaceUsed(status.Inventory, item.Name)) && (numManagedToBeDeleted == 0 || isInstanceOfManagedType(status.Inventory, item)) {
					if orphan {
						return false, fmt.Errorf("invalid usage of deletion policy: object %s is scheduled for completion and therefore cannot be orphaned", item)
					} else {
						// note: here is a theoretical risk that we delete an existing foreign object, because informers are not yet synced
						// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
						// TODO: perform an additional owner id check
						if err := t.deleteObject(ctx, item, existingObject); err != nil {
							return false, errors.Wrapf(err, "error deleting object %s", item)
						}
						item.Phase = PhaseCompleting
						item.Status = kstatus.TerminatingStatus.String()
						numToBeDeleted++
					}
				} else {
					numToBeDeleted++
				}
			case PhaseDeleting:
				// if object is gone, we can remove it from inventory
				if existingObject == nil {
					item.Phase = ""
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
				// note: any other phase value would indicate a severe code problem, so we want to see the panic in that case
				panic("this cannot happen")
			}
		}

		// trigger another reconcile if this is the last object of the wave, and some deletions are not yet completed
		if k == len(status.Inventory)-1 || status.Inventory[k+1].DeleteOrder > item.DeleteOrder {
			log.V(2).Info("end of deletion wave", "order", item.DeleteOrder)
			if numToBeDeleted > 0 {
				break
			}
		}
	}

	status.Inventory = slices.Select(status.Inventory, func(item *InventoryItem) bool { return item.Phase != "" })

	// trigger another reconcile
	if numToBeDeleted > 0 {
		return false, nil
	}

	// note: after this point, PhaseScheduledForDeletion, PhaseScheduledForCompletion, PhaseDeleting, PhaseCompleting cannot occur anymore in status.Inventory
	// in other words: status.Inventory and objects contains the same resources

	// create missing namespaces
	if t.createMissingNamespaces {
		for _, namespace := range findMissingNamespaces(objects) {
			if err := t.client.Get(ctx, apitypes.NamespacedName{Name: namespace}, &corev1.Namespace{}); err != nil {
				if !apierrors.IsNotFound(err) {
					return false, errors.Wrapf(err, "error reading namespace %s", namespace)
				}
				if err := t.client.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, client.FieldOwner(t.reconcilerName)); err != nil {
					return false, errors.Wrapf(err, "error creating namespace %s", namespace)
				}
			}
		}
	}

	// put objects into right order for applying
	objects = sortObjectsForApply(objects, getApplyOrder)

	// apply objects and maintain inventory;
	// objects are applied (i.e. created/updated) in waves according to their apply order;
	// that means, only if all objects of a wave are ready or completed, the next wave
	// will be procesed; within each wave, objects which are instances of managed types
	// will be applied after all other objects
	numNotManagedToBeApplied := 0
	numUnready := 0
	for k, object := range objects {
		// retrieve inventory item corresponding to this object
		item := mustGetItem(status.Inventory, object)

		// retrieve object order
		applyOrder := getApplyOrder(object)

		// if this is the first object of an order, then
		// count instances of managed types in this order which are about to be applied
		if k == 0 || getApplyOrder(objects[k-1]) < applyOrder {
			log.V(2).Info("begin of apply wave", "order", applyOrder)
			numNotManagedToBeApplied = 0
			for j := k; j < len(objects) && getApplyOrder(objects[j]) == applyOrder; j++ {
				_object := objects[j]
				_item := mustGetItem(status.Inventory, _object)
				if _item.Phase != PhaseReady && _item.Phase != PhaseCompleted && !isInstanceOfManagedType(status.Inventory, _object) {
					// that means: _item.Phase is one of PhaseScheduledForApplication, PhaseCreating, PhaseUpdating
					numNotManagedToBeApplied++
				}
			}
		}

		// for non-completed objects, compute and update status, and apply (create or update) the object if necessary
		if item.Phase != PhaseCompleted {
			// reconcile all instances of managed types after remaining objects
			// this ensures that everything is running what is needed for the reconciliation of the managed instances,
			// such as webhook servers, api servers, ...
			if numNotManagedToBeApplied == 0 || !isInstanceOfManagedType(status.Inventory, object) {
				// fetch object (if existing)
				existingObject, err := t.readObject(ctx, item)
				if err != nil {
					return false, errors.Wrapf(err, "error reading object %s", item)
				}

				setLabel(object, t.labelKeyOwnerId, hashedOwnerId)
				setAnnotation(object, t.annotationKeyOwnerId, ownerId)
				setAnnotation(object, t.annotationKeyDigest, item.Digest)

				if existingObject == nil {
					if err := t.createObject(ctx, object, nil); err != nil {
						return false, errors.Wrapf(err, "error creating object %s", item)
					}
					item.Phase = PhaseCreating
					item.Status = kstatus.InProgressStatus.String()
					numUnready++
				} else if existingObject.GetDeletionTimestamp().IsZero() && existingObject.GetAnnotations()[t.annotationKeyDigest] != item.Digest {
					updatePolicy := getUpdatePolicy(object)
					switch updatePolicy {
					case UpdatePolicyRecreate:
						// TODO: perform an additional owner id check
						if err := t.deleteObject(ctx, object, existingObject); err != nil {
							return false, errors.Wrapf(err, "error deleting (while recreating) object %s", item)
						}
					default:
						// TODO: perform an additional owner id check
						if err := t.updateObject(ctx, object, existingObject, nil, updatePolicy); err != nil {
							return false, errors.Wrapf(err, "error updating object %s", item)
						}
					}
					item.Phase = PhaseUpdating
					item.Status = kstatus.InProgressStatus.String()
					numUnready++
				} else {
					res, err := computeStatus(existingObject)
					if err != nil {
						return false, errors.Wrapf(err, "error checking status of object %s", item)
					}
					if existingObject.GetDeletionTimestamp().IsZero() && res.Status == kstatus.CurrentStatus {
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
		if k == len(objects)-1 || getApplyOrder(objects[k+1]) > applyOrder {
			log.V(2).Info("end of apply wave", "order", applyOrder)
			if numUnready == 0 {
				numPurged := 0
				for j := 0; j <= k; j++ {
					_object := objects[j]
					_item := mustGetItem(status.Inventory, _object)
					_purgeOrder := getPurgeOrder(_object)
					if (k == len(objects)-1 && _purgeOrder <= maxOrder || _purgeOrder <= applyOrder) && _item.Phase != PhaseCompleted {
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
	log := log.FromContext(ctx)
	status := component.GetStatus()

	// delete objects and maintain inventory;
	// objects are deleted in waves according to their delete order;
	// that means, only if all objects of a wave are gone, the next wave will be processed;
	// within each wave, objects which are instances of managed types are deleted before all
	// other objects, and namespaces will only be deleted if they are not used by any
	// object in the inventory (note that this may cause deadlocks)
	numManagedToBeDeleted := 0
	numToBeDeleted := 0
	for k, item := range status.Inventory {
		// if this is the first object of an order, then
		// count instances of managed types in this wave which are about to be deleted
		if k == 0 || status.Inventory[k-1].DeleteOrder < item.DeleteOrder {
			log.V(2).Info("begin of deletion wave", "order", item.DeleteOrder)
			numManagedToBeDeleted = 0
			for j := k; j < len(status.Inventory) && status.Inventory[j].DeleteOrder == item.DeleteOrder; j++ {
				_item := status.Inventory[j]
				if isInstanceOfManagedType(status.Inventory, _item) {
					numManagedToBeDeleted++
				}
			}
		}

		// fetch object (if existing)
		existingObject, err := t.readObject(ctx, item)
		if err != nil {
			return false, errors.Wrapf(err, "error reading object %s", item)
		}

		orphan := item.DeletePolicy == DeletePolicyOrphan

		switch item.Phase {
		case PhaseDeleting:
			// if object is gone, we can remove it from inventory
			if existingObject == nil {
				item.Phase = ""
			} else {
				numToBeDeleted++
			}
		default:
			// delete namespaces after all contained inventory items
			// delete all instances of managed types before remaining objects; this ensures that no objects are prematurely
			// deleted which are needed for the deletion of the managed instances, such as webhook servers, api servers, ...
			if (!isNamespace(item) || !isNamespaceUsed(status.Inventory, item.Name)) && (numManagedToBeDeleted == 0 || isInstanceOfManagedType(status.Inventory, item)) {
				if orphan {
					item.Phase = ""
				} else {
					// delete the object
					// note: here is a theoretical risk that we delete an existing (foreign) object, because informers are not yet synced
					// however not sending the delete request is also not an option, because this might lead to orphaned own dependents
					// TODO: perform an additional owner id check
					if err := t.deleteObject(ctx, item, existingObject); err != nil {
						return false, errors.Wrapf(err, "error deleting object %s", item)
					}
					item.Phase = PhaseDeleting
					item.Status = kstatus.TerminatingStatus.String()
					numToBeDeleted++
				}
			} else {
				numToBeDeleted++
			}
		}

		// trigger another reconcile if this is the last object of the wave, and some deletions are not yet completed
		if k == len(status.Inventory)-1 || status.Inventory[k+1].DeleteOrder > item.DeleteOrder {
			log.V(2).Info("end of deletion wave", "order", item.DeleteOrder)
			if numToBeDeleted > 0 {
				break
			}
		}
	}

	status.Inventory = slices.Select(status.Inventory, func(item *InventoryItem) bool { return item.Phase != "" })

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

// reaad object and return as unstructured
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

// create object; object may be a concrete type or unstructured; in any case, type meta must be populated;
// createdObject is optional; if non-nil, it will be populated with the created object; the same variable can be supplied as object and createObject;
// if object is a crd or an api services, the reconciler's name will be added as finalizer
func (t *reconcileTarget[T]) createObject(ctx context.Context, object client.Object, createdObject any) (err error) {
	defer func() {
		if err == nil && createdObject != nil {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.(*unstructured.Unstructured).Object, createdObject)
		}
		if err == nil {
			t.client.EventRecorder().Event(object, corev1.EventTypeNormal, objectReasonCreated, "Object successfully created")
		}
	}()
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return err
	}
	object = &unstructured.Unstructured{Object: data}
	if isCrd(object) || isApiService(object) {
		controllerutil.AddFinalizer(object, t.reconcilerName)
	}
	return t.client.Create(ctx, object, client.FieldOwner(t.reconcilerName))
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
func (t *reconcileTarget[T]) updateObject(ctx context.Context, object client.Object, existingObject *unstructured.Unstructured, updatedObject any, updatePolicy UpdatePolicy) (err error) {
	defer func() {
		if err == nil && updatedObject != nil {
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.(*unstructured.Unstructured).Object, updatedObject)
		}
		if err == nil {
			t.client.EventRecorder().Event(object, corev1.EventTypeNormal, objectReasonUpdated, "Object successfully updated")
		} else {
			t.client.EventRecorder().Eventf(existingObject, corev1.EventTypeWarning, objectReasonUpdateError, "Error updating object: %s", err)
		}
	}()
	// TODO: validate (by panic) that existingObject fits to object
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
		controllerutil.AddFinalizer(object, t.reconcilerName)
	}
	if object.GetResourceVersion() == "" {
		object.SetResourceVersion((existingObject.GetResourceVersion()))
	}
	// note: clearing managedFields is anyway required for ssa; but also in the replace (put, update) case it does not harm;
	// because replace will only claim fields which are new or which have changed; the field owner of declared (but unmodified)
	// fields will not be touched
	object.SetManagedFields(nil)
	switch updatePolicy {
	case UpdatePolicySsaMerge:
		return t.client.Patch(ctx, object, client.Apply, client.FieldOwner(t.reconcilerName), client.ForceOwnership)
	case UpdatePolicySsaOverride:
		// TODO: add ways (per reconciler, per component, per object) to configure the list of field manager (prefixes) which are reclaimed
		if managedFields, changed, err := replaceFieldManager(existingObject.GetManagedFields(), []string{"kubectl", "helm"}, t.reconcilerName); err != nil {
			return err
		} else if changed {
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
			if err := t.client.Patch(ctx, obj, client.RawPatch(apitypes.JSONPatchType, must(json.Marshal(preparePatch))), client.FieldOwner(t.reconcilerName)); err != nil {
				return err
			}
			object.SetResourceVersion(obj.GetResourceVersion())
		}
		return t.client.Patch(ctx, object, client.Apply, client.FieldOwner(t.reconcilerName), client.ForceOwnership)
	default:
		for _, finalizer := range existingObject.GetFinalizers() {
			controllerutil.AddFinalizer(object, finalizer)
		}
		return t.client.Update(ctx, object, client.FieldOwner(t.reconcilerName))
	}
}

// delete object; existingObject is optional; if present, its resourceVersion will be used as a delete precondition;
// deletion will always be performed with background propagation; if the object is a crd or an api service which is still in use,
// then an error will be returned after issuing the delete call; otherwise, if the crd or api service is not in use, then our
// finalizer (i.e. the finalizer equal to the reconciler name) will be cleared, such that the object can be physically
// removed (unless other finalizers prevent this)
func (t *reconcileTarget[T]) deleteObject(ctx context.Context, key types.ObjectKey, existingObject *unstructured.Unstructured) (err error) {
	defer func() {
		if existingObject == nil {
			return
		}
		if err == nil {
			t.client.EventRecorder().Event(existingObject, corev1.EventTypeNormal, objectReasonDeleted, "Object successfully deleted")
		} else {
			t.client.EventRecorder().Eventf(existingObject, corev1.EventTypeWarning, objectReasonDeleteError, "Error deleting object: %s", err)
		}
	}()
	// TODO: validate (by panic) that existingObject (if present) fits to key
	log := log.FromContext(ctx)

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
			if ok := controllerutil.RemoveFinalizer(crd, t.reconcilerName); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the api server); this is why we retry once
				if err := t.client.Update(ctx, crd, client.FieldOwner(t.reconcilerName)); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating CustomResourcedefinition (409 conflict); doing one retry", "name", t.reconcilerName, "error", err.Error())
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
			if ok := controllerutil.RemoveFinalizer(apiService, t.reconcilerName); ok {
				// note: 409 error is very likely here (because of concurrent updates happening through the api server); this is why we retry once
				if err := t.client.Update(ctx, apiService, client.FieldOwner(t.reconcilerName)); err != nil {
					if i == 1 && apierrors.IsConflict(err) {
						log.V(1).Info("error while updating APIService (409 conflict); doing one retry", "name", t.reconcilerName, "error", err.Error())
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

func (t *reconcileTarget[T]) getAdoptionPolicy(object client.Object) (AdoptionPolicy, error) {
	adoptionPolicy := object.GetAnnotations()[t.annotationKeyAdoptionPolicy]
	switch adoptionPolicy {
	case "":
		return t.adoptionPolicy, nil
	case types.AdoptionPolicyNever, types.AdoptionPolicyIfUnowned, types.AdoptionPolicyAlways:
		return adoptionPolicyByAnnotation[adoptionPolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyAdoptionPolicy, adoptionPolicy)
	}
}

func (t *reconcileTarget[T]) getReconcilePolicy(object client.Object) (ReconcilePolicy, error) {
	reconcilePolicy := object.GetAnnotations()[t.annotationKeyReconcilePolicy]
	switch reconcilePolicy {
	case "":
		return t.reconcilePolicy, nil
	case types.ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnce:
		return reconcilePolicyByAnnotation[reconcilePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyReconcilePolicy, reconcilePolicy)
	}
}

func (t *reconcileTarget[T]) getUpdatePolicy(object client.Object) (UpdatePolicy, error) {
	updatePolicy := object.GetAnnotations()[t.annotationKeyUpdatePolicy]
	switch updatePolicy {
	case "", types.UpdatePolicyDefault:
		return t.updatePolicy, nil
	case types.UpdatePolicyRecreate, types.UpdatePolicyReplace, types.UpdatePolicySsaMerge, types.UpdatePolicySsaOverride:
		return updatePolicyByAnnotation[updatePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyUpdatePolicy, updatePolicy)
	}
}

func (t *reconcileTarget[T]) getDeletePolicy(object client.Object) (DeletePolicy, error) {
	deletePolicy := object.GetAnnotations()[t.annotationKeyDeletePolicy]
	switch deletePolicy {
	case "", types.DeletePolicyDefault:
		return t.deletePolicy, nil
	case types.DeletePolicyDelete, types.DeletePolicyOrphan:
		return deletePolicyByAnnotation[deletePolicy], nil
	default:
		return "", fmt.Errorf("invalid value for annotation %s: %s", t.annotationKeyDeletePolicy, deletePolicy)
	}
}

func (t *reconcileTarget[T]) getApplyOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[t.annotationKeyApplyOrder]
	if !ok {
		return 0, nil
	}
	applyOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyApplyOrder, value)
	}
	if err := checkRange(applyOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyApplyOrder, value)
	}
	return applyOrder, nil
}

func (t *reconcileTarget[T]) getPurgeOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[t.annotationKeyPurgeOrder]
	if !ok {
		return maxOrder + 1, nil
	}
	purgeOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyPurgeOrder, value)
	}
	if err := checkRange(purgeOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyPurgeOrder, value)
	}
	return purgeOrder, nil
}

func (t *reconcileTarget[T]) getDeleteOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[t.annotationKeyDeleteOrder]
	if !ok {
		return 0, nil
	}
	deleteOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyDeleteOrder, value)
	}
	if err := checkRange(deleteOrder, minOrder, maxOrder); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyDeleteOrder, value)
	}
	return deleteOrder, nil
}

func (t *reconcileTarget[T]) isCrdUsed(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, onlyForeign bool) (bool, error) {
	gvk := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Versions[0].Name,
		Kind:    crd.Spec.Names.Kind,
	}
	// TODO: better use metav1.PartialObjectMetadataList?
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	labelSelector := labels.Everything()
	if onlyForeign {
		// TODO: remove the below workaround logic (needed until we are sure that all owner id labels have the new format)
		hashedOwnerId := ""
		legacyOwnerId := ""
		if strings.ContainsRune(crd.Labels[t.labelKeyOwnerId], '_') {
			hashedOwnerId = sha256base32([]byte(t.reconcilerId + "/" + crd.Annotations[t.annotationKeyOwnerId]))
			legacyOwnerId = crd.Labels[t.labelKeyOwnerId]
		} else {
			hashedOwnerId = crd.Labels[t.labelKeyOwnerId]
			legacyOwnerId = strings.Join(slices.Last(strings.Split(crd.Annotations[t.annotationKeyOwnerId], "/"), 2), "_")
		}
		// note: this must() is ok because the label selector string is static, and correct
		labelSelector = must(labels.Parse(t.labelKeyOwnerId + " notin (" + hashedOwnerId + "," + legacyOwnerId + ")"))
		// labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + "!=" + crd.Labels[t.labelKeyOwnerId])
	}
	if err := t.client.List(ctx, list, &client.ListOptions{LabelSelector: labelSelector, Limit: 1}); err != nil {
		return false, err
	}
	return len(list.Items) > 0, nil
}

func (t *reconcileTarget[T]) isApiServiceUsed(ctx context.Context, apiService *apiregistrationv1.APIService, onlyForeign bool) (bool, error) {
	gv := schema.GroupVersion{Group: apiService.Spec.Group, Version: apiService.Spec.Version}
	resList, err := t.client.DiscoveryClient().ServerResourcesForGroupVersion(gv.String())
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
		// TODO: remove the below workaround logic (needed until we are sure that all owner id labels have the new format)
		hashedOwnerId := ""
		legacyOwnerId := ""
		if strings.ContainsRune(apiService.Labels[t.labelKeyOwnerId], '_') {
			hashedOwnerId = sha256base32([]byte(t.reconcilerId + "/" + apiService.Annotations[t.annotationKeyOwnerId]))
			legacyOwnerId = apiService.Labels[t.labelKeyOwnerId]
		} else {
			hashedOwnerId = apiService.Labels[t.labelKeyOwnerId]
			legacyOwnerId = strings.Join(slices.Last(strings.Split(apiService.Annotations[t.annotationKeyOwnerId], "/"), 2), "_")
		}
		// note: this must() is ok because the label selector string is static, and correct
		labelSelector = must(labels.Parse(t.labelKeyOwnerId + " notin (" + hashedOwnerId + "," + legacyOwnerId + ")"))
		// labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + "!=" + crd.Labels[t.labelKeyOwnerId])
	}
	for _, kind := range kinds {
		gvk := schema.GroupVersionKind{
			Group:   apiService.Spec.Group,
			Version: apiService.Spec.Version,
			Kind:    kind,
		}
		// TODO: better use metav1.PartialObjectMetadataList?
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

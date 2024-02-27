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
	annotationKeyDigest          string
	annotationKeyAdoptionPolicy  string
	annotationKeyReconcilePolicy string
	annotationKeyUpdatePolicy    string
	annotationKeyDeletePolicy    string
	annotationKeyOrder           string
	annotationKeyPurgeOrder      string
	annotationKeyOwnerId         string
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
		annotationKeyDigest:          reconcilerName + "/" + types.AnnotationKeySuffixDigest,
		annotationKeyAdoptionPolicy:  reconcilerName + "/" + types.AnnotationKeySuffixAdoptionPolicy,
		annotationKeyReconcilePolicy: reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy,
		annotationKeyUpdatePolicy:    reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy,
		annotationKeyDeletePolicy:    reconcilerName + "/" + types.AnnotationKeySuffixDeletePolicy,
		annotationKeyOrder:           reconcilerName + "/" + types.AnnotationKeySuffixOrder,
		annotationKeyPurgeOrder:      reconcilerName + "/" + types.AnnotationKeySuffixPurgeOrder,
		annotationKeyOwnerId:         reconcilerName + "/" + types.AnnotationKeySuffixOwnerId,
	}
}

func (t *reconcileTarget[T]) Reconcile(ctx context.Context, component T) (bool, error) {
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
		if _, err := t.getOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
		if _, err := t.getPurgeOrder(object); err != nil {
			return false, errors.Wrapf(err, "error validating object %s", types.ObjectKeyToString(object))
		}
	}

	// define getter functions for later usage
	getOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getOrder(object))
	}
	getPurgeOrder := func(object client.Object) int {
		// note: this must() is ok because we checked the generated objects above, and this function will be called for these objects only
		return must(t.getPurgeOrder(object))
	}

	// add/update inventory with target objects
	numAdded := 0
	for _, object := range objects {
		// retrieve inventory item belonging to this object (if existing)
		item := getItem(status.Inventory, object)

		// calculate object digest
		digest, err := calculateObjectDigest(object)
		if err != nil {
			return false, errors.Wrapf(err, "error calculating digest for object %s", types.ObjectKeyToString(object))
		}
		// note: this must() is ok because we checked the generated objects above
		switch must(t.getReconcilePolicy(object)) {
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
			if existingObject != nil {
				// note: this must() is ok because we checked the generated objects above
				adoptionPolicy := must(t.getAdoptionPolicy(object))
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

	// TODO: delete-order
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

			orphan := false
			if existingObject != nil {
				deletePolicy, err := t.getDeletePolicy(existingObject)
				if err != nil {
					// note: this should not happen under normal circumstances, because we checked the annotation when persisting it
					return false, errors.Wrapf(err, "error validating existing object %s", types.ObjectKeyToString(existingObject))
				}
				orphan = deletePolicy == DeletePolicyOrphan
			}

			switch item.Phase {
			case PhaseScheduledForDeletion:
				if numManagedToBeDeleted == 0 || t.isManaged(item, component) {
					if orphan {
						continue
					}
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
					if orphan {
						return false, fmt.Errorf("invalid usage of deletion policy: object %s is scheduled for completion (due to purge order) and therefore cannot be orphaned", item)
					}
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
				// note: any other phase value would indicate a severe code problem, so we want to see the panic in that case
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
	objects = sortObjectsForApply(objects, getOrder)

	// apply objects and maintain inventory
	numUnready := 0
	numNotManagedToBeApplied := 0
	for k, object := range objects {
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

				setLabel(object, t.labelKeyOwnerId, hashedOwnerId)
				setAnnotation(object, t.annotationKeyOwnerId, ownerId)
				setAnnotation(object, t.annotationKeyDigest, item.Digest)

				if existingObject == nil {
					if err := t.createObject(ctx, object); err != nil {
						return false, errors.Wrapf(err, "error creating object %s", item)
					}
					item.Phase = PhaseCreating
					item.Status = kstatus.InProgressStatus.String()
					numUnready++
				} else if existingObject.GetDeletionTimestamp().IsZero() && existingObject.GetAnnotations()[t.annotationKeyDigest] != item.Digest {
					// note: this must() is ok because we checked the generated objects above
					updatePolicy := must(t.getUpdatePolicy(object))
					switch updatePolicy {
					case UpdatePolicyRecreate:
						if err := t.deleteObject(ctx, object, existingObject); err != nil {
							return false, errors.Wrapf(err, "error deleting (while recreating) object %s", item)
						}
					default:
						if err := t.updateObject(ctx, object, existingObject, updatePolicy); err != nil {
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
	// TODO: delete-order
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
			// orphan the object, if according deletion policy is set
			if existingObject != nil {
				deletePolicy, err := t.getDeletePolicy(existingObject)
				if err != nil {
					// note: this should not happen under normal circumstances, because we checked the annotation when persisting it
					return false, errors.Wrapf(err, "error validating existing object %s", types.ObjectKeyToString(existingObject))
				}
				if deletePolicy == DeletePolicyOrphan {
					continue
				}
			}
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
// if object is a crd or an api services, the reconciler's name will be added as finalizer
func (t *reconcileTarget[T]) createObject(ctx context.Context, object client.Object) (err error) {
	defer func() {
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
// if object is a crd or an api services, the reconciler's name will be added as finalizer;
// existingObject is required, and should represent the last-read state of the object; it must not have a deletionTimestamp set;
// object may have a resourceVersion; if it does not, the resourceVersion of existingObject will be used for conflict checks during put/patch;
// if updatePolicy equals UpdatePolicyReplace, an update (put) will be performed; finalizers of existingObject will be copied;
// if updatePolicy equals UpdatePolicySsaMerge, a conflict-forcing server-side-apply (patch) will be performed;
// if updatePolicy equals UpdatePolicySsaOverride, then in addition, a preparation patch request will be performed before doing the conflict-forcing
// server-side-apply patch; this preparation patch will adjust managedFields, reclaiming fields/values previously owned by kubectl or helm
func (t *reconcileTarget[T]) updateObject(ctx context.Context, object client.Object, existingObject *unstructured.Unstructured, updatePolicy UpdatePolicy) (err error) {
	defer func() {
		if err == nil {
			t.client.EventRecorder().Event(object, corev1.EventTypeNormal, objectReasonUpdated, "Object successfully updated")
		} else {
			t.client.EventRecorder().Eventf(existingObject, corev1.EventTypeWarning, objectReasonUpdateError, "Error updating object: %s", err)
		}
	}()
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
	// note: this must() is ok because we checked the generated objects before
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
// then an error will be returned; otherwise, if it is not in use, then our finalizer (i.e. the finalizer equal to the reconciler name)
// will be cleared, such that the object can be physically removed (unless other finalizers prevent this)
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
				// note: 409 error is very likely here (because of concurrent updates happening through the API server); this is why we retry once
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
				// note: 409 error is very likely here (because of concurrent updates happening through the API server); this is why we retry once
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

func (t *reconcileTarget[T]) getOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[t.annotationKeyOrder]
	if !ok {
		return 0, nil
	}
	order, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyOrder, value)
	}
	if err := checkRange(order, math.MinInt16, math.MaxInt16); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyOrder, value)
	}
	return order, nil
}

func (t *reconcileTarget[T]) getPurgeOrder(object client.Object) (int, error) {
	value, ok := object.GetAnnotations()[t.annotationKeyPurgeOrder]
	if !ok {
		return math.MaxInt, nil
	}
	purgeOrder, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyPurgeOrder, value)
	}
	if err := checkRange(purgeOrder, math.MinInt16, math.MaxInt16); err != nil {
		return 0, errors.Wrapf(err, "invalid value for annotation %s: %s", t.annotationKeyPurgeOrder, value)
	}
	return purgeOrder, nil
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
		labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + " notin (" + hashedOwnerId + "," + legacyOwnerId + ")")
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
		labelSelector = mustParseLabelSelector(t.labelKeyOwnerId + " notin (" + hashedOwnerId + "," + legacyOwnerId + ")")
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

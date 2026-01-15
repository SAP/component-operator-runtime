/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sap/component-operator-runtime/pkg/status"
)

// TypeVersionInfo represents a Kubernetes type version.
type TypeVersionInfo struct {
	// API group.
	Group string `json:"group"`
	// API group version.
	Version string `json:"version"`
	// API kind.
	Kind string `json:"kind"`
}

// TypeInfo represents a Kubernetes type.
type TypeInfo struct {
	// API group.
	Group string `json:"group"`
	// API kind.
	Kind string `json:"kind"`
}

// NameInfo represents an object's namespace and name.
type NameInfo struct {
	// Namespace of the referenced object; empty for non-namespaced objects
	Namespace string `json:"namespace,omitempty"`
	// Name of the referenced object.
	Name string `json:"name"`
}

// AdoptionPolicy defines how the reconciler reacts if a dependent object exists but has no or a different owner.
type AdoptionPolicy string

const (
	// Fail if the dependent object exists but has no or a different owner.
	AdoptionPolicyNever AdoptionPolicy = "Never"
	// Adopt existing dependent objects if they have no owner set.
	AdoptionPolicyIfUnowned AdoptionPolicy = "IfUnowned"
	// Adopt existing dependent objects, even if they have a conflicting owner.
	AdoptionPolicyAlways AdoptionPolicy = "Always"
)

// ReconcilePolicy defines when the reconciler will reconcile the dependent object.
type ReconcilePolicy string

const (
	// Reconcile the dependent object if its manifest, as produced by the generator, changes.
	ReconcilePolicyOnObjectChange ReconcilePolicy = "OnObjectChange"
	// Reconcile the dependent object if its manifest, as produced by the generator, changes, or if the owning
	// component changes (identified by a change of its digest, including references).
	ReconcilePolicyOnObjectOrComponentChange ReconcilePolicy = "OnObjectOrComponentChange"
	// Reconcile the dependent object only once; afterwards it will never be touched again by the reconciler.
	ReconcilePolicyOnce ReconcilePolicy = "Once"
)

// UpdatePolicy defines how the reconciler will update dependent objects.
type UpdatePolicy string

const (
	// Recreate (that is: delete and create) existing dependent objects.
	UpdatePolicyRecreate UpdatePolicy = "Recreate"
	// Replace existing dependent objects.
	UpdatePolicyReplace UpdatePolicy = "Replace"
	// Use server side apply to update existing dependents.
	UpdatePolicySsaMerge UpdatePolicy = "SsaMerge"
	// Use server side apply to update existing dependents and, in addition, reclaim fields owned by certain
	// field owners, such as kubectl or helm.
	UpdatePolicySsaOverride UpdatePolicy = "SsaOverride"
)

// DeletePolicy defines how the reconciler will delete dependent objects.
type DeletePolicy string

const (
	// Delete dependent objects.
	DeletePolicyDelete DeletePolicy = "Delete"
	// Orphan dependent objects; that is, always, both if they become redundant when the component is applied,
	// and if the component is deleted.
	DeletePolicyOrphan DeletePolicy = "Orphan"
	// Orphan dependent objects if they become redundant when the compponent is applied.
	DeletePolicyOrphanOnApply DeletePolicy = "OrphanOnApply"
	// Orphan dependent objects if the component is deleted.
	DeletePolicyOrphanOnDelete DeletePolicy = "OrphanOnDelete"
)

// MissingNamespacesPolicy defines what the reconciler does if namespaces of dependent objects are not existing.
type MissingNamespacesPolicy string

const (
	// Do not create missing namespaces.
	MissingNamespacesPolicyDoNotCreate MissingNamespacesPolicy = "DoNotCreate"
	// Create missing namespaces.
	MissingNamespacesPolicyCreate MissingNamespacesPolicy = "Create"
)

// +kubebuilder:object:generate=true

// InventoryItem represents a dependent object managed by this operator.
type InventoryItem struct {
	// Type of the dependent object.
	TypeVersionInfo `json:",inline"`
	// Namespace and name of the dependent object.
	NameInfo `json:",inline"`
	// Adoption policy.
	AdoptionPolicy AdoptionPolicy `json:"adoptionPolicy"`
	// Reconcile policy.
	ReconcilePolicy ReconcilePolicy `json:"reconcilePolicy"`
	// Update policy.
	UpdatePolicy UpdatePolicy `json:"updatePolicy"`
	// Delete policy.
	DeletePolicy DeletePolicy `json:"deletePolicy"`
	// Apply order.
	ApplyOrder int `json:"applyOrder"`
	// Delete order.
	DeleteOrder int `json:"deleteOrder"`
	// Managed types.
	ManagedTypes []TypeVersionInfo `json:"managedTypes,omitempty"`
	// Digest of the descriptor of the dependent object.
	Digest string `json:"digest"`
	// Phase of the dependent object.
	Phase Phase `json:"phase,omitempty"`
	// Observed status of the dependent object.
	Status status.Status `json:"status,omitempty"`
	// Timestamp when this object was last applied.
	LastAppliedAt *metav1.Time `json:"lastAppliedAt,omitempty"`
}

type Phase string

const (
	PhaseScheduledForApplication = "ScheduledForApplication"
	PhaseScheduledForDeletion    = "ScheduledForDeletion"
	PhaseScheduledForCompletion  = "ScheduledForCompletion"
	PhaseCreating                = "Creating"
	PhaseUpdating                = "Updating"
	PhaseDeleting                = "Deleting"
	PhaseCompleting              = "Completing"
	PhaseReady                   = "Ready"
	PhaseCompleted               = "Completed"
)

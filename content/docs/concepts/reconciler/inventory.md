---
title: "The Inventory"
linkTitle: "The Inventory"
weight: 90
type: "docs"
description: >
  Anatomy and function of the inventory
---

The **inventory** is the reconciler's memory. It is what makes the engine level-based:
successive `Apply()` and `Delete()` calls consult and update it to know what has already
been done. It is a slice of `*InventoryItem`, one entry per managed object, and is
typically persisted in the status of a custom resource between reconciliations.

## `InventoryItem`

```go
package reconciler

// InventoryItem represents a dependent object managed by this operator.
type InventoryItem struct {
	// Type of the dependent object (Group, Version, Kind), inlined.
	TypeVersionInfo `json:",inline"`
	// Namespace and name of the dependent object, inlined.
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
	// Managed types (present for CRDs / APIServices).
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
```

The two embedded structs carry the object's identity:

```go
// TypeVersionInfo represents a Kubernetes type version.
type TypeVersionInfo struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

// NameInfo represents an object's namespace and name.
type NameInfo struct {
	Namespace string `json:"namespace,omitempty"` // empty for cluster-scoped objects
	Name      string `json:"name"`
}
```

## Field by field

| Field | Purpose |
|-------|---------|
| `Group` / `Version` / `Kind` | The object's type identity (from the embedded `TypeVersionInfo`). |
| `Namespace` / `Name` | The object's name and namespace (from the embedded `NameInfo`); `Namespace` is empty for cluster-scoped objects. |
| `AdoptionPolicy` | The effective [adoption policy](../policies/#adoption-policy--and-the-ownership-concept) for this object. |
| `ReconcilePolicy` | The effective [reconcile policy](../policies/#reconcile-policy). |
| `UpdatePolicy` | The effective [update policy](../policies/#update-policy). |
| `DeletePolicy` | The effective [delete policy](../policies/#delete-policy). This is the *last known* policy and governs the object even after it becomes redundant. |
| `ApplyOrder` | The [apply wave](../waves/#apply-waves) this object belongs to. |
| `DeleteOrder` | The [delete wave](../waves/#delete-waves) this object belongs to. |
| `ManagedTypes` | For a CRD or APIService, the extension type versions it introduces; drives the [managed-types](../managed-types/) handling. |
| `Digest` | The object's digest, used for [drift detection](../drift-detection/). An empty digest marks an object scheduled for deletion. |
| `Phase` | Where the object currently is in its lifecycle (see below). |
| `Status` | The last observed readiness status (a `status.Status`, kstatus-based). |
| `LastAppliedAt` | When the object was last applied; combined with the reapply interval to decide when to force-reapply. |

## Phases

The `Phase` field tracks the object's position in the apply/delete lifecycle:

```go
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
```

- **ScheduledForApplication → Creating / Updating → Ready** — the normal apply path.
- **ScheduledForCompletion → Completing → Completed** — the [purge](../completion/) path
  for objects with a purge order; the record stays as `Completed`.
- **ScheduledForDeletion → Deleting** — the removal path; once the object is gone the
  record is dropped from the inventory entirely.

## Ordering within the inventory

After each apply, the inventory is stored sorted into the canonical **deletion** order.
This means that if the object set ever needs to be deleted, the reconciler can simply
walk the inventory in order. It is also why the inventory must be passed back unchanged
between calls — the reconciler relies on its own bookkeeping being intact.

/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

const (
	LabelKeySuffixOwnerId              = "owner-id"
	AnnotationKeySuffixOwnerId         = "owner-id"
	AnnotationKeySuffixDigest          = "digest"
	AnnotationKeySuffixAdoptionPolicy  = "adoption-policy"
	AnnotationKeySuffixReconcilePolicy = "reconcile-policy"
	AnnotationKeySuffixUpdatePolicy    = "update-policy"
	AnnotationKeySuffixDeletePolicy    = "delete-policy"
	AnnotationKeySuffixReapplyInterval = "reapply-interval"
	AnnotationKeySuffixApplyOrder      = "apply-order"
	AnnotationKeySuffixPurgeOrder      = "purge-order"
	AnnotationKeySuffixDeleteOrder     = "delete-order"
	AnnotationKeySuffixStatusHint      = "status-hint"
	AnnotationKeySuffixDisableEvents   = "disable-events"
)

const (
	AdoptionPolicyNever     = "never"
	AdoptionPolicyIfUnowned = "if-unowned"
	AdoptionPolicyAlways    = "always"
)

const (
	ReconcilePolicyOnObjectChange            = "on-object-change"
	ReconcilePolicyOnObjectOrComponentChange = "on-object-or-component-change"
	ReconcilePolicyOnce                      = "once"
)

const (
	UpdatePolicyDefault     = "default"
	UpdatePolicyRecreate    = "recreate"
	UpdatePolicyReplace     = "replace"
	UpdatePolicySsaMerge    = "ssa-merge"
	UpdatePolicySsaOverride = "ssa-override"
)

const (
	DeletePolicyDefault        = "default"
	DeletePolicyDelete         = "delete"
	DeletePolicyOrphan         = "orphan"
	DeletePolicyOrphanOnApply  = "orphan-on-apply"
	DeletePolicyOrphanOnDelete = "orphan-on-delete"
)

const (
	StatusHintHasObservedGeneration = "has-observed-generation"
	StatusHintHasReadyCondition     = "has-ready-condition"
	StatusHintConditions            = "conditions"
)

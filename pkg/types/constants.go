/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

const (
	LabelKeySuffixOwnerId              = "owner-id"
	AnnotationKeySuffixDigest          = "digest"
	AnnotationKeySuffixReconcilePolicy = "reconcile-policy"
	AnnotationKeySuffixUpdatePolicy    = "update-policy"
	AnnotationKeySuffixOrder           = "order"
	AnnotationKeySuffixPurgeOrder      = "purge-order"
	AnnotationKeySuffixOwnerId         = "owner-id"
)

const (
	ReconcilePolicyOnObjectChange            = "on-object-change"
	ReconcilePolicyOnObjectOrComponentChange = "on-object-or-component-change"
	ReconcilePolicyOnce                      = "once"
)

const (
	UpdatePolicyDefault  = "default"
	UpdatePolicyRecreate = "recreate"
)

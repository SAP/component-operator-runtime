/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"context"
	"encoding/json"

	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This is a workaround. Ideally Kubernetes would have a finalizers subresource, and controller-runtime
// would offer the according subresource client.
func UpdateFinalizers(ctx context.Context, clnt client.Client, obj client.Object, fieldOwner string) error {
	// note: it's crucial to use string keyed maps to construct the patch; using metav1.ObjectMeta would lead to
	// finalizers to be removed entirely from the patch when empty (because of omitempty usage there);
	// but we need to preserve empty or null finalizers in the patch, in order to make JSON merge patch apply it correctly
	finalizerPatch := map[string]any{
		"metadata": map[string]any{
			"resourceVersion": obj.GetResourceVersion(),
			"finalizers":      obj.GetFinalizers(),
		},
	}
	// note: this must() is ok because marshalling finalizers should always work
	return clnt.Patch(ctx, obj, client.RawPatch(apitypes.MergePatchType, must(json.Marshal(finalizerPatch))), client.FieldOwner(fieldOwner))
}

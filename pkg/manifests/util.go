/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// TODO: consolidate all the util files into an internal reuse package

// Deep-merge two maps with the usual logic and return the result.
// The first map (x) must be deeply JSON (i.e. consist deeply of JSON values only).
// The maps given as input will not be changed.
// Both maps can be passed as nil.
func MergeMaps(x, y map[string]any) map[string]any {
	if x == nil {
		x = make(map[string]any)
	} else {
		x = runtime.DeepCopyJSON(x)
	}
	MergeMapInto(x, y)
	return x
}

// Deep-merge second map (y) over first map (x) with the usual logic.
// The first map will be changed (unless y is empty or nil), the second map will not be changed.
// The first map must not be nil, the second map is allowed to be nil.
func MergeMapInto(x map[string]any, y map[string]any) {
	for k := range y {
		if _, ok := x[k]; ok {
			if v, ok := x[k].(map[string]any); ok {
				if w, ok := y[k].(map[string]any); ok {
					MergeMapInto(v, w)
				} else {
					x[k] = w
				}
			} else {
				x[k] = y[k]
			}
		} else {
			x[k] = y[k]
		}
	}
}

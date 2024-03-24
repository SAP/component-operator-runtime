/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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
func MergeMaps(x, y map[string]any) map[string]any {
	if x == nil {
		x = make(map[string]any)
	} else {
		x = runtime.DeepCopyJSON(x)
	}
	for k, w := range y {
		if v, ok := x[k]; ok {
			if v, ok := v.(map[string]any); ok {
				if _w, ok := w.(map[string]any); ok {
					x[k] = MergeMaps(v, _w)
				} else {
					x[k] = w
				}
			} else {
				x[k] = w
			}
		} else {
			x[k] = w
		}
	}
	return x
}

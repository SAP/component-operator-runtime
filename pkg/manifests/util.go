/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manifests

import (
	"k8s.io/apimachinery/pkg/runtime"
)

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

/*
{{- if .spdxLicenseHeaders }}
SPDX-FileCopyrightText: {{ now.Year }} {{ .owner }}
SPDX-License-Identifier: Apache-2.0
{{- else }}
Copyright {{ now.Year }} {{ .owner }}.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
{{- end }}
*/

// +kubebuilder:object:generate=true
// +groupName={{ .groupName }}

// Package {{ .groupVersion }} contains API Schema definitions for the {{ .groupName | splitList "." | first }} {{ .groupVersion }} API group.
package {{ .groupVersion }}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "{{ .groupName }}", Version: "{{ .groupVersion }}"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	// Needed by kubernetes/code-generator.
	SchemeGroupVersion = GroupVersion
)

// Needed by kubernetes/code-generator.
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}

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

package {{ .groupVersion }}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"
)

// {{ .kind }}Spec defines the desired state of {{ .kind }}.
type {{ .kind }}Spec struct {
	// Uncomment the following if you want to implement the PlacementConfiguration interface
	// (that is, want to make deployment namespace and name configurable here in the spec, independently of
	// the component's metadata.namespace and metadata.name).
	// component.PlacementSpec                  `json:",inline"`
	// Uncomment the following if you want to implement the ClientConfiguration interface
	// (that is, want to allow remote deployments via a specified kubeconfig).
	// Note, that when implementing the ClientConfiguration interface, then also the PlacementConfiguration
	// interface should be implemented.
	// component.ClientSpec        `json:",inline"`
	// Uncomment the following if you want to implement the ImpersonationConfiguratio interface
	// (that is, want to allow use a specified service account in the target namespace for the deployment).
	// component.ImpersonationSpec `json:",inline"`
	// Uncomment the following if you want to implement the RequeueConfiguration interface
	// (that is, want to allow to override the default requeue interval of 10m).
	// component.RequeueSpec `json:",inline"`
	// Uncomment the following if you want to implement the RetryConfiguration interface
	// (that is, want to allow to override the default retry interval, which equals the effective requeue interval).
	// component.RequeueSpec `json:",inline"`

	// Add your own fields here, describing the deployment of the managed component.
}

// {{ .kind }}Status defines the observed state of {{ .kind }}.
type {{ .kind }}Status struct {
	component.Status `json:",inline"`

	// You may add your own fields here; this is rarely needed.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// {{ .kind }} is the Schema for the {{ .resource }} API.
type {{ .kind }} struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec {{ .kind }}Spec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status {{ .kind }}Status `json:"status,omitempty"`
}

var _ component.Component = &{{ .kind }}{}

// +kubebuilder:object:root=true

// {{ .kind }}List contains a list of {{ .kind }}.
type {{ .kind }}List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []{{ .kind }} `json:"items"`
}

func (s *{{ .kind }}Spec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

func (c *{{ .kind }}) GetSpec() componentoperatorruntimetypes.Unstructurable {
	return &c.Spec
}

func (c *{{ .kind }}) GetStatus() *component.Status {
	return &c.Status.Status
}

func init() {
	SchemeBuilder.Register(&{{ .kind }}{}, &{{ .kind }}List{})
}

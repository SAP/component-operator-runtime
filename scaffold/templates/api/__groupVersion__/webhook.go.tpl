{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
/*
Copyright 2023 SAP SE.

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

package {{ .groupVersion }}

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/sap/admission-webhook-runtime/pkg/admission"
)

// +kubebuilder:object:generate=false
type Webhook struct {
}

{{- if .validatingWebhookEnabled }}

var _ admission.ValidatingWebhook[*{{ .kind }}] = &Webhook{}
{{- end }}
{{- if .mutatingWebhookEnabled }}

var _ admission.MutatingWebhook[*{{ .kind }}] = &Webhook{}
{{- end }}

func NewWebhook() *Webhook {
	return &Webhook{}
}

{{- if .validatingWebhookEnabled }}

func (w *Webhook) ValidateCreate(ctx context.Context, component *{{ .kind }}) error {
	return nil
}

func (w *Webhook) ValidateUpdate(ctx context.Context, oldComponent *{{ .kind }}, component *{{ .kind }}) error {
	return nil
}

func (w *Webhook) ValidateDelete(ctx context.Context, component *{{ .kind }}) error {
	return nil
}
{{- end }}

{{- if .mutatingWebhookEnabled }}

func (w *Webhook) MutateCreate(ctx context.Context, component *{{ .kind }}) error {
	return nil
}

func (w *Webhook) MutateUpdate(ctx context.Context, oldComponent *{{ .kind }}, component *{{ .kind }}) error {
	return nil
}
{{- end }}

func (w *Webhook) SetupWithManager(mgr manager.Manager) {
	{{- if .validatingWebhookEnabled }}
	mgr.GetWebhookServer().Register(
		fmt.Sprintf("/admission/%s/{{ .resource }}/validate", GroupVersion),
		admission.NewValidatingWebhookHandler[*{{ .kind }}](w, mgr.GetScheme(), mgr.GetLogger().WithName("webhook-runtime")),
	)
	{{- end }}
	{{- if .mutatingWebhookEnabled }}
	mgr.GetWebhookServer().Register(
		fmt.Sprintf("/admission/%s/{{ .resource }}/mutate", GroupVersion),
		admission.NewMutatingWebhookHandler[*{{ .kind }}](w, mgr.GetScheme(), mgr.GetLogger().WithName("webhook-runtime")),
	)
	{{- end }}
}
{{- end }}

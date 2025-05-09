//go:build tools
// +build tools

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

package tools

import (
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/controller-runtime/tools/setup-envtest"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)

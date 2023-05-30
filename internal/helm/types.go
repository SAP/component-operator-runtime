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

package helm

type ChartData struct {
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	Type       string `json:"type,omitempty"`
	AppVersion string `json:"appVersion,omitempty"`
}

const (
	ChartTypeApplication = "application"
	ChartTypeLibrary     = "library"
)

type TemplateData struct {
	Name     string `json:"name,omitempty"`
	BasePath string `json:"basePath,omitempty"`
}

type ReleaseData struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Service   string `json:"service,omitempty"`
	IsInstall bool   `json:"isInstall,omitempty"`
	IsUpgrade bool   `json:"isUpgrade,omitempty"`
}

type CapabilitiesData struct {
	KubeVersion KubeVersionData `json:"kubeVersion,omitempty"`
	APIVersions ApiVersionsData `json:"apiVersions,omitempty"`
}

type ApiVersionsData []string

func (apiVersions ApiVersionsData) Has(version string) bool {
	for _, v := range apiVersions {
		if v == version {
			return true
		}
	}
	return false
}

type KubeVersionData struct {
	Version string `json:"version,omitempty"`
	Major   string `json:"major,omitempty"`
	Minor   string `json:"minor,omitempty"`
	// GitVersion is actually deprecated, but some charts still use it
	GitVersion string `json:"gitVersion,omitempty"`
}

func (kubeVersion *KubeVersionData) String() string {
	return kubeVersion.Version
}

type HookMetadata struct {
	Types          []string
	Weight         int
	DeletePolicies []string
}

const (
	HookTypePreInstall   = "pre-install"
	HookTypePostInstall  = "post-install"
	HookTypePreUpgrade   = "pre-upgrade"
	HookTypePostUpgrade  = "post-upgrade"
	HookTypePreDelete    = "pre-delete"
	HookTypePostDelete   = "post-delete"
	HookTypePreRollback  = "pre-rollback"
	HookTypePostRollback = "post-rollback"
	HookTypeTest         = "test"
	HookTypeTestSuccess  = "test-success"
)

const (
	HookMinWeight = -100
	HookMaxWeight = 100
)

const (
	HookDeletePolicyBeforeHookCreation = "before-hook-creation"
	HookDeletePolicyHookSucceeded      = "hook-succeeded"
	HookDeletePolicyHookFailed         = "hook-failed"
)

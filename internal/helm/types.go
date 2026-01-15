/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"encoding/json"
	"fmt"
)

// +kubebuilder:object:generate=true

type ChartMetadata struct {
	Name         string            `json:"name,omitempty"`
	Version      string            `json:"version,omitempty"`
	Type         string            `json:"type,omitempty"`
	AppVersion   string            `json:"appVersion,omitempty"`
	Dependencies []ChartDependency `json:"dependencies,omitempty"`
}

const (
	ChartTypeApplication = "application"
	ChartTypeLibrary     = "library"
)

// +kubebuilder:object:generate=true

type ChartDependency struct {
	Name         string             `json:"name,omitempty"`
	Alias        string             `json:"alias,omitempty"`
	Condition    string             `json:"condition,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	ImportValues []ChartImportValue `json:"import-values,omitempty"`
}

// +kubebuilder:object:generate=true

type ChartImportValue struct {
	Child  string `json:"child"`
	Parent string `json:"parent"`
}

func (v *ChartImportValue) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		v.Child = fmt.Sprintf("exports.%s", s)
	} else {
		type chartImportValue ChartImportValue
		w := chartImportValue(*v)
		if err := json.Unmarshal(b, &w); err != nil {
			return err
		}
		*v = ChartImportValue(w)
	}
	return nil
}

// +kubebuilder:object:generate=true

type Template struct {
	Name     string `json:"name,omitempty"`
	BasePath string `json:"basePath,omitempty"`
}

// +kubebuilder:object:generate=true

type Release struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Service   string `json:"service,omitempty"`
	IsInstall bool   `json:"isInstall,omitempty"`
	IsUpgrade bool   `json:"isUpgrade,omitempty"`
	Revision  int64  `json:"revision"`
}

// +kubebuilder:object:generate=true

type Capabilities struct {
	KubeVersion KubeVersion `json:"kubeVersion,omitempty"`
	APIVersions ApiVersions `json:"apiVersions,omitempty"`
}

// +kubebuilder:object:generate=true

type KubeVersion struct {
	Version string `json:"version,omitempty"`
	Major   string `json:"major,omitempty"`
	Minor   string `json:"minor,omitempty"`
	// GitVersion is actually deprecated, but some charts still use it
	GitVersion string `json:"gitVersion,omitempty"`
}

func (kubeVersion *KubeVersion) String() string {
	return kubeVersion.Version
}

// +kubebuilder:object:generate=true

type ApiVersions []string

func (apiVersions ApiVersions) Has(version string) bool {
	for _, v := range apiVersions {
		if v == version {
			return true
		}
	}
	return false
}

type ResourceMetadata struct {
	Policy string
}

const (
	ResourcePolicyKeep = "keep"
)

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

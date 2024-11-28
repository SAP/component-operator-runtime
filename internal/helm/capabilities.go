/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"github.com/sap/go-generics/slices"

	"k8s.io/client-go/discovery"
)

func GetCapabilities(discoveryClient discovery.DiscoveryInterface) (*Capabilities, error) {
	kubeVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}
	var apiVersions []string
	_, apiResourceLists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}
	for _, apiResourceList := range apiResourceLists {
		apiVersions = append(apiVersions, apiResourceList.GroupVersion)
		for _, apiResource := range apiResourceList.APIResources {
			apiVersions = append(apiVersions, apiResourceList.GroupVersion+"/"+apiResource.Kind)
		}
	}
	capabilities := &Capabilities{
		KubeVersion: KubeVersion{
			Version:    kubeVersion.GitVersion,
			Major:      kubeVersion.Major,
			Minor:      kubeVersion.Minor,
			GitVersion: kubeVersion.GitVersion,
		},
		APIVersions: slices.Uniq(apiVersions),
	}
	return capabilities, nil
}

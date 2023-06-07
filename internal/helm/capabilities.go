/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"github.com/sap/go-generics/slices"

	"k8s.io/client-go/discovery"
)

func GetCapabilities(client discovery.DiscoveryInterface) (*CapabilitiesData, error) {
	kubeVersion, err := client.ServerVersion()
	if err != nil {
		return nil, err
	}
	var apiVersions []string
	_, apiResourceLists, err := client.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}
	for _, apiResourceList := range apiResourceLists {
		apiVersions = append(apiVersions, apiResourceList.GroupVersion)
		for _, apiResource := range apiResourceList.APIResources {
			apiVersions = append(apiVersions, apiResourceList.GroupVersion+"/"+apiResource.Kind)
		}
	}
	capabilities := &CapabilitiesData{
		KubeVersion: KubeVersionData{
			Version:    kubeVersion.GitVersion,
			Major:      kubeVersion.Major,
			Minor:      kubeVersion.Minor,
			GitVersion: kubeVersion.GitVersion,
		},
		APIVersions: slices.Uniq(apiVersions),
	}
	return capabilities, nil
}

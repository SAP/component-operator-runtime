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
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		},
		APIVersions: slices.Uniq(apiVersions),
	}
	return capabilities, nil
}

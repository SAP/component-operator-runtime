/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package environment

import (
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func buildKubeConfig(cfg *rest.Config) *clientcmdapi.Config {
	apiConfig := clientcmdapi.NewConfig()

	apiConfig.Clusters["envtest"] = clientcmdapi.NewCluster()
	cluster := apiConfig.Clusters["envtest"]
	cluster.Server = cfg.Host
	cluster.CertificateAuthorityData = cfg.CAData

	apiConfig.AuthInfos["envtest"] = clientcmdapi.NewAuthInfo()
	authInfo := apiConfig.AuthInfos["envtest"]
	authInfo.ClientKeyData = cfg.KeyData
	authInfo.ClientCertificateData = cfg.CertData

	apiConfig.Contexts["envtest"] = clientcmdapi.NewContext()
	context := apiConfig.Contexts["envtest"]
	context.Cluster = "envtest"
	context.AuthInfo = "envtest"

	apiConfig.CurrentContext = "envtest"

	return apiConfig
}

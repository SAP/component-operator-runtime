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

package main

{{- $webhooksEnabled := or .validatingWebhookEnabled .mutatingWebhookEnabled }}

import (
	"flag"
	{{- if $webhooksEnabled }}
	"net"
	{{- end }}
	"os"
	{{- if $webhooksEnabled }}
	"strconv"
	{{- end }}

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	{{- if $webhooksEnabled }}
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	{{- end }}

	"{{ .goModule }}/pkg/operator"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(apiregistrationv1.AddToScheme(scheme))

	operator.InitScheme(scheme)
}

func main() {
	var metricsAddr string
	var probeAddr string
	{{- if $webhooksEnabled }}
	var webhookAddr string
	var webhookCertDir string
	{{- else }}
	// Uncomment the following lines to enable webhooks.
	// var webhookAddr string
	// var webhookCertDir string
	{{- end }}
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	{{- if $webhooksEnabled }}
	flag.StringVar(&webhookAddr, "webhook-bind-address", ":2443",
		"The address the webhooks endpoint binds to.")
	flag.StringVar(&webhookCertDir, "webhook-tls-directory", "",
		"The directory containing tls server key and certificate, as tls.key and tls.crt; defaults to $TMPDIR/k8s-webhook-server/serving-certs")
	{{- else }}
	// Uncomment the following lines to enable webhooks.
	// flag.StringVar(&webhookAddr, "webhook-bind-address", ":2443",
	//	"The address the webhooks endpoint binds to.")
	// flag.StringVar(&webhookCertDir, "webhook-tls-directory", "",
	//	"The directory containing tls server key and certificate, as tls.key and tls.crt; defaults to $TMPDIR/k8s-webhook-server/serving-certs")
	{{- end }}
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	operator.InitFlags(flag.CommandLine)
	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if err := operator.ValidateFlags(); err != nil {
		setupLog.Error(err, "error validating command line flags")
		os.Exit(1)
	}

	{{- if $webhooksEnabled }}

	webhookHost, webhookPort, err := parseAddress(webhookAddr)
	if err != nil {
		setupLog.Error(err, "error parsing webhook address")
		os.Exit(1)
	}
	{{- else }}

	// Uncomment the following lines to enable webhooks.
	// webhookHost, webhookPort, err := parseAddress(webhookAddr)
	// if err != nil {
	//	setupLog.Error(err, "error parsing webhook address")
	//	os.Exit(1)
	// }
	{{- end }}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: append(operator.GetUncacheableTypes(), &apiextensionsv1.CustomResourceDefinition{}, &apiregistrationv1.APIService{}),
			},
		},
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              operator.GetName(),
		LeaderElectionReleaseOnCancel: true,
		{{- if $webhooksEnabled }}
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookHost,
			Port:    webhookPort,
			CertDir: webhookCertDir,
		}),
		{{- else }}
		// Uncomment the following lines to enable webhooks.
		// WebhookServer: webhook.NewServer(webhook.Options{
		// 	Host:    webhookHost,
		//	Port:    webhookPort,
		//	CertDir: webhookCertDir,
		// }),
		{{- end }}
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Uncomment to enable conversion webhook (in case additional api versions are added in ./api).
	// Note: to make conversion work, additional changes are necessary:
	// - additional api versions have to be added to InitScheme() in pkg/operator/operator.go
	// - one of the api versions has to marked as Hub, all other versions need to implement the
	//   conversion.Convertible interface (see https://book.kubebuilder.io/multiversion-tutorial/conversion.html)
	// - one of the api versions has to be marked as storage version (+kubebuilder:storageversion)
	// - the crd resource has to be enhanced with a conversion section, telling the Kubernetes API server how to
	//   connect to the conversion endpoint.
	// mgr.GetWebhookServer().Register("/convert", conversion.NewWebhookHandler(mgr.GetScheme()))

	if err := operator.Setup(mgr); err != nil {
		setupLog.Error(err, "error registering controller with manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

{{- if $webhooksEnabled }}

func parseAddress(address string) (string, int, error) {
	host, p, err := net.SplitHostPort(address)
	if err != nil {
		return "", -1, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return "", -1, err
	}
	return host, port, nil
}
{{- else }}

// Uncomment the following lines to enable webhooks.
// func parseAddress(address string) (string, int, error) {
//	host, p, err := net.SplitHostPort(address)
//	if err != nil {
//		return "", -1, err
//	}
//	port, err := strconv.Atoi(p)
//	if err != nil {
//		return "", -1, err
//	}
//	return host, port, nil
// }
{{- end }}

/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sap/go-generics/slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/internal/clientfactory"
	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
)

// TODO: consolidate all the util files into an internal reuse package

func ref[T any](x T) *T {
	return &x
}

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func getClient(kubeConfigPath string) (cluster.Client, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("no kubeconfig was specified")
	}
	kubeConfig, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(apiregistrationv1.AddToScheme(scheme))
	return clientfactory.NewClientFor(config, scheme, fullName)
}

func isEphmeralError(err error) bool {
	if apierrors.IsConflict(err) {
		return true
	}
	return false
}

func formatTimestamp(t time.Time) string {
	d := time.Since(t)
	if d > 24*time.Hour {
		return fmt.Sprintf("%dd", d/24*time.Hour)
	} else if d > time.Hour {
		return fmt.Sprintf("%dh", d/time.Hour)
	} else if d > time.Minute {
		return fmt.Sprintf("%dm", d/time.Minute)
	} else {
		return fmt.Sprintf("%ds", d/time.Second)
	}
}

type releaseDetails struct {
	Namespace           string `json:"namespace"`
	Name                string `json:"name"`
	Revision            int64  `json:"revision"`
	State               string `json:"state"`
	NumAllObjects       int64  `json:"numAllObjects"`
	NumReadyObjects     int64  `json:"numReadyObjects"`
	NumCompletedObjects int64  `json:"numCompletedObjects"`
	CreatedAt           string `json:"createdAt"`
	LastUpdatedAt       string `json:"lastUpdatedAt"`
}

func getReleaseDetails(release *release.Release) *releaseDetails {
	return &releaseDetails{
		Namespace:           release.GetNamespace(),
		Name:                release.GetName(),
		Revision:            release.Revision,
		State:               string(release.State),
		NumAllObjects:       int64(len(release.Inventory)),
		NumReadyObjects:     int64(slices.Count(release.Inventory, func(item *reconciler.InventoryItem) bool { return item.Phase == reconciler.PhaseReady })),
		NumCompletedObjects: int64(slices.Count(release.Inventory, func(item *reconciler.InventoryItem) bool { return item.Phase == reconciler.PhaseCompleted })),
		CreatedAt:           formatTimestamp(*release.GetCreationTimestamp()),
		LastUpdatedAt:       formatTimestamp(*release.GetUpdateTimestamp()),
	}
}

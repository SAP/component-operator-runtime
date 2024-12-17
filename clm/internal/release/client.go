/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package release

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	labelKeyRelease = "release.clm.cs.sap.com"
)

type Client struct {
	name      string
	finalizer string
	client    client.Client
}

func NewClient(name string, clnt client.Client) *Client {
	return &Client{
		name:      name,
		finalizer: fmt.Sprintf("%s/finalizer", name),
		client:    clnt,
	}
}

func (c *Client) Get(ctx context.Context, namespace string, name string) (*Release, error) {
	release := &Release{
		namespace: namespace,
		name:      name,
		configMap: &corev1.ConfigMap{},
	}
	if err := c.client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: c.configMapName(name)}, release.configMap); err != nil {
		return nil, err
	}
	if err := release.importData(); err != nil {
		return nil, err
	}
	return release, nil
}

func (c *Client) List(ctx context.Context, namespace string) ([]*Release, error) {
	configMapList := &corev1.ConfigMapList{}
	if err := c.client.List(ctx, configMapList, client.InNamespace(namespace), client.HasLabels{labelKeyRelease}); err != nil {
		return nil, err
	}
	releases := make([]*Release, len(configMapList.Items))
	for i := 0; i < len(configMapList.Items); i++ {
		configMap := &configMapList.Items[i]
		releases[i] = &Release{
			namespace: configMap.Namespace,
			name:      c.releaseName(configMap.Name),
			configMap: configMap,
		}
		if err := releases[i].importData(); err != nil {
			return nil, err
		}
	}
	return releases, nil
}

func (c *Client) Create(ctx context.Context, namespace string, name string) (*Release, error) {
	now := time.Now()

	release := &Release{
		namespace:         namespace,
		name:              name,
		creationTimestamp: &now,
		updateTimestamp:   &now,
		configMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      c.configMapName(name),
				Labels: map[string]string{
					labelKeyRelease: name,
				},
				Finalizers: []string{c.finalizer},
			},
		},
	}
	if err := release.exportData(); err != nil {
		return nil, err
	}
	if err := c.client.Create(ctx, release.configMap); err != nil {
		return nil, err
	}
	return release, nil
}

func (c *Client) Update(ctx context.Context, release *Release) error {
	if release.configMap.UID == "" || release.configMap.ResourceVersion == "" {
		return fmt.Errorf("error updating release %s/%s: empty uid or resource version", release.GetNamespace(), release.GetName())
	}
	if !release.configMap.DeletionTimestamp.IsZero() && len(release.Inventory) == 0 {
		controllerutil.RemoveFinalizer(release.configMap, c.finalizer)
	}
	now := time.Now()
	release.updateTimestamp = &now
	if err := release.exportData(); err != nil {
		return err
	}
	return c.client.Update(ctx, release.configMap)
}

func (c *Client) Delete(ctx context.Context, release *Release) error {
	if release.configMap.UID == "" || release.configMap.ResourceVersion == "" {
		return fmt.Errorf("error deleting release %s/%s: empty uid or resource version", release.GetNamespace(), release.GetName())
	}
	return c.client.Delete(ctx, release.configMap, client.Preconditions{ResourceVersion: &release.configMap.ResourceVersion})
}

func (c *Client) configMapName(releaseName string) string {
	return fmt.Sprintf("%s.release.%s", strings.Join(slices.Reverse(strings.Split(c.name, ".")), "."), releaseName)
}

func (c *Client) releaseName(configMapName string) string {
	return configMapName[strings.LastIndex(configMapName, ".")+1:]
}

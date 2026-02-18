/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"context"
	"net/http"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

func NewClient(clnt client.Client, discoveryClient discovery.DiscoveryInterface, eventRecorder record.EventRecorder, config *rest.Config, httpClient *http.Client) Client {
	return &clientImpl{
		Client:          clnt,
		discoveryClient: discoveryClient,
		eventRecorder:   eventRecorder,
		config:          config,
		httpClient:      httpClient,
		inflightRetries: make(map[apitypes.UID]time.Time),
	}
}

const (
	retryAfter         = time.Second
	nextRetryNotBefore = time.Minute
)

type clientImpl struct {
	client.Client
	discoveryClient discovery.DiscoveryInterface
	eventRecorder   record.EventRecorder
	config          *rest.Config
	httpClient      *http.Client
	mu              sync.Mutex
	inflightRetries map[apitypes.UID]time.Time
}

func (c *clientImpl) DiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}

func (c *clientImpl) EventRecorder() record.EventRecorder {
	return c.eventRecorder
}

func (c *clientImpl) Config() *rest.Config {
	return c.config
}

func (c *clientImpl) HttpClient() *http.Client {
	return c.httpClient
}

func (c *clientImpl) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	return c.Client.Apply(ctx, obj, opts...)
}

func (c *clientImpl) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return c.retryIfEligible(c.Client.Create(ctx, obj, opts...), obj.GetUID())
}

func (c *clientImpl) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return c.retryIfEligible(c.Client.Delete(ctx, obj, opts...), obj.GetUID())
}

func (c *clientImpl) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return c.retryIfEligible(c.Client.Update(ctx, obj, opts...), obj.GetUID())
}

func (c *clientImpl) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.retryIfEligible(c.Client.Patch(ctx, obj, patch, opts...), obj.GetUID())
}

func (c *clientImpl) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

func (c *clientImpl) Status() client.SubResourceWriter {
	return &subResourceClientImpl{SubResourceClient: c.Client.SubResource("status"), client: c}
}

func (c *clientImpl) SubResource(subResource string) client.SubResourceClient {
	return &subResourceClientImpl{SubResourceClient: c.Client.SubResource(subResource), client: c}
}

func (c *clientImpl) retryIfEligible(err error, uid apitypes.UID) error {
	if apierrors.IsConflict(err) && uid != "" {
		c.mu.Lock()
		defer c.mu.Unlock()
		now := time.Now()
		for uid, notBefore := range c.inflightRetries {
			if notBefore.After(now) {
				delete(c.inflightRetries, uid)
			}
		}
		if _, ok := c.inflightRetries[uid]; !ok {
			c.inflightRetries[uid] = now.Add(nextRetryNotBefore)
			return types.NewRetriableError(err, ref(retryAfter))
		}
	}
	return err
}

type subResourceClientImpl struct {
	client.SubResourceClient
	client *clientImpl
}

func (s *subResourceClientImpl) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return s.client.retryIfEligible(s.SubResourceClient.Create(ctx, obj, subResource, opts...), obj.GetUID())
}

func (s *subResourceClientImpl) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return s.client.retryIfEligible(s.SubResourceClient.Update(ctx, obj, opts...), obj.GetUID())
}

func (s *subResourceClientImpl) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return s.client.retryIfEligible(s.SubResourceClient.Patch(ctx, obj, patch, opts...), obj.GetUID())
}

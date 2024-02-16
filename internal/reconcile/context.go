/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconcile

import (
	"context"
	"fmt"

	"github.com/sap/component-operator-runtime/internal/cluster"
)

type reconcilerNameContextKey struct{}
type clientContextKey struct{}
type componentDigestContextKey struct{}

// Create new context (i.e. wrap a context.Context into a manifests.Context).
func NewContext(ctx context.Context) Context {
	return &contextImpl{Context: ctx}
}

// Create new context with reconciler name added as value.
func NewContextWithReconcilerName(ctx context.Context, reconcilerName string) Context {
	return NewContext(ctx).WithReconcilerName(reconcilerName)
}

// Create new context with client added as value.
func NewContextWithClient(ctx context.Context, client cluster.Client) Context {
	return NewContext(ctx).WithClient(client)
}

// Create new context with component digest added as value.
func NewContextWithComponentDigest(ctx context.Context, componentDigest string) Context {
	return NewContext(ctx).WithComponentDigest(componentDigest)
}

// Retrieve reconciler name from given context.
func ReconcilerNameFromContext(ctx context.Context) (string, error) {
	if reconcilerName, ok := ctx.Value(reconcilerNameContextKey{}).(string); ok {
		return reconcilerName, nil
	}
	return "", fmt.Errorf("reconciler name not found in context")
}

// Retrieve client from given context.
func ClientFromContext(ctx context.Context) (cluster.Client, error) {
	if client, ok := ctx.Value(clientContextKey{}).(cluster.Client); ok {
		return client, nil
	}
	return nil, fmt.Errorf("client not found in context")
}

// Retrieve component digest from given context.
func ComponentDigestFromContext(ctx context.Context) (string, error) {
	if componentDigest, ok := ctx.Value(componentDigestContextKey{}).(string); ok {
		return componentDigest, nil
	}
	return "", fmt.Errorf("component digest not found in context")
}

type contextImpl struct {
	context.Context
}

var _ Context = &contextImpl{}

// Return new context with given reconciler name added as value.
func (c *contextImpl) WithReconcilerName(reconcilerName string) Context {
	return &contextImpl{Context: context.WithValue(c, reconcilerNameContextKey{}, reconcilerName)}
}

// Return new context with given client added as value.
func (c *contextImpl) WithClient(client cluster.Client) Context {
	return &contextImpl{Context: context.WithValue(c, clientContextKey{}, client)}
}

// Return new context with given component digest as value.
func (c *contextImpl) WithComponentDigest(componentDigest string) Context {
	return &contextImpl{Context: context.WithValue(c, componentDigestContextKey{}, componentDigest)}
}

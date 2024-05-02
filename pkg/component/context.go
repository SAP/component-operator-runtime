/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

type (
	reconcilerNameContextKeyType  struct{}
	clientContextKeyType          struct{}
	componentContextKeyType       struct{}
	componentDigestContextKeyType struct{}
)

var (
	reconcilerNameContextKey  = reconcilerNameContextKeyType{}
	clientContextKey          = clientContextKeyType{}
	componentContextKey       = componentContextKeyType{}
	componentDigestContextKey = componentDigestContextKeyType{}
)

func newContext(ctx context.Context) *reconcileContext {
	return &reconcileContext{Context: ctx}
}

type reconcileContext struct {
	context.Context
}

func (c *reconcileContext) WithReconcilerName(reconcilerName string) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, reconcilerNameContextKey, reconcilerName)}
}

func (c *reconcileContext) WithClient(clnt cluster.Client) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, clientContextKey, clnt)}
}

func (c *reconcileContext) WithComponent(component Component) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, componentContextKey, component)}
}

func (c *reconcileContext) WithComponentDigest(componentDigest string) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, componentDigestContextKey, componentDigest)}
}

func ReconcilerNameFromContext(ctx context.Context) (string, error) {
	if reconcilerName, ok := ctx.Value(reconcilerNameContextKey).(string); ok {
		return reconcilerName, nil
	}
	return "", fmt.Errorf("reconciler name not found in context")
}

func ClientFromContext(ctx context.Context) (cluster.Client, error) {
	if clnt, ok := ctx.Value(clientContextKey).(cluster.Client); ok {
		return clnt, nil
	}
	return nil, fmt.Errorf("client not found in context")
}

func ComponentFromContext(ctx context.Context) (Component, error) {
	if component, ok := ctx.Value(componentContextKey).(Component); ok {
		return component, nil
	}
	return nil, fmt.Errorf("component not found in context")
}

func ComponentDigestFromContext(ctx context.Context) (string, error) {
	if componentDigest, ok := ctx.Value(componentDigestContextKey).(string); ok {
		return componentDigest, nil
	}
	return "", fmt.Errorf("component digest not found in context")
}

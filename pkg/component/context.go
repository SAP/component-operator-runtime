/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

type (
	reconcilerNameContextKeyType     struct{}
	clientContextKeyType             struct{}
	componentContextKeyType          struct{}
	componentNameContextKeyType      struct{}
	componentNamespaceContextKeyType struct{}
	componentDigestContextKeyType    struct{}
)

var (
	reconcilerNameContextKey     = reconcilerNameContextKeyType{}
	clientContextKey             = clientContextKeyType{}
	componentContextKey          = componentContextKeyType{}
	componentNameContextKey      = componentNameContextKeyType{}
	componentNamespaceContextKey = componentNamespaceContextKeyType{}
	componentDigestContextKey    = componentDigestContextKeyType{}
)

type Context interface {
	context.Context
	WithReconcilerName(reconcilerName string) Context
	WithClient(clnt cluster.Client) Context
	WithComponent(component Component) Context
	WithComponentName(componentName string) Context
	WithComponentNamespace(componentNamespace string) Context
	WithComponentDigest(componentDigest string) Context
}

func NewContext(ctx context.Context) Context {
	return &contextImpl{Context: ctx}
}

type contextImpl struct {
	context.Context
}

func (c *contextImpl) WithReconcilerName(reconcilerName string) Context {
	return &contextImpl{Context: context.WithValue(c, reconcilerNameContextKey, reconcilerName)}
}

func (c *contextImpl) WithClient(clnt cluster.Client) Context {
	return &contextImpl{Context: context.WithValue(c, clientContextKey, clnt)}
}

func (c *contextImpl) WithComponent(component Component) Context {
	return &contextImpl{Context: context.WithValue(c, componentContextKey, component)}
}

func (c *contextImpl) WithComponentName(componentName string) Context {
	return &contextImpl{Context: context.WithValue(c, componentNameContextKey, componentName)}
}

func (c *contextImpl) WithComponentNamespace(componentNamespace string) Context {
	return &contextImpl{Context: context.WithValue(c, componentNamespaceContextKey, componentNamespace)}
}

func (c *contextImpl) WithComponentDigest(componentDigest string) Context {
	return &contextImpl{Context: context.WithValue(c, componentDigestContextKey, componentDigest)}
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

// TODO: should this method be parameterized?
func ComponentFromContext(ctx context.Context) (Component, error) {
	if component, ok := ctx.Value(componentContextKey).(Component); ok {
		return component, nil
	}
	return nil, fmt.Errorf("component not found in context")
}

func ComponentNameFromContext(ctx context.Context) (string, error) {
	if componentName, ok := ctx.Value(componentNameContextKey).(string); ok {
		return componentName, nil
	}
	return "", fmt.Errorf("component name not found in context")
}

func ComponentNamespaceFromContext(ctx context.Context) (string, error) {
	if componentNamespace, ok := ctx.Value(componentNamespaceContextKey).(string); ok {
		return componentNamespace, nil
	}
	return "", fmt.Errorf("component namespace not found in context")
}

func ComponentDigestFromContext(ctx context.Context) (string, error) {
	if componentDigest, ok := ctx.Value(componentDigestContextKey).(string); ok {
		return componentDigest, nil
	}
	return "", fmt.Errorf("component digest not found in context")
}

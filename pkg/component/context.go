/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"

	"github.com/sap/component-operator-runtime/internal/contexts"
	"github.com/sap/component-operator-runtime/pkg/cluster"
)

func newContext(ctx context.Context) *reconcileContext {
	return &reconcileContext{Context: ctx}
}

type reconcileContext struct {
	context.Context
}

func (c *reconcileContext) WithReconcilerName(reconcilerName string) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, contexts.ReconcilerNameKey, reconcilerName)}
}

func (c *reconcileContext) WithClient(clnt cluster.Client) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, contexts.ClientKey, clnt)}
}

func (c *reconcileContext) WithComponent(component Component) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, contexts.ComponentKey, component)}
}

func (c *reconcileContext) WithComponentDigest(componentDigest string) *reconcileContext {
	return &reconcileContext{Context: context.WithValue(c, contexts.ComponentDigestKey, componentDigest)}
}

func ReconcilerNameFromContext(ctx context.Context) (string, error) {
	if reconcilerName, ok := ctx.Value(contexts.ReconcilerNameKey).(string); ok {
		return reconcilerName, nil
	}
	return "", fmt.Errorf("reconciler name not found in context")
}

func ClientFromContext(ctx context.Context) (cluster.Client, error) {
	if clnt, ok := ctx.Value(contexts.ClientKey).(cluster.Client); ok {
		return clnt, nil
	}
	return nil, fmt.Errorf("client not found in context")
}

func ComponentFromContext(ctx context.Context) (Component, error) {
	if component, ok := ctx.Value(contexts.ComponentKey).(Component); ok {
		return component, nil
	}
	return nil, fmt.Errorf("component not found in context")
}

func ComponentDigestFromContext(ctx context.Context) (string, error) {
	if componentDigest, ok := ctx.Value(contexts.ComponentDigestKey).(string); ok {
		return componentDigest, nil
	}
	return "", fmt.Errorf("component digest not found in context")
}

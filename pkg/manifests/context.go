/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"context"
	"fmt"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

type reconcilerNameContextKey struct{}
type clientContextKey struct{}

func NewContextWithReconcilerName(ctx context.Context, reconcilerName string) context.Context {
	return context.WithValue(ctx, reconcilerNameContextKey{}, reconcilerName)
}

func NewContextWithClient(ctx context.Context, client cluster.Client) context.Context {
	return context.WithValue(ctx, clientContextKey{}, client)
}

func ReconcilerNameFromContext(ctx context.Context) (string, error) {
	if reconcilerName, ok := ctx.Value(reconcilerNameContextKey{}).(string); ok {
		return reconcilerName, nil
	}
	return "", fmt.Errorf("reconciler name not found in context")
}

func ClientFromContext(ctx context.Context) (cluster.Client, error) {
	if client, ok := ctx.Value(clientContextKey{}).(cluster.Client); ok {
		return client, nil
	}
	return nil, fmt.Errorf("client not found in context")
}

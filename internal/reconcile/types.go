/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconcile

import (
	"context"

	"github.com/sap/component-operator-runtime/internal/cluster"
)

// Context type the framework passes to generators' Generate() method.
type Context interface {
	context.Context
	// Return new context with given reconciler name added as value.
	WithReconcilerName(reconcilerName string) Context
	// Return new context with given client added as value.
	WithClient(client cluster.Client) Context
	// Return new context with given component as value.
	WithComponentDigest(componentDigest string) Context
}

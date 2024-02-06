/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package utils

import "github.com/sap/component-operator-runtime/internal/reconcile"

var (
	// Retrieve reconciler name from given context.
	ReconcilerNameFromContext = reconcile.ReconcilerNameFromContext
	// Retrieve client from given context.
	ClientFromContext = reconcile.ClientFromContext
	// Retrieve component digest from given context.
	ComponentDigestFromContext = reconcile.ComponentDigestFromContext
)

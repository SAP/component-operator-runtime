/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package contexts

type reconcilerNameKey struct{}
type controllerNameKey struct{}
type clientKey struct{}
type componentKey struct{}
type componentDigestKey struct{}

var (
	ReconcilerNameKey  = reconcilerNameKey{}
	ControllerNameKey  = controllerNameKey{}
	ClientKey          = clientKey{}
	ComponentKey       = componentKey{}
	ComponentDigestKey = componentDigestKey{}
)

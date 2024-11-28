/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

import "k8s.io/apimachinery/pkg/runtime"

// SchemeBuilder interface.
type SchemeBuilder interface {
	AddToScheme(scheme *runtime.Scheme) error
}

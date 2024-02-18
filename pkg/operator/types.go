/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"flag"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Operator interface {
	GetName() string
	InitScheme(scheme *runtime.Scheme)
	InitFlags(flagset *flag.FlagSet)
	ValidateFlags() error
	GetUncacheableTypes() []client.Object
	Setup(mgr ctrl.Manager) error
}

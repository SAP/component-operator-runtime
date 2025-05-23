/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	prefix = "component_operator_runtime"
)

var (
	Reconciles = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_reconcile_total",
			Help: "Total number of reconciliations per controller",
		},
		[]string{"controller"},
	)
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_reconcile_errors_total",
			Help: "Total number of reconciliation errors per controller and type",
		},
		[]string{"controller", "type"},
	)
	Requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_requests_total",
			Help: "Kubernetes API server requests per controller and method",
		},
		[]string{"controller", "method"},
	)
	Operations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_operations_total",
			Help: "Dependent operations per controller and action",
		},
		[]string{"controller", "action"},
	)
	CreatedClients = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_created_clients_total",
			Help: "Kubernetes API clients created since the controller was started",
		},
		[]string{"controller"},
	)
	ActiveClients = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_active_clients_total",
			Help: "Currently active Kubernetes API clients",
		},
		[]string{"controller"},
	)
	ComponentState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_component_state",
			Help: "Current state of a component",
		},
		[]string{"controller", "group", "kind", "namespace", "name", "state"},
	)
	Dependents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_dependents_total",
			Help: "Number of dependent objects",
		},
		[]string{"controller", "group", "kind", "namespace", "name"},
	)
	UnreadyDependents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_unready_dependents_total",
			Help: "Number of unready dependent objects",
		},
		[]string{"controller", "group", "kind", "namespace", "name"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		Reconciles,
		ReconcileErrors,
		Requests,
		Operations,
		CreatedClients,
		ActiveClients,
		ComponentState,
		Dependents,
		UnreadyDependents,
	)
}

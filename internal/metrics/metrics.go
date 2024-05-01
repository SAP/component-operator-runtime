/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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
	/*
		ProjectTTLSecondsInitial = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "project_ttl_seconds_initial",
				Help: "Initial TTL per project. Zero means that the project has no TTL, i.e. does not expire",
			},
			[]string{"project"},
		)
		ProjectTTLSecondsRemaining = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "project_ttl_seconds_remaining",
				Help: "Remaining TTL per project. Zero, and therefore not meaningful, if the project has no TTL set",
			},
			[]string{"project"},
		)
		ProjectExpired = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "project_expired",
				Help: "Whether project is expired. Because it has no TTL or TTL is expired. One means true, zero means false",
			},
			[]string{"project"},
		)
	*/
)

func init() {
	metrics.Registry.MustRegister(
		Reconciles,
		ReconcileErrors,
		Requests,
		Operations,
		//ProjectTTLSecondsInitial,
		//ProjectTTLSecondsRemaining,
		//ProjectExpired,
	)
}

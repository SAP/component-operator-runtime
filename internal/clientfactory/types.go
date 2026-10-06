/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"time"

	"k8s.io/client-go/tools/events"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

type Client struct {
	cluster.Client
	eventBroadcaster events.EventBroadcaster
	validUntil       time.Time
}

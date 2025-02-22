/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

type Client struct {
	cluster.Client
	eventBroadcaster record.EventBroadcaster
	validUntil       time.Time
}

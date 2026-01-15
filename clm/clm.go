/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sap/component-operator-runtime/clm/cmd"
)

func main() {
	// ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	log.SetLogger(logr.Discard())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

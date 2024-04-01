/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"

	"github.com/pkg/errors"

	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
)

type reconcileTarget[T Component] struct {
	reconciler        *reconciler.Reconciler
	reconcilerName    string
	reconcilerId      string
	client            cluster.Client
	resourceGenerator manifests.Generator
}

func newReconcileTarget[T Component](reconcilerName string, reconcilerId string, clnt cluster.Client, resourceGenerator manifests.Generator, options reconciler.ReconcilerOptions) *reconcileTarget[T] {
	return &reconcileTarget[T]{
		reconcilerName:    reconcilerName,
		reconcilerId:      reconcilerId,
		reconciler:        reconciler.NewReconciler(reconcilerName, clnt, options),
		client:            clnt,
		resourceGenerator: resourceGenerator,
	}
}

func (t *reconcileTarget[T]) Apply(ctx context.Context, component T) (bool, error) {
	//log := log.FromContext(ctx)
	namespace := ""
	name := ""
	if placementConfiguration, ok := assertPlacementConfiguration(component); ok {
		namespace = placementConfiguration.GetDeploymentNamespace()
		name = placementConfiguration.GetDeploymentName()
	}
	if namespace == "" {
		namespace = component.GetNamespace()
	}
	if name == "" {
		name = component.GetName()
	}
	ownerId := t.reconcilerId + "/" + component.GetNamespace() + "/" + component.GetName()
	status := component.GetStatus()
	componentDigest := calculateComponentDigest(component)

	generateCtx := newContext(ctx).
		WithReconcilerName(t.reconcilerName).
		WithClient(t.client).
		WithComponent(component).
		WithComponentDigest(componentDigest)
	objects, err := t.resourceGenerator.Generate(generateCtx, namespace, name, component.GetSpec())
	if err != nil {
		return false, errors.Wrap(err, "error rendering manifests")
	}

	return t.reconciler.Apply(ctx, &status.Inventory, objects, namespace, ownerId, component.GetGeneration())
}

func (t *reconcileTarget[T]) Delete(ctx context.Context, component T) (bool, error) {
	// log := log.FromContext(ctx)
	status := component.GetStatus()

	return t.reconciler.Delete(ctx, &status.Inventory)
}

func (t *reconcileTarget[T]) IsDeletionAllowed(ctx context.Context, component T) (bool, string, error) {
	// log := log.FromContext(ctx)
	status := component.GetStatus()

	return t.reconciler.IsDeletionAllowed(ctx, &status.Inventory)
}

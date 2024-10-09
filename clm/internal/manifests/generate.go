/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/manifests/helm"
	"github.com/sap/component-operator-runtime/pkg/manifests/kustomize"
	"github.com/sap/component-operator-runtime/pkg/types"
)

func Generate(manifestSources []string, valuesSources []string, reconcilerName string, clnt cluster.Client, namespace string, name string) ([]client.Object, error) {
	var allObjects []client.Object
	var allValues = make(map[string]any)

	for _, source := range valuesSources {
		// TODO: suuport URLs
		path := source

		rawValues, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var values map[string]any
		if err := kyaml.Unmarshal(rawValues, &values); err != nil {
			return nil, err
		}
		manifests.MergeMapInto(allValues, values)
	}

	for _, source := range manifestSources {
		// TODO: support helm, oci URLs
		path := source

		if info, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("no such file or directory: %s", path)
			} else {
				return nil, err
			}
		} else if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", path)
		}
		path, err := filepath.Abs(source)
		if err != nil {
			return nil, err
		}
		fsys := os.DirFS(path)

		var generator manifests.Generator
		if _, err = fs.Stat(fsys, "Chart.yaml"); err == nil {
			generator, err = helm.NewHelmGenerator(fsys, "", clnt)
			if err != nil {
				return nil, err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			generator, err = kustomize.NewKustomizeGenerator(fsys, "", clnt, kustomize.KustomizeGeneratorOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}

		// TODO: what about component and component digest
		generateCtx := component.NewContext(context.TODO()).
			WithReconcilerName(reconcilerName).
			WithClient(clnt).
			WithComponent(nil).
			WithComponentDigest("")
		objects, err := generator.Generate(generateCtx, namespace, name, types.UnstructurableMap(allValues))
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}
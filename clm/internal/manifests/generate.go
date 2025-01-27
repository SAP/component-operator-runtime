/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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

	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/manifests/helm"
	"github.com/sap/component-operator-runtime/pkg/manifests/kustomize"
	"github.com/sap/component-operator-runtime/pkg/types"
)

func Generate(manifestSources []string, valuesSources []string, reconcilerName string, clnt cluster.Client, release *release.Release) ([]client.Object, error) {
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
			tmpdir, err := os.MkdirTemp("", "clm-")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(tmpdir)
			if _, err := copyFile(path, fmt.Sprintf("%s/%s", tmpdir, "resources.yaml")); err != nil {
				return nil, err
			}
			path = tmpdir
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		fsys := os.DirFS(path)

		var generator manifests.Generator
		if _, err = fs.Stat(fsys, "Chart.yaml"); err == nil {
			generator, err = helm.NewHelmGenerator(fsys, "", nil)
			if err != nil {
				return nil, err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			generator, err = kustomize.NewKustomizeGenerator(fsys, "", nil, kustomize.KustomizeGeneratorOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}

		releaseComponent := componentFromRelease(release, allValues)
		// TODO: what about component digest
		generateCtx := component.NewContext(context.TODO()).
			WithReconcilerName(reconcilerName).
			WithLocalClient(clnt).
			WithClient(clnt).
			WithComponent(releaseComponent).
			WithComponentName(releaseComponent.GetName()).
			WithComponentNamespace(releaseComponent.GetNamespace()).
			WithComponentDigest("")
		objects, err := generator.Generate(generateCtx, release.GetNamespace(), release.GetName(), types.UnstructurableMap(allValues))
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}

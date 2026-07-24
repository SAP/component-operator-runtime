/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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

		var fsys fs.FS
		var path string

		source, err := filepath.Abs(source)
		if err != nil {
			return nil, err
		}

		if info, err := os.Stat(source); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("no such file or directory: %s", source)
			} else {
				return nil, err
			}
		} else if info.IsDir() {
			fsys = os.DirFS("/")
			path = source[1:]
		} else {
			tmpdir, err := os.MkdirTemp("", "clm-")
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(tmpdir)
			if _, err := copyFile(source, fmt.Sprintf("%s/%s", tmpdir, "resources.yaml")); err != nil {
				return nil, err
			}
			fsys = os.DirFS(tmpdir)
			path = "."
		}

		var generator manifests.Generator
		if _, err = fs.Stat(fsys, filepath.Clean(path+"/Chart.yaml")); err == nil {
			generator, err = helm.NewHelmGenerator(fsys, path, nil)
			if err != nil {
				return nil, err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			generator, err = kustomize.NewKustomizeGenerator(fsys, path, nil, kustomize.KustomizeGeneratorOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}

		releaseComponent := componentFromRelease(release, allValues)

		generateCtx := component.NewContext(context.TODO()).
			WithReconcilerName(reconcilerName).
			WithLocalClient(clnt).
			WithClient(clnt).
			WithComponent(releaseComponent).
			WithComponentName(releaseComponent.Name).
			WithComponentNamespace(releaseComponent.Namespace).
			WithComponentDigest(releaseComponent.Status.ProcessingDigest).
			WithComponentRevision(releaseComponent.Status.Revision)
		objects, err := generator.Generate(generateCtx, releaseComponent.Namespace, releaseComponent.Name, types.UnstructurableMap(allValues))
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}

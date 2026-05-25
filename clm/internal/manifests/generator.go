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

	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/manifests/helm"
	"github.com/sap/component-operator-runtime/pkg/manifests/kustomize"
	"github.com/sap/component-operator-runtime/pkg/types"
)

type Generator struct {
	generators []manifests.Generator
}

func NewGenerator(manifestSources []string) (manifests.Generator, error) {
	var generators []manifests.Generator

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

		generators = append(generators, generator)
	}

	return &Generator{
		generators: generators,
	}, nil
}

func (g *Generator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var allObjects []client.Object

	for _, generator := range g.generators {
		objects, err := generator.Generate(ctx, namespace, name, parameters)
		if err != nil {
			return nil, err
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}

/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// KustomizeGenerator is a Generator implementation that basically renders a given Kustomization.
type KustomizeGenerator struct {
	kustomizer *krusty.Kustomizer
	templates  []*template.Template
}

var _ Generator = &KustomizeGenerator{}

// Create a new KustomizeGenerator.
func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client) (*KustomizeGenerator, error) {
	g := KustomizeGenerator{}

	if fsys == nil {
		fsys = os.DirFS("/")
		absoluteKustomizationPath, err := filepath.Abs(kustomizationPath)
		if err != nil {
			return nil, err
		}
		kustomizationPath = absoluteKustomizationPath[1:]
	}

	options := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	g.kustomizer = krusty.MakeKustomizer(options)

	var t *template.Template
	files, err := find(fsys, kustomizationPath, "*"+templateSuffix, fileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		// Note: we use relative paths as templates names to make it easier to copy the kustomization
		// content into the ephemeral in-memory filesystem used by krusty in Generate()
		name, err := filepath.Rel(kustomizationPath, file)
		if err != nil {
			// TODO: is it ok to panic here in case of error ?
			panic("this cannot happen")
		}
		if t == nil {
			t = template.New(name)
		} else {
			t = t.New(name)
		}
		t.Option("missingkey=zero").
			Funcs(sprig.TxtFuncMap()).
			Funcs(templatex.FuncMap()).
			Funcs(templatex.FuncMapForTemplate(t)).
			Funcs(templatex.FuncMapForClient(client))
		if _, err := t.Parse(string(raw)); err != nil {
			return nil, err
		}
		g.templates = append(g.templates, t)
	}

	return &g, nil
}

// Create a new KustomizeGenerator as TransformableGenerator
func NewTransformableKustomizeGenerator(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client) (TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return NewGenerator(g), nil
}

// Create a new KustomizeGenerator with a ParameterTransformer attached (further transformers can be attached to the reeturned generator object).
func NewKustomizeGeneratorWithParameterTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ParameterTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new KustomizeGenerator with an ObjectTransformer attached (further transformers can be attached to the reeturned generator object).
func NewKustomizeGeneratorWithObjectTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ObjectTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *KustomizeGenerator) Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	data := parameters.ToUnstructured()
	fsys := kustfsys.MakeFsInMemory()

	for _, t := range g.templates {
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(t.Name(), buf.Bytes()); err != nil {
			return nil, err
		}
	}

	resmap, err := g.kustomizer.Run(fsys, "/")
	if err != nil {
		return nil, err
	}

	raw, err := resmap.AsYaml()
	if err != nil {
		return nil, err
	}

	decoder := utilyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(raw))
	for {
		object := &unstructured.Unstructured{}
		if err := decoder.Decode(&object.Object); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if object.Object == nil {
			continue
		}
		objects = append(objects, object)
	}

	return objects, nil
}

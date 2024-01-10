/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// KustomizeGenerator is a Generator implementation that basically renders a given Kustomization.
type KustomizeGenerator struct {
	kustomizer *krusty.Kustomizer
	files      map[string][]byte
	templates  map[string]*template.Template
}

var _ Generator = &KustomizeGenerator{}

// Create a new KustomizeGenerator.
// Deprecation warning: the parameter client is ignored (can be passed as nil) and will be removed in a future release;
// the according value will be retrieved from the context passed to Generate().
// If fsys is nil, the local operating system filesystem will be used, and kustomizationPath can be an absolute or relative path (in the latter case it will be considered
// relative to the current working directory). If fsys is non-nil, then kustomizationPath should be a relative path; if an absolute path is supplied, it will be turned
// An empty kustomizationPath will be treated like ".".
func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client) (*KustomizeGenerator, error) {
	g := KustomizeGenerator{
		files:     make(map[string][]byte),
		templates: make(map[string]*template.Template),
	}

	if fsys == nil {
		fsys = os.DirFS("/")
		absoluteKustomizationPath, err := filepath.Abs(kustomizationPath)
		if err != nil {
			return nil, err
		}
		kustomizationPath = absoluteKustomizationPath[1:]
	} else if filepath.IsAbs(kustomizationPath) {
		kustomizationPath = kustomizationPath[1:]
	}
	kustomizationPath = filepath.Clean(kustomizationPath)

	options := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	g.kustomizer = krusty.MakeKustomizer(options)

	var t *template.Template
	files, err := find(fsys, kustomizationPath, "*", fileTypeRegular, 0)
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
		if strings.HasSuffix(name, templateSuffix) {
			if t == nil {
				t = template.New(name)
				t.Option("missingkey=zero").
					Funcs(sprig.TxtFuncMap()).
					Funcs(templatex.FuncMap()).
					Funcs(templatex.FuncMapForTemplate(t)).
					Funcs(templatex.FuncMapForClient(nil)).
					Funcs(funcMapForGenerateContext(nil, "", ""))
			} else {
				t = t.New(name)
			}
			if _, err := t.Parse(string(raw)); err != nil {
				return nil, err
			}
			g.templates[strings.TrimSuffix(name, templateSuffix)] = t
		} else {
			g.files[name] = raw
		}
	}

	// TODO: check that g.files and g.templates are disjoint

	return &g, nil
}

// Create a new KustomizeGenerator as TransformableGenerator.
// Deprecation warning: the parameter client is ignored (can be passed as nil) and will be removed in a future release.
func NewTransformableKustomizeGenerator(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client) (TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return NewGenerator(g), nil
}

// Create a new KustomizeGenerator with a ParameterTransformer attached (further transformers can be attached to the returned generator object).
// Deprecation warning: the parameter client is ignored (can be passed as nil) and will be removed in a future release.
func NewKustomizeGeneratorWithParameterTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ParameterTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new KustomizeGenerator with an ObjectTransformer attached (further transformers can be attached to the returned generator object).
// Deprecation warning: the parameter client is ignored (can be passed as nil) and will be removed in a future release.
func NewKustomizeGeneratorWithObjectTransformer(fsys fs.FS, kustomizationPath string, templateSuffix string, client client.Client, transformer ObjectTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, templateSuffix, client)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *KustomizeGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	client, err := ClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	data := parameters.ToUnstructured()
	fsys := kustfsys.MakeFsInMemory()

	for n, f := range g.files {
		if err := fsys.WriteFile(n, f); err != nil {
			return nil, err
		}
	}

	var t0 *template.Template
	for n, t := range g.templates {
		if t0 == nil {
			t0, err = t.Clone()
			if err != nil {
				return nil, err
			}
			t0.Option("missingkey=zero").
				Funcs(templatex.FuncMapForClient(client)).
				Funcs(funcMapForGenerateContext(client, namespace, name))
		}
		var buf bytes.Buffer
		if err := t0.ExecuteTemplate(&buf, t.Name(), data); err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(n, buf.Bytes()); err != nil {
			return nil, err
		}
	}

	haveKustomization := false
	for _, kustomizationName := range konfig.RecognizedKustomizationFileNames() {
		if fsys.Exists(kustomizationName) {
			haveKustomization = true
			break
		}
	}
	if !haveKustomization {
		kustomization, err := generateKustomization(fsys)
		if err != nil {
			return nil, err
		}
		if err := fsys.WriteFile(konfig.DefaultKustomizationFileName(), kustomization); err != nil {
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

func funcMapForGenerateContext(client cluster.Client, namespace string, name string) template.FuncMap {
	// TODO: add accessors for Kubernetes version etc.
	return template.FuncMap{
		"namespace": func() string { return namespace },
		"name":      func() string { return name },
	}
}

func generateKustomization(fsys kustfsys.FileSystem) ([]byte, error) {
	var resources []string

	f := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			resources = append(resources, path)
		}
		return nil
	}

	if err := fsys.Walk(".", f); err != nil {
		return nil, err
	}

	kustomization := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
		Resources: resources,
	}

	rawKustomization, err := kyaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}

	return rawKustomization, nil
}

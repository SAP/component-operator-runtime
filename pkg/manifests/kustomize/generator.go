/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/fileutils"
	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: carve out logic into an internal Kustomization type (similar to the helm Chart case)

const (
	componentConfigFilename = ".component-config.yaml"
)

// KustomizeGeneratorOptions allows to tweak the behavior of the kustomize generator.
type KustomizeGeneratorOptions struct {
	// If defined, only files with that suffix will be subject to templating.
	TemplateSuffix *string
	// If defined, the given left delimiter will be used to parse go templates; otherwise, defaults to '{{'
	LeftTemplateDelimiter *string
	// If defined, the given right delimiter will be used to parse go templates; otherwise, defaults to '}}'
	RightTemplateDelimiter *string
}

// KustomizeGenerator is a Generator implementation that basically renders a given Kustomization.
// Note: KustomizeGenerator's Generate() method expects local client, client and component to be set in the passed context;
// see: Context.WithLocalClient(), Context.WithClient() and Context.WithComponent() in package pkg/component.
type KustomizeGenerator struct {
	kustomizer *krusty.Kustomizer
	files      map[string][]byte
	templates  map[string]*template.Template
}

var _ manifests.Generator = &KustomizeGenerator{}

// TODO: add a way to pass custom template functions

// Create a new KustomizeGenerator.
// The client parameter is deprecated (ignored) and will be removed in a future release.
// If fsys is nil, the local operating system filesystem will be used, and kustomizationPath can be an absolute or relative path (in the latter case it will be considered
// relative to the current working directory). If fsys is non-nil, then kustomizationPath should be a relative path; if an absolute path is supplied, it will be turned
// An empty kustomizationPath will be treated like ".".
func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, _ client.Client, options KustomizeGeneratorOptions) (*KustomizeGenerator, error) {
	if options.TemplateSuffix == nil {
		options.TemplateSuffix = ref("")
	}
	if options.LeftTemplateDelimiter == nil {
		options.LeftTemplateDelimiter = ref("")
	}
	if options.RightTemplateDelimiter == nil {
		options.RightTemplateDelimiter = ref("")
	}

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

	kustomizerOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	g.kustomizer = krusty.MakeKustomizer(kustomizerOptions)

	if err := readOptions(fsys, filepath.Clean(kustomizationPath+"/"+componentConfigFilename), &options); err != nil {
		return nil, err
	}

	var t *template.Template
	// TODO: we should consider the whole of fsys, not only the subtree rooted at kustomizationPath;
	// this would allow people to reference resources or patches or components located in parent directories
	// (which is probably a common usecase); however it has to be clarified how to handle template scopes;
	// for example it might be desired that subtrees with a kustomization.yaml file are processed in an own
	// template context
	files, err := fileutils.Find(fsys, kustomizationPath, "*", fileutils.FileTypeRegular, 0)
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
		if filepath.Base(name) != componentConfigFilename && strings.HasSuffix(name, *options.TemplateSuffix) {
			if t == nil {
				t = template.New(name)
				t.Delims(*options.LeftTemplateDelimiter, *options.RightTemplateDelimiter)
				t.Option("missingkey=zero").
					Funcs(sprig.TxtFuncMap()).
					Funcs(templatex.FuncMap()).
					Funcs(templatex.FuncMapForTemplate(nil)).
					Funcs(templatex.FuncMapForLocalClient(nil)).
					Funcs(templatex.FuncMapForClient(nil)).
					Funcs(funcMapForGenerateContext(nil, nil, "", ""))
			} else {
				t = t.New(name)
			}
			if _, err := t.Parse(string(raw)); err != nil {
				return nil, err
			}
			g.templates[strings.TrimSuffix(name, *options.TemplateSuffix)] = t
		} else {
			g.files[name] = raw
		}
	}

	// TODO: check that g.files and g.templates are disjoint

	return &g, nil
}

// Create a new KustomizeGenerator as TransformableGenerator.
func NewTransformableKustomizeGenerator(fsys fs.FS, kustomizationPath string, clnt client.Client, options KustomizeGeneratorOptions) (manifests.TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, clnt, options)
	if err != nil {
		return nil, err
	}
	return manifests.NewGenerator(g), nil
}

// Create a new KustomizeGenerator with a ParameterTransformer attached (further transformers can be attached to the returned generator object).
func NewKustomizeGeneratorWithParameterTransformer(fsys fs.FS, kustomizationPath string, clnt client.Client, options KustomizeGeneratorOptions, transformer manifests.ParameterTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, clnt, options)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new KustomizeGenerator with an ObjectTransformer attached (further transformers can be attached to the returned generator object).
func NewKustomizeGeneratorWithObjectTransformer(fsys fs.FS, kustomizationPath string, clnt client.Client, options KustomizeGeneratorOptions, transformer manifests.ObjectTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, clnt, options)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *KustomizeGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	localClient, err := component.LocalClientFromContext(ctx)
	if err != nil {
		return nil, err
	}
	clnt, err := component.ClientFromContext(ctx)
	if err != nil {
		return nil, err
	}
	component, err := component.ComponentFromContext(ctx)
	if err != nil {
		return nil, err
	}
	serverInfo, err := clnt.DiscoveryClient().ServerVersion()
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
				Funcs(templatex.FuncMapForTemplate(t0)).
				Funcs(templatex.FuncMapForLocalClient(localClient)).
				Funcs(templatex.FuncMapForClient(clnt)).
				Funcs(funcMapForGenerateContext(serverInfo, component, namespace, name))
		}
		var buf bytes.Buffer
		// TODO: templates (accidentally or intentionally) could modify data, or even some of the objects supplied through builtin functions;
		// such as serverInfo or component; this should be hardened, e.g. by deep-copying things upfront, or serializing them; see the comment in
		// funcMapForGenerateContext()
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

func funcMapForGenerateContext(serverInfo *version.Info, component component.Component, namespace string, name string) template.FuncMap {
	return template.FuncMap{
		// TODO: maybe it would it be better to convert component to unstructured;
		// then calling methods would no longer be possible, and attributes would be in lowercase
		"component":         makeFuncData(component),
		"namespace":         func() string { return namespace },
		"name":              func() string { return name },
		"kubernetesVersion": func() *version.Info { return serverInfo },
	}
}

func makeFuncData(data any) any {
	if data == nil {
		return func() any { return nil }
	}
	ival := reflect.ValueOf(data)
	ityp := ival.Type()
	ftyp := reflect.FuncOf(nil, []reflect.Type{ityp}, false)
	fval := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value { return []reflect.Value{ival} })
	return fval.Interface()
}

func generateKustomization(fsys kustfsys.FileSystem) ([]byte, error) {
	var resources []string

	f := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasPrefix(filepath.Base(path), ".") && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
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

func readOptions(fsys fs.FS, path string, options *KustomizeGeneratorOptions) error {
	rawOptions, err := fs.ReadFile(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	if err := kyaml.Unmarshal(rawOptions, options); err != nil {
		return err
	}

	return nil
}

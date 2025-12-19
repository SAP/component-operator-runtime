/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize

import (
	"bytes"
	"context"
	"io"
	"io/fs"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/sap/component-operator-runtime/internal/kustomize"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// KustomizeGeneratorOptions allows to tweak the behavior of the kustomize generator.
type KustomizeGeneratorOptions struct {
	TemplateSuffix *string
	// If defined, the given left delimiter will be used to parse go templates; otherwise, defaults to '{{'
	LeftTemplateDelimiter *string
	// If defined, the given right delimiter will be used to parse go templates; otherwise, defaults to '}}'
	RightTemplateDelimiter *string
	// If defined, used to decrypt files
	Decryptor manifests.Decryptor
}

// KustomizeGenerator is a Generator implementation that basically renders a given Kustomization.
// Note: KustomizeGenerator's Generate() method expects local client, client and component to be set in the passed context;
// see: Context.WithLocalClient(), Context.WithClient() and Context.WithComponent() in package pkg/component.
type KustomizeGenerator struct {
	kustomization *kustomize.Kustomization
	kustomizer    *krusty.Kustomizer
}

var _ manifests.Generator = &KustomizeGenerator{}

// TODO: add a way to pass custom template functions

// Create a new KustomizeGenerator.
// The client parameter is deprecated (ignored) and will be removed in a future release.
// If fsys is nil, the local operating system filesystem will be used, and kustomizationPath can be an absolute or relative path (in the latter case it will be considered
// relative to the current working directory). If fsys is non-nil, then kustomizationPath should be a relative path; if an absolute path is supplied, it will be turned
// into a relative path by stripping the leading slash. If fsys is specified as a real filesystem, it is recommended to use os.Root.FS() instead of os.DirFS(), in order
// to fence symbolic links. An empty kustomizationPath will be treated like ".".
func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, _ client.Client, options KustomizeGeneratorOptions) (*KustomizeGenerator, error) {
	kustomization, err := kustomize.ParseKustomization(fsys, kustomizationPath, kustomize.KustomizationOptions{
		TemplateSuffix:         options.TemplateSuffix,
		LeftTemplateDelimiter:  options.LeftTemplateDelimiter,
		RightTemplateDelimiter: options.RightTemplateDelimiter,
		Decryptor:              options.Decryptor,
	})
	if err != nil {
		return nil, err
	}

	kustomizerOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}
	kustomizer := krusty.MakeKustomizer(kustomizerOptions)

	return &KustomizeGenerator{
		kustomization: kustomization,
		kustomizer:    kustomizer,
	}, nil
}

// Create a new KustomizeGenerator as TransformableGenerator.
func NewTransformableKustomizeGenerator(fsys fs.FS, kustomizationPath string, _ client.Client, options KustomizeGeneratorOptions) (manifests.TransformableGenerator, error) {
	g, err := NewKustomizeGenerator(fsys, kustomizationPath, nil, options)
	if err != nil {
		return nil, err
	}
	return manifests.NewGenerator(g), nil
}

// Create a new KustomizeGenerator with a ParameterTransformer attached (further transformers can be attached to the returned generator object).
func NewKustomizeGeneratorWithParameterTransformer(fsys fs.FS, kustomizationPath string, _ client.Client, options KustomizeGeneratorOptions, transformer manifests.ParameterTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, nil, options)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new KustomizeGenerator with an ObjectTransformer attached (further transformers can be attached to the returned generator object).
func NewKustomizeGeneratorWithObjectTransformer(fsys fs.FS, kustomizationPath string, _ client.Client, options KustomizeGeneratorOptions, transformer manifests.ObjectTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableKustomizeGenerator(fsys, kustomizationPath, nil, options)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *KustomizeGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object
	fsys := kustfsys.MakeFsInMemory()

	localClient, err := component.LocalClientFromContext(ctx)
	if err != nil {
		return nil, err
	}
	clnt, err := component.ClientFromContext(ctx)
	if err != nil {
		return nil, err
	}
	componentDigest, err := component.ComponentDigestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	component, err := component.ComponentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := g.kustomization.Render(kustomize.RenderContext{
		LocalClient:     localClient,
		Client:          clnt,
		DiscoveryClient: clnt.DiscoveryClient(),
		Component:       component,
		ComponentDigest: componentDigest,
		Namespace:       namespace,
		Name:            name,
		Parameters:      parameters.ToUnstructured(),
	}, fsys); err != nil {
		return nil, err
	}

	resmap, err := g.kustomizer.Run(fsys, g.kustomization.Path())
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

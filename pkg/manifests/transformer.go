/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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
	"github.com/drone/envsubst"
	"github.com/sap/go-generics/slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	kusttypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TemplateParameterTransformer allows to transform parameters through a given go template.
// The template can use all functions from the sprig library, plus toYaml, fromYaml, toJson, fromJson, required.
type TemplateParameterTransformer struct {
	template *template.Template
}

var _ ParameterTransformer = &TemplateParameterTransformer{}

// Create a new TemplateParameterTransformer (reading the template from the given fsys and path).
// If fsys is nil, the local OS filesystem will be used.
func NewTemplateParameterTransformer(fsys fs.FS, path string) (*TemplateParameterTransformer, error) {
	t := &TemplateParameterTransformer{}

	if fsys == nil {
		fsys = os.DirFS("/")
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		path = absolutePath[1:]
	}

	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	t.template = template.New("gotpl").Option("missingkey=zero").Funcs(sprig.TxtFuncMap()).Funcs(templatex.FuncMap())
	if _, err := t.template.Parse(string(raw)); err != nil {
		return nil, err
	}

	return t, nil
}

// Transform parameters.
func (t *TemplateParameterTransformer) TransformParameters(namespace string, name string, parameters types.Unstructurable) (types.Unstructurable, error) {
	data := parameters.ToUnstructured()
	data["Namespace"] = namespace
	data["Name"] = name
	var buf bytes.Buffer
	if err := t.template.Execute(&buf, data); err != nil {
		return nil, err
	}
	var transformedParameters types.UnstructurableMap
	if err := kyaml.Unmarshal(templatex.AdjustTemplateOutput(buf.Bytes()), &transformedParameters); err != nil {
		return nil, err
	}
	return transformedParameters, nil
}

type SubstitutionObjectTransformer struct {
	substitutions map[string]string
	selector      types.Selector[client.Object]
}

var _ ObjectTransformer = &SubstitutionObjectTransformer{}

func NewSubstitutionObjectTransformer(substitutions map[string]string, selector types.Selector[client.Object]) (*SubstitutionObjectTransformer, error) {
	return &SubstitutionObjectTransformer{
		substitutions: substitutions,
		selector:      selector,
	}, nil
}

func (t *SubstitutionObjectTransformer) TransformObjects(namespace string, name string, objects []client.Object) ([]client.Object, error) {
	if len(t.substitutions) == 0 {
		return objects, nil
	}

	var transformedObjects []client.Object

	for _, object := range objects {
		if !t.selector.Matches(object) {
			transformedObjects = append(transformedObjects, object)
			continue
		}
		rawObject, err := kyaml.Marshal(object)
		if err != nil {
			return nil, err
		}
		stringObject := string(rawObject)
		stringObject, err = envsubst.Eval(stringObject, func(s string) string {
			return t.substitutions[s]
		})
		if err != nil {
			return nil, err
		}
		rawObject = []byte(stringObject)
		if err := kyaml.Unmarshal(rawObject, object); err != nil {
			return nil, err
		}
		transformedObjects = append(transformedObjects, object)
	}

	return transformedObjects, nil
}

type KustomizeObjectTransformer struct {
	patches    []KustomizePatch
	images     []KustomizeImage
	kustomizer *krusty.Kustomizer
}

var _ ObjectTransformer = &KustomizeObjectTransformer{}

func NewKustomizeObjectTransformer(patches []KustomizePatch, images []KustomizeImage) (*KustomizeObjectTransformer, error) {
	kustomizerOptions := &krusty.Options{
		LoadRestrictions: kusttypes.LoadRestrictionsNone,
		PluginConfig:     kusttypes.DisabledPluginConfig(),
	}
	kustomizer := krusty.MakeKustomizer(kustomizerOptions)

	return &KustomizeObjectTransformer{
		patches:    patches,
		images:     images,
		kustomizer: kustomizer,
	}, nil
}

func (t *KustomizeObjectTransformer) TransformObjects(namespace string, name string, objects []client.Object) ([]client.Object, error) {
	const resourcePath = "objects.yaml"

	fsys := kustfsys.MakeFsInMemory()

	var buf bytes.Buffer
	for _, object := range objects {
		rawObject, err := kyaml.Marshal(object)
		if err != nil {
			return nil, err
		}
		// note: no need to handle write errors below, they are always nil (see documentation)
		buf.WriteString("---\n")
		buf.Write(rawObject)
	}
	if err := fsys.WriteFile(resourcePath, buf.Bytes()); err != nil {
		return nil, err
	}

	kustomization := kusttypes.Kustomization{
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		Resources: []string{resourcePath},
		Images: slices.Collect(t.images, func(i KustomizeImage) kusttypes.Image {
			return kusttypes.Image{
				Name:    i.Name,
				NewName: i.NewName,
				NewTag:  i.NewTag,
				Digest:  i.Digest,
			}
		}),
		Patches: slices.Collect(t.patches, func(p KustomizePatch) kusttypes.Patch {
			return kusttypes.Patch{
				Patch: p.Patch,
				Target: &kusttypes.Selector{
					ResId: resid.ResId{
						Gvk: resid.Gvk{
							Group:   p.Target.Group,
							Version: p.Target.Version,
							Kind:    p.Target.Kind,
						},
						Name:      p.Target.Name,
						Namespace: p.Target.Namespace,
					},
					LabelSelector:      p.Target.LabelSelector,
					AnnotationSelector: p.Target.AnnotationSelector,
				},
			}
		}),
	}
	rawKustomization, err := kyaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}
	if err := fsys.WriteFile(konfig.DefaultKustomizationFileName(), rawKustomization); err != nil {
		return nil, err
	}

	resmap, err := t.kustomizer.Run(fsys, ".")
	if err != nil {
		return nil, err
	}

	raw, err := resmap.AsYaml()
	if err != nil {
		return nil, err
	}

	var transformedObjects []client.Object

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
		transformedObjects = append(transformedObjects, object)
	}

	return transformedObjects, nil
}

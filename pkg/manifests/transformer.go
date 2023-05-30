/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manifests

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

type TemplateParameterTransformer struct {
	template *template.Template
}

var _ ParameterTransformer = &TemplateParameterTransformer{}

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

func (t *TemplateParameterTransformer) TransformParameters(namespace string, name string, parameters types.Unstructurable) (types.Unstructurable, error) {
	data := parameters.ToUnstructured()
	data["Namespace"] = namespace
	data["Name"] = name
	var buf bytes.Buffer
	if err := t.template.Execute(&buf, data); err != nil {
		return nil, err
	}
	var transformedParameters types.UnstructurableMap
	if err := kyaml.Unmarshal(buf.Bytes(), &transformedParameters); err != nil {
		return nil, err
	}
	return transformedParameters, nil
}

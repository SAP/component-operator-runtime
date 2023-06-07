/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
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

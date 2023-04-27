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

package templatex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"
)

// template FuncMap generator
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"toYaml":   toYaml,
		"fromYaml": fromYaml,
		"toJson":   toJson,
		"fromJson": fromJson,
		"required": required,
	}
}

// template FuncMap generator for functions called in a template context
func FuncMapForTemplate(t *template.Template) template.FuncMap {
	return template.FuncMap{
		"include": makeFuncInclude(t),
		"tpl":     makeFuncTpl(t),
	}
}

// template FuncMap generator for functions called in a Kubernetes context
func FuncMapForClient(c client.Client) template.FuncMap {
	return template.FuncMap{
		"lookup": makeFuncLookup(c),
	}
}

func toYaml(data any) (string, error) {
	raw, err := kyaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(raw), "\n"), nil
}

func fromYaml(data string) (map[string]any, error) {
	var res map[string]any
	if err := kyaml.Unmarshal([]byte(data), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func toJson(data any) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func fromJson(data string) (map[string]any, error) {
	var res map[string]any
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func required(warn string, data any) (any, error) {
	if data == nil {
		return data, errors.New(warn)
	} else if s, ok := data.(string); ok {
		if s == "" {
			return data, errors.New(warn)
		}
	}
	return data, nil
}

func makeFuncInclude(t *template.Template) func(string, any) (string, error) {
	includedNames := make(map[string]int)
	recursionMaxNums := 1000

	return func(name string, data any) (string, error) {
		var buf strings.Builder
		if v, ok := includedNames[name]; ok {
			if v > recursionMaxNums {
				return "", errors.Wrapf(fmt.Errorf("unable to execute template"), "rendering template has a nested reference name: %s", name)
			}
			includedNames[name]++
		} else {
			includedNames[name] = 1
		}
		err := t.ExecuteTemplate(&buf, name, data)
		includedNames[name]--
		return buf.String(), err
	}
}

func makeFuncTpl(t *template.Template) func(string, any) (string, error) {
	return func(text string, data any) (string, error) {
		var buf strings.Builder
		_t, err := t.Clone()
		if err != nil {
			// Clone() should never produce an error
			panic("this cannot happen")
		}
		_t = _t.New("gotpl")
		if _, err := _t.Parse(text); err != nil {
			return "", err
		}
		err = _t.Execute(&buf, data)
		return buf.String(), err
	}
}

func makeFuncLookup(c client.Client) func(string, string, string, string) (map[string]any, error) {
	return func(apiVersion string, kind string, namespace string, name string) (map[string]any, error) {
		object := &unstructured.Unstructured{}
		object.SetAPIVersion(apiVersion)
		object.SetKind(kind)
		if err := c.Get(context.Background(), apitypes.NamespacedName{Namespace: namespace, Name: name}, object); err != nil {
			if apierrors.IsNotFound(err) {
				err = nil
			}
			return map[string]any{}, err
		}
		return object.UnstructuredContent(), nil
	}
}

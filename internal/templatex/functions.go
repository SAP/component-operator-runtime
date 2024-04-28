/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package templatex

import (
	"bytes"
	"context"
	"encoding/json"
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
		"toYaml":           toYaml,
		"mustToYaml":       toYaml,
		"fromYaml":         fromYaml,
		"mustFromYaml":     fromYaml,
		"toJson":           toJson,
		"mustToJson":       toJson,
		"toPrettyJson":     toPrettyJson,
		"mustToPrettyJson": toPrettyJson,
		"toRawJson":        toRawJson,
		"mustToRawJson":    toRawJson,
		"fromJson":         fromJson,
		"mustFromJson":     fromJson,
		"required":         required,
	}
}

// template FuncMap generator for functions called in a template context
func FuncMapForTemplate(t *template.Template) template.FuncMap {
	return template.FuncMap{
		"include": makeFuncInclude(t),
		"tpl":     makeFuncTpl(t),
	}
}

// template FuncMap generator for functions called in target Kubernetes context
func FuncMapForClient(c client.Client) template.FuncMap {
	return template.FuncMap{
		"lookup":     makeFuncLookup(c, true),
		"mustLookup": makeFuncLookup(c, false),
	}
}

// template FuncMap generator for functions called in local Kubernetes context
func FuncMapForLocalClient(c client.Client) template.FuncMap {
	return template.FuncMap{
		"localLookup":     makeFuncLookup(c, true),
		"mustLocalLookup": makeFuncLookup(c, false),
	}
}

func toYaml(data any) (string, error) {
	raw, err := kyaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(raw), "\n"), nil
}

func fromYaml(data string) (any, error) {
	var res any
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

func toPrettyJson(data any) (string, error) {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func toRawJson(data any) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(&data)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func fromJson(data string) (any, error) {
	var res any
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
	return func(name string, data any) (string, error) {
		var buf strings.Builder
		err := t.ExecuteTemplate(&buf, name, data)
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

func makeFuncLookup(c client.Client, ignoreNotFound bool) func(string, string, string, string) (map[string]any, error) {
	return func(apiVersion string, kind string, namespace string, name string) (map[string]any, error) {
		object := &unstructured.Unstructured{}
		object.SetAPIVersion(apiVersion)
		object.SetKind(kind)
		if err := c.Get(context.Background(), apitypes.NamespacedName{Namespace: namespace, Name: name}, object); err != nil {
			// TODO: should apimeta.IsNoMatchError be ignored as well?
			if apierrors.IsNotFound(err) && ignoreNotFound {
				err = nil
			}
			return map[string]any{}, err
		}
		return object.UnstructuredContent(), nil
	}
}

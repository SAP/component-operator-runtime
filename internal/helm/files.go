/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"encoding/base64"
	"path"
	"strings"

	"github.com/gobwas/glob"

	kyaml "sigs.k8s.io/yaml"
)

type Files map[string][]byte

func (f Files) add(name string, data []byte) {
	if isIgnored(name) {
		return
	}
	f[name] = data
}

func (f Files) Get(name string) string {
	return string(f.GetBytes(name))
}

func (f Files) GetBytes(name string) []byte {
	if data, ok := f[name]; ok {
		return data
	}
	return []byte{}
}

func (f Files) Lines(name string) []string {
	data, ok := f[name]
	if !ok {
		return []string{}
	}
	s := string(data)
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n")
}

func (f Files) Glob(pattern string) (Files, error) {
	g, err := glob.Compile(pattern, '/')
	if err != nil {
		return nil, err
	}

	files := Files{}

	for name, data := range f {
		if g.Match(name) {
			files[name] = data
		}
	}

	return files, nil
}

func (f Files) AsConfig() string {
	configData := make(map[string]string)

	for name, data := range f {
		configData[path.Base(name)] = string(data)
	}

	// note: this must() is ok because the map can always be serialized
	return string(must(kyaml.Marshal(configData)))
}

func (f Files) AsSecrets() string {
	secretData := make(map[string]string)

	for name, data := range f {
		secretData[path.Base(name)] = base64.StdEncoding.EncodeToString(data)
	}

	// note: this must() is ok because the map can always be serialized
	return string(must(kyaml.Marshal(secretData)))
}

func isIgnored(name string) bool {
	return name == "Chart.yaml" ||
		name == "Chart.lock" ||
		name == "LICENSE" ||
		name == "README.md" ||
		name == "values.yaml" ||
		name == "values.schema.json" ||
		name == ".helmignore" ||
		strings.HasPrefix(name, "charts/") ||
		strings.HasPrefix(name, "crds/") ||
		strings.HasPrefix(name, "templates/")
}

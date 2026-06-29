/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package environment

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed all:crds
var crdfs embed.FS

var CRDS map[string]*apiextensionsv1.CustomResourceDefinition

func init() {
	CRDS = make(map[string]*apiextensionsv1.CustomResourceDefinition)

	err := fs.WalkDir(crdfs, ".", func(path string, f os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		raw, err := crdfs.ReadFile(path)
		if err != nil {
			return err
		}

		decoder := utilyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(raw))
		for {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := decoder.Decode(crd); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			CRDS[crd.Name] = crd
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

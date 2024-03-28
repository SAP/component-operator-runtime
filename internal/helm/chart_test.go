/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/helm"
	"github.com/sap/go-generics/slices"
)

var _ = Describe("testing: chart.go", func() {
	var clientset *fake.Clientset

	BeforeEach(func() {
		clientset = fake.NewSimpleClientset()
		clientset.Discovery().(*discoveryfake.FakeDiscovery).FakedServerVersion = &version.Info{
			Major: "1",
			Minor: "1",
		}
	})

	Context("using: testdata/main", func() {
		var chart *helm.Chart

		BeforeEach(func() {
			var err error
			chart, err = helm.ParseChart(os.DirFS("testdata"), "main", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("testing: Render()", func() []any {
			f := func(releaseNamespace string, releaseName string, valuesPath string, manifestsPath string) {
				values, err := loadValues(valuesPath)
				Expect(err).NotTo(HaveOccurred())
				expectedObjects, err := loadManifests(manifestsPath)
				Expect(err).NotTo(HaveOccurred())

				objects, err := chart.Render(helm.RenderContext{
					LocalClient:     nil,
					Client:          nil,
					DiscoveryClient: clientset.Discovery(),
					Release: &helm.Release{
						Namespace: releaseNamespace,
						Name:      releaseName,
						Service:   "Helm",
						IsInstall: true,
						IsUpgrade: false,
					},
					Values: values,
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(objects).To(ConsistOf(expectedObjects))
			}
			paths, err := filepath.Glob("testdata/values/*.yaml")
			Expect(err).NotTo(HaveOccurred())
			entries := slices.Collect(paths, func(path string) any {
				name := filepath.Base(path)
				return Entry(nil, "my-namespace", "my-name", path, fmt.Sprintf("testdata/manifests/%s", name))
			})
			return append([]any{f}, entries...)
		}()...,
		)
	})
})

func loadValues(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var values map[string]any
	if err := kyaml.Unmarshal(raw, &values); err != nil {
		return nil, err
	}

	return values, nil
}

func loadManifests(path string) ([]client.Object, error) {
	var objects []client.Object

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	decoder := utilyaml.NewYAMLToJSONDecoder(file)
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

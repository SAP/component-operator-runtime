/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/krusty"
	kusttypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/kustomize"
	"github.com/sap/component-operator-runtime/pkg/component"
)

var _ = Describe("testing: kustomization.go", func() {

	var namespace string

	BeforeEach(func() {
		var err error

		namespace, err = targetEnv.CreateNamespace()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should parse and render an empty kustomization", func() {
		basePath := "testdata/fs1"
		kustomizationPath := "."
		name := "test-1"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{}),
			},
			Status: component.Status{},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(1))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(0))
	})

	It("should parse and render a simple kustomization", func() {
		basePath := "testdata/fs2"
		kustomizationPath := "."
		name := "test-2"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{}),
			},
			Status: component.Status{},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(2))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})

	It("should parse and render a complex kustomization", func() {
		basePath := "testdata/fs3"
		kustomizationPath := "component"
		name := "test-3"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{
					"username": "testuser",
					"password": "testpass",
				}),
			},
			Status: component.Status{},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(11))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(3))
	})

	It("should handle options, .component-config.yaml and .component-ignore correctly", func() {
		basePath := "testdata/fs4"
		kustomizationPath := "."
		name := "test-4"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{}),
			},
			Status: component.Status{},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{
			TemplateSuffix:         new(".tpl"),
			LeftTemplateDelimiter:  new("{%"),
			RightTemplateDelimiter: new("%}"),
		}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(4))

		Expect(fsys.ReadFile("files/content.txt")).To(ContainSubstring("{{ name }}"))
		Expect(fsys.ReadFile("README")).To(Equal([]byte("The foo is: bar")))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})

	It("should handle template functions correctly", func() {
		basePath := "testdata/fs5"
		kustomizationPath := "."
		name := "test-5"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{
					"foo": "bar",
				}),
			},
			Status: component.Status{
				ProcessingDigest: "12345",
				Revision:         42,
				State:            component.StateReady,
			},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(2))

		rawResult, err := fsys.ReadFile("result.yaml")
		Expect(err).NotTo(HaveOccurred())
		result := make(map[string]any)
		err = kyaml.Unmarshal(rawResult, &result)
		Expect(err).NotTo(HaveOccurred())
		Expect(result["foo"]).To(BeEquivalentTo("bar"))
		Expect(result["componentDigest"]).To(BeEquivalentTo(component.Status.ProcessingDigest))
		Expect(result["componentRevision"]).To(BeEquivalentTo(component.Status.Revision))
		Expect(result["componentState"]).To(BeEquivalentTo(component.Status.State))
		Expect(result["kubernetesVersion"]).To(Equal(targetEnv.Version().String()))
		// TODO: add a test for the apiResources function
		Expect(result["localClusterId"]).To(Equal(localEnv.Id()))
		Expect(result["targetClusterId"]).To(Equal(targetEnv.Id()))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(0))
	})

	It("should parse and render an encrypted kustomization", func() {
		basePath := "testdata/fs6"
		kustomizationPath := "."
		name := "test-6"

		component := &Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: ComponentSpec{
				Values: serializeValues(map[string]any{}),
			},
			Status: component.Status{},
		}
		fsys, err := parseAndRender(basePath, kustomizationPath, kustomize.KustomizationOptions{
			Decryptor: &Decryptor{},
		}, component)
		Expect(err).NotTo(HaveOccurred())
		Expect(countFiles(fsys)).To(Equal(2))

		objects, err := runKustomize(fsys, kustomizationPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(objects).To(HaveLen(1))
	})

})

func parseAndRender(fspath string, kustomizationPath string, options kustomize.KustomizationOptions, component *Component) (kustfsys.FileSystem, error) {
	kustomization, err := kustomize.ParseKustomization(os.DirFS(fspath), kustomizationPath, options)
	if err != nil {
		return nil, err
	}

	fsys := kustfsys.MakeFsInMemory()

	var values map[string]any
	if err := json.Unmarshal(component.Spec.Values.Raw, &values); err != nil {
		return nil, err
	}

	err = kustomization.Render(kustomize.RenderContext{
		LocalClient:       localEnv.Client(),
		Client:            targetEnv.Client(),
		DiscoveryClient:   targetEnv.DiscoveryClient(),
		Component:         component,
		ComponentDigest:   component.Status.ProcessingDigest,
		ComponentRevision: component.Status.Revision,
		Namespace:         component.Namespace,
		Name:              component.Name,
		Parameters:        values,
	}, fsys)
	if err != nil {
		return nil, err
	}

	return fsys, nil
}

func serializeValues(values map[string]any) *apiextensionsv1.JSON {
	raw, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return &apiextensionsv1.JSON{Raw: raw}
}

func countFiles(fsys kustfsys.FileSystem) (int, error) {
	count := 0
	err := fsys.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func runKustomize(fsys kustfsys.FileSystem, path string) ([]*unstructured.Unstructured, error) {
	kustomizerOptions := &krusty.Options{
		LoadRestrictions: kusttypes.LoadRestrictionsNone,
		PluginConfig:     kusttypes.DisabledPluginConfig(),
	}
	kustomizer := krusty.MakeKustomizer(kustomizerOptions)

	resmap, err := kustomizer.Run(fsys, path)
	if err != nil {
		return nil, err
	}

	raw, err := resmap.AsYaml()
	if err != nil {
		return nil, err
	}

	var objects []*unstructured.Unstructured
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

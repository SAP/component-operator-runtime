/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package templatex

import (
	"context"
	"fmt"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gcustom"
)

var _ = Describe("testing: functions.go", func() {

	var dataObject map[string]any
	var dataObjectAsYaml string
	var dataObjectAsJson string
	var dataObjectAsPrettyJson string
	var dataObjectAsRawJson string

	var dataArray []any
	var dataArrayAsYaml string
	var dataArrayAsJson string
	var dataArrayAsPrettyJson string
	var dataArrayAsRawJson string

	BeforeEach(func() {
		dataObject = map[string]any{
			"foo":  "bar",
			"baz":  42.0,
			"html": "&nbsp;",
		}
		dataObjectAsYaml = "baz: 42\nfoo: bar\nhtml: '&nbsp;'"
		dataObjectAsJson = "{\"baz\":42,\"foo\":\"bar\",\"html\":\"\\u0026nbsp;\"}"
		dataObjectAsPrettyJson = "{\n  \"baz\": 42,\n  \"foo\": \"bar\",\n  \"html\": \"\\u0026nbsp;\"\n}"
		dataObjectAsRawJson = "{\"baz\":42,\"foo\":\"bar\",\"html\":\"&nbsp;\"}"

		dataArray = []any{
			"foo",
			42.0,
			"&nbsp;",
		}
		dataArrayAsYaml = "- foo\n- 42\n- '&nbsp;'"
		dataArrayAsJson = "[\"foo\",42,\"\\u0026nbsp;\"]"
		dataArrayAsPrettyJson = "[\n  \"foo\",\n  42,\n  \"\\u0026nbsp;\"\n]"
		dataArrayAsRawJson = "[\"foo\",42,\"&nbsp;\"]"
	})

	Describe("testing: toYaml", func() {

		It("should serialize an object to YAML", func() {
			res, err := toYaml(dataObject)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObjectAsYaml))
		})

		It("should serialize an array to YAML", func() {
			res, err := toYaml(dataArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArrayAsYaml))
		})

		It("should fail to serialize an invalid input to YAML", func() {
			invalidData := make(chan int)
			_, err := toYaml(invalidData)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: fromYaml", func() {

		It("should deserialize an object from YAML", func() {
			res, err := fromYaml(dataObjectAsYaml)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObject))
		})

		It("should deserialize an array from YAML", func() {
			res, err := fromYaml(dataArrayAsYaml)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
		})

		It("should fail to deserialize an invalid input from YAML", func() {
			invalidYaml := "foo: bar\nbaz: [1, 2, 3"
			_, err := fromYaml(invalidYaml)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: fromYamlArray", func() {

		It("should deserialize an array from YAML", func() {
			res, err := fromYamlArray(dataArrayAsYaml)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
			Expect(len(res)).To(Equal(len(dataArray)))
		})

		It("should fail to deserialize an invalid input from YAML", func() {
			invalidYaml := "[1, 2, 3"
			_, err := fromYamlArray(invalidYaml)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: toJson", func() {

		It("should serialize an object to JSON", func() {
			res, err := toJson(dataObject)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObjectAsJson))
		})

		It("should serialize an array to JSON", func() {
			res, err := toJson(dataArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArrayAsJson))
		})

		It("should fail to serialize an invalid input to JSON", func() {
			invalidData := make(chan int)
			_, err := toJson(invalidData)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: toPrettyJson", func() {

		It("should serialize an object to pretty JSON", func() {
			res, err := toPrettyJson(dataObject)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObjectAsPrettyJson))
		})

		It("should serialize an array to pretty JSON", func() {
			res, err := toPrettyJson(dataArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArrayAsPrettyJson))
		})

		It("should fail to serialize an invalid input to pretty JSON", func() {
			invalidData := make(chan int)
			_, err := toPrettyJson(invalidData)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: toRawJson", func() {

		It("should serialize an object to raw JSON", func() {
			res, err := toRawJson(dataObject)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObjectAsRawJson))
		})

		It("should serialize an array to raw JSON", func() {
			res, err := toRawJson(dataArray)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArrayAsRawJson))
		})

		It("should fail to serialize an invalid input to raw JSON", func() {
			invalidData := make(chan int)
			_, err := toRawJson(invalidData)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: fromJson", func() {

		It("should deserialize an object from JSON", func() {
			res, err := fromJson(dataObjectAsJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObject))
		})

		It("should deserialize an object from JSON", func() {
			res, err := fromJson(dataObjectAsPrettyJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObject))
		})

		It("should deserialize an object from JSON", func() {
			res, err := fromJson(dataObjectAsRawJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataObject))
		})

		It("should deserialize an array from JSON", func() {
			res, err := fromJson(dataArrayAsJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
		})

		It("should deserialize an array from JSON", func() {
			res, err := fromJson(dataArrayAsPrettyJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
		})

		It("should deserialize an array from JSON", func() {
			res, err := fromJson(dataArrayAsRawJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
		})

		It("should fail to deserialize an invalid input from JSON", func() {
			invalidJson := "{\"foo\": \"bar\", \"baz\": [1, 2, 3}"
			_, err := fromJson(invalidJson)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: fromJsonArray", func() {

		It("should deserialize an array from JSON", func() {
			res, err := fromJsonArray(dataArrayAsJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
			Expect(len(res)).To(Equal(len(dataArray)))
		})

		It("should deserialize an array from JSON", func() {
			res, err := fromJsonArray(dataArrayAsPrettyJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
			Expect(len(res)).To(Equal(len(dataArray)))
		})

		It("should deserialize an array from JSON", func() {
			res, err := fromJsonArray(dataArrayAsRawJson)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(dataArray))
			Expect(len(res)).To(Equal(len(dataArray)))
		})

		It("should fail to deserialize an invalid input from JSON", func() {
			invalidJson := "{\"foo\": \"bar\", \"baz\": [1, 2, 3}"
			_, err := fromJsonArray(invalidJson)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: required", func() {

		It("should fail with nil input", func() {
			var inp any
			res, err := required("input is required", inp)
			Expect(err).To(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should fail with empty string input", func() {
			var inp string = ""
			res, err := required("input is required", inp)
			Expect(err).To(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should fail with empty string-assertable input", func() {
			var inp any = ""
			res, err := required("input is required", inp)
			Expect(err).To(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should succeed with non-empty string input", func() {
			var inp string = "foo"
			res, err := required("input is required", inp)
			Expect(err).NotTo(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should succeed with non-empty string-assertable input", func() {
			var inp any = "foo"
			res, err := required("input is required", inp)
			Expect(err).NotTo(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should succeed with non-string (even zero) input", func() {
			var inp int = 0
			res, err := required("input is required", inp)
			Expect(err).NotTo(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

		It("should succeed with nil pointer input", func() {
			// weird, but that's the way helm does it, so we do it too
			var inp *int
			res, err := required("input is required", inp)
			Expect(err).NotTo(HaveOccurred())
			Expect(res == inp).To(BeTrue())
		})

	})

	Describe("testing: bitwiseShiftLeft", func() {

		It("should calculate the bitwise left shift correctly", func() {
			res, err := bitwiseShiftLeft(2, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(12)))
		})

	})

	Describe("testing: bitwiseShiftRight", func() {

		It("should calculate the bitwise right shift correctly", func() {
			res, err := bitwiseShiftRight(2, 14)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(3)))
		})

	})

	Describe("testing: bitwiseAnd", func() {

		It("should calculate the bitwise and correctly", func() {
			res, err := bitwiseAnd(14, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(2)))

			res, err = bitwiseAnd(12, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(0)))
		})

	})

	Describe("testing: bitwiseOr", func() {

		It("should calculate the bitwise or correctly", func() {
			res, err := bitwiseOr(14, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(15)))

			res, err = bitwiseOr(9, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(11)))
		})

	})

	Describe("testing: bitwiseXor", func() {

		It("should calculate the bitwise xor correctly", func() {
			res, err := bitwiseXor(14, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(13)))

			res, err = bitwiseXor(9, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint64(11)))
		})
	})

	Describe("testing: parseIPv4Address", func() {

		It("should parse an IPv4 address correctly", func() {
			res, err := parseIPv4Address("1.2.3.4")
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(uint32(0x01020304)))
		})

		It("should fail with invalid IPv4 address", func() {
			_, err := parseIPv4Address("256.256.256.256")
			Expect(err).To(HaveOccurred())

			_, err = parseIPv4Address("1.2.3")
			Expect(err).To(HaveOccurred())

			_, err = parseIPv4Address("1.2.3.4.5")
			Expect(err).To(HaveOccurred())

			_, err = parseIPv4Address("1.2.3.a")
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: formatIPv4Address", func() {

		It("should format an IPv4 address correctly", func() {
			res, err := formatIPv4Address(uint32(0x01020304))
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("1.2.3.4"))
		})

		It("should fail with invalid IPv4 address", func() {
			_, err := formatIPv4Address("bla")
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: include", func() {

		var t *template.Template
		var include func(string, any) (string, error)

		BeforeEach(func() {
			t = template.New("__tpl__").Option("missingkey=error")
			_, err := t.Parse(`{{  .name }}`)
			Expect(err).NotTo(HaveOccurred())
			_, err = t.New("helpers").Parse(`{{ define "hello" }}Hello, {{ .name }}!{{ end }}`)
			Expect(err).NotTo(HaveOccurred())

			include = makeFuncInclude(t)
		})

		It("should render the specified template", func() {
			res, err := include("__tpl__", map[string]any{"name": "foo"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("foo"))

			res, err = include("hello", map[string]any{"name": "stranger"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("Hello, stranger!"))
		})

		It("should fail if an error occurs while rendering the specified template", func() {
			_, err := include("hello", map[string]any{"foo": "stranger"})
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the specified template does not exist", func() {
			_, err := include("invalid", map[string]any{"name": "stranger"})
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: tpl", func() {

		var t *template.Template
		var tpl func(string, any) (string, error)

		BeforeEach(func() {
			t = template.New("__tpl__").Option("missingkey=error")
			_, err := t.Parse(`{{  .name }}`)
			Expect(err).NotTo(HaveOccurred())

			tpl = makeFuncTpl(t)
		})

		It("should render the specified template", func() {
			res, err := tpl(`Hello, {{ .name }}!`, map[string]any{"name": "stranger"})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal("Hello, stranger!"))
		})

		It("should fail if an error occurs while rendering the specified template", func() {
			_, err := tpl(`Hello, {{ .foo }}!`, map[string]any{"name": "stranger"})
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the specified template does not exist", func() {
			_, err := tpl(`{{ template "invalid" . }}`, map[string]any{"name": "stranger"})
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: lookup, lookupWithKubeConfig", func() {

		var namespace string
		var configMap *corev1.ConfigMap

		var lookup func(string, string, string, string) (map[string]any, error)
		var mustLookup func(string, string, string, string) (map[string]any, error)
		var lookupWithKubeConfig func(string, string, string, string, string) (map[string]any, error)
		var mustLookupWithKubeConfig func(string, string, string, string, string) (map[string]any, error)

		BeforeEach(func() {
			var err error

			namespace, err = env.CreateNamespace()
			Expect(err).NotTo(HaveOccurred())

			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}
			err = env.CreateObject(configMap)
			Expect(err).NotTo(HaveOccurred())

			lookup = makeFuncLookup(env.Client(), true)
			mustLookup = makeFuncLookup(env.Client(), false)
			lookupWithKubeConfig = makeFuncLookupWithKubeConfig(true)
			mustLookupWithKubeConfig = makeFuncLookupWithKubeConfig(false)
		})

		It("should find an existing object using lookup (ignoreNotFound=true)", func() {
			obj, err := lookup("v1", "ConfigMap", namespace, "c")
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(MatchConfigMap(configMap))
		})

		It("should find an existing object using lookup (ignoreNotFound=false)", func() {
			obj, err := mustLookup("v1", "ConfigMap", namespace, "c")
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(MatchConfigMap(configMap))
		})

		It("should find an existing object using lookupWithKubeConfig (ignoreNotFound=true)", func() {
			obj, err := lookupWithKubeConfig("v1", "ConfigMap", namespace, "c", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(MatchConfigMap(configMap))
		})

		It("should find an existing object using lookupWithKubeConfig (ignoreNotFound=false)", func() {
			obj, err := mustLookupWithKubeConfig("v1", "ConfigMap", namespace, "c", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(MatchConfigMap(configMap))
		})

		It("should fail on errors (other than NotFound) using lookup (ignoreNotFound=true)", func() {
			_, err := lookup("v1", "ConfigMap", namespace, "")
			Expect(err).To(HaveOccurred())
		})

		It("should fail on errors (other than NotFound) using lookup (ignoreNotFound=false)", func() {
			_, err := mustLookup("v1", "ConfigMap", namespace, "")
			Expect(err).To(HaveOccurred())
		})

		It("should fail on errors (other than NotFound) using lookupWithKubeConfig (ignoreNotFound=true)", func() {
			_, err := lookupWithKubeConfig("v1", "ConfigMap", namespace, "", env.KubeConfig())
			Expect(err).To(HaveOccurred())
		})

		It("should fail on errors (other than NotFound) using lookupWithKubeConfig (ignoreNotFound=false)", func() {
			_, err := mustLookupWithKubeConfig("v1", "ConfigMap", namespace, "", env.KubeConfig())
			Expect(err).To(HaveOccurred())
		})

		It("should return an empty object when using lookup (ignoreNotFound=true) with a non-existing object", func() {
			obj, err := lookup("v1", "ConfigMap", namespace, "d")
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(BeEmpty())
		})

		It("should return an error when using lookup (ignoreNotFound=false) with a non-existing object", func() {
			_, err := mustLookup("v1", "ConfigMap", namespace, "d")
			Expect(err).To(HaveOccurred())
		})

		It("should return an empty object when using lookupWithKubeConfig (ignoreNotFound=true) with a non-existing object", func() {
			obj, err := lookupWithKubeConfig("v1", "ConfigMap", namespace, "d", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(BeEmpty())
		})

		It("should return an error when using lookupWithKubeConfig (ignoreNotFound=false) with a non-existing object", func() {
			_, err := mustLookupWithKubeConfig("v1", "ConfigMap", namespace, "d", env.KubeConfig())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("testing: lookupList, lookupListWithKubeConfig", func() {

		var namespace1 string
		var namespace2 string
		var configMap11 *corev1.ConfigMap
		var configMap12 *corev1.ConfigMap
		var configMap21 *corev1.ConfigMap
		var configMap22 *corev1.ConfigMap

		var lookupList func(string, string, string, string) ([]map[string]any, error)
		var lookupListWithKubeConfig func(string, string, string, string, string) ([]map[string]any, error)

		BeforeEach(func() {
			var err error

			namespace1, err = env.CreateNamespace()
			Expect(err).NotTo(HaveOccurred())
			namespace2, err = env.CreateNamespace()
			Expect(err).NotTo(HaveOccurred())

			configMap11 = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c1",
					Namespace: namespace1,
					Labels: map[string]string{
						"a": "v",
					},
				},
				Data: map[string]string{
					"foo": "bar11",
				},
			}
			err = env.CreateObject(configMap11)
			Expect(err).NotTo(HaveOccurred())

			configMap12 = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c2",
					Namespace: namespace1,
					Labels: map[string]string{
						"a": "w",
					},
				},
				Data: map[string]string{
					"foo": "bar12",
				},
			}
			err = env.CreateObject(configMap12)
			Expect(err).NotTo(HaveOccurred())

			configMap21 = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c1",
					Namespace: namespace2,
					Labels: map[string]string{
						"a": "v",
					},
				},
				Data: map[string]string{
					"foo": "bar21",
				},
			}
			err = env.CreateObject(configMap21)
			Expect(err).NotTo(HaveOccurred())

			configMap22 = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c2",
					Namespace: namespace2,
				},
				Data: map[string]string{
					"foo": "bar22",
				},
			}
			err = env.CreateObject(configMap22)
			Expect(err).NotTo(HaveOccurred())

			lookupList = makeFuncLookupList(env.Client())
			lookupListWithKubeConfig = makeFuncLookupListWithKubeConfig()
		})

		AfterEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := env.CleanupObjects(ctx, configMap11, configMap12, configMap21, configMap22)
			if err != nil {
				AbortSuite(fmt.Sprintf("failed to cleanup objects: %v", err))
			}
		})

		It("should find objects using lookupList", func() {
			objs, err := lookupList("v1", "ConfigMap", "", "a")
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap12, configMap21))

			objs, err = lookupList("v1", "ConfigMap", "", "a=v")
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap21))

			objs, err = lookupList("v1", "ConfigMap", namespace1, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap12))

			objs, err = lookupList("v1", "ConfigMap", "invalid", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps())
		})

		It("should fail on errors using lookupList", func() {
			_, err := lookupList("v1", "ConfigMap", "", ";")
			Expect(err).To(HaveOccurred())
		})

		It("should find objects using lookupListWithKubeConfig", func() {
			objs, err := lookupListWithKubeConfig("v1", "ConfigMap", "", "a", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap12, configMap21))

			objs, err = lookupListWithKubeConfig("v1", "ConfigMap", "", "a=v", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap21))

			objs, err = lookupListWithKubeConfig("v1", "ConfigMap", namespace1, "", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps(configMap11, configMap12))

			objs, err = lookupListWithKubeConfig("v1", "ConfigMap", "invalid", "", env.KubeConfig())
			Expect(err).NotTo(HaveOccurred())
			Expect(objs).To(MatchConfigMaps())
		})

		It("should fail on errors using lookupListWithKubeConfig", func() {
			_, err := lookupListWithKubeConfig("v1", "ConfigMap", "", ";", env.KubeConfig())
			Expect(err).To(HaveOccurred())
		})

	})

})

func MatchConfigMap(expected *corev1.ConfigMap) OmegaMatcher {
	return MakeMatcher(func(actual map[string]any) (bool, error) {
		if actual["apiVersion"] != "v1" || actual["kind"] != "ConfigMap" {
			return false, fmt.Errorf("actual object is not a ConfigMap")
		}
		if expected == nil {
			return false, fmt.Errorf("expected object is nil")
		}
		actualObjectKey := apitypes.NamespacedName{
			Namespace: actual["metadata"].(map[string]any)["namespace"].(string),
			Name:      actual["metadata"].(map[string]any)["name"].(string),
		}
		expectedObjectKey := apitypes.NamespacedName{
			Namespace: expected.Namespace,
			Name:      expected.Name,
		}
		return Equal(expectedObjectKey).Match(actualObjectKey)
	}).WithTemplate("Expected object:\n{{.FormattedActual}}\n{{.To}} to match object:\n{{format .Data 1}}", expected)
}

func MatchConfigMaps(expected ...*corev1.ConfigMap) OmegaMatcher {
	return MakeMatcher(func(actual []map[string]any) (bool, error) {
		if len(actual) != len(expected) {
			return false, nil
		}

		var actualObjectKeys []apitypes.NamespacedName
		for _, obj := range actual {
			if obj["apiVersion"] != "v1" || obj["kind"] != "ConfigMap" {
				return false, fmt.Errorf("actual object is not a ConfigMap")
			}
			actualObjectKeys = append(actualObjectKeys, apitypes.NamespacedName{
				Namespace: obj["metadata"].(map[string]any)["namespace"].(string),
				Name:      obj["metadata"].(map[string]any)["name"].(string),
			})
		}

		var expectedObjectKeys []apitypes.NamespacedName
		for _, obj := range expected {
			if obj == nil {
				return false, fmt.Errorf("expected object is nil")
			}
			expectedObjectKeys = append(expectedObjectKeys, apitypes.NamespacedName{
				Namespace: obj.Namespace,
				Name:      obj.Name,
			})
		}

		return ConsistOf(expectedObjectKeys).Match(actualObjectKeys)
	}).WithTemplate("Expected objects:\n{{.FormattedActual}}\n{{.To}} to match objects:\n{{format .Data 1}}", expected)
}

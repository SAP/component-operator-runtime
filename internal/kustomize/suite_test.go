/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize_test

import (
	"encoding/base64"
	"os"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/types"
	"github.com/sap/component-operator-runtime/testing/environment"
)

func TestPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Package Suite: internal/kustomize")
}

var localEnv *environment.Environment
var targetEnv *environment.Environment

var _ = BeforeSuite(func() {
	By("initializing suite")
	var err error
	localEnv, err = environment.Run(os.Stdout, os.Stderr, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	targetEnv, err = environment.Run(os.Stdout, os.Stderr, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down suite")
	err := localEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = targetEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec    `json:"spec,omitempty"`
	Status component.Status `json:"status,omitempty"`
}

type ComponentSpec struct {
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}

func (c *Component) GetSpec() types.Unstructurable {
	return &c.Spec
}

func (c *Component) GetStatus() *component.Status {
	return &c.Status
}

func (s *ComponentSpec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

type Decryptor struct{}

func (d *Decryptor) Decrypt(input []byte, path string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(input))
}

/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("testing: util.go", func() {
	Context("testing: sha256hex()", func() {
		It("should generate correct sha256 digest (as hex)", func() {
			Expect(sha256hex([]byte("BzmXTiWs0KGdcDapQYmfVmzfDvEIXrCz"))).To(Equal("720350190661930afb966b47a42b98ca078f3394243f078442f2938ee3031980"))
		})
	})

	Context("testing: sha256base32()", func() {
		It("should generate correct sha256 digest (as base32)", func() {
			Expect(sha256base32([]byte("BzmXTiWs0KGdcDapQYmfVmzfDvEIXrCz"))).To(Equal("oibvagigmgjqv64wnnd2ik4yzidy6m4ueq7qpbcc6kjy5yyddgaa"))
		})
	})

	Context("testing: calculateObjectDigest()", func() {
		It("should calculate correct object digest", func() {
			object := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "default",
					Name:            "test",
					ResourceVersion: "123456789",
					Generation:      123,
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager:    "kubectl-create",
							Operation:  "Update",
							FieldsType: "FieldsV1",
							FieldsV1:   nil,
						},
					},
				},
				Data: map[string]string{
					"key": "value",
				},
			}
			savedObject := object.DeepCopy()
			digest, err := calculateObjectDigest(object)
			Expect(err).NotTo(HaveOccurred())
			Expect(object).To(Equal(savedObject))
			Expect(digest).To(Equal("e4c686ae966ae26514ad41367eb4cdc559fe24e03c3af53d6193c833e8ee80e4"))
		})
	})
})

/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"os"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/testing/environment"
)

func TestPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Package Suite: internal/clientfactory")
}

var env *environment.Environment

var _ = BeforeSuite(func() {
	By("initializing suite")
	var err error
	env, err = environment.Run(os.Stdout, os.Stderr, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	err = env.CreateObject(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "configmap-reader",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
	err = env.CreateObject(
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "configmap-reader",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Namespace: "default",
					Name:      "configmap-reader",
				},
				{
					APIGroup: rbacv1.GroupName,
					Kind:     "User",
					Name:     "configmap-reader",
				},
				{
					APIGroup: rbacv1.GroupName,
					Kind:     "Group",
					Name:     "configmap-readers",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "configmap-reader",
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down suite")
	err := env.Stop()
	Expect(err).NotTo(HaveOccurred())
})

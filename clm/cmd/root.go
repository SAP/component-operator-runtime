/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"time"

	"github.com/sap/go-generics/slices"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	fullName  = "clm.cs.sap.com"
	shortName = "clm"
)

const rootUsage = `A Kubernetes package manager

Common actions for clm:
- clm apply              Apply given component manifests to Kubernetes cluster
- clm delete             Remove component from Kubernetes cluster
- clm status             Show component status
- clm ls                 List components
`

func newRootCmd() *cobra.Command {
	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.Namespace = ref("default")

	cmd := &cobra.Command{
		Use:          shortName,
		Short:        "A Kubernetes component manager",
		Long:         rootUsage,
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	configFlags.AddFlags(cmd.PersistentFlags())

	if err := cmd.RegisterFlagCompletionFunc("namespace", func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if clnt, err := getClient(c.Flag("kubeconfig").Value.String()); err == nil {
			namespaceList := &corev1.NamespaceList{}
			ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
			defer cancel()
			if err := clnt.List(ctx, namespaceList); err == nil {
				return slices.Collect(namespaceList.Items, func(namespace corev1.Namespace) string { return namespace.Name }), cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveDefault
	}); err != nil {
		panic(err)
	}

	cmd.AddCommand(
		newVersionCmd(),
		newApplyCmd(),
		newDeleteCmd(),
		newStatusCmd(),
		newListCmd(),
	)

	return cmd
}

func Execute() error {
	return newRootCmd().Execute()
}

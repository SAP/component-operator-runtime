/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/sap/go-generics/slices"
	"github.com/spf13/cobra"

	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/clm/internal/manifests"
	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/internal/util"
)

const templateUsage = `Render component manifests to standard output without applying them to the cluster`

type templateOptions struct {
	valuesSources   []string
	targetNamespace string
	targetName      string
}

func newTemplateCmd() *cobra.Command {
	options := &templateOptions{}

	cmd := &cobra.Command{
		Use:          "template NAME SOURCE...",
		Short:        "Render component",
		Long:         templateUsage,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(2),
		PreRunE: func(c *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(c *cobra.Command, args []string) (err error) {
			name := args[0]
			manifestSources := args[1:]
			namespace := c.Flag("namespace").Value.String()

			clnt, _ := getClient(c.Flag("kubeconfig").Value.String())

			release := release.NewRelease(namespace, name, options.targetNamespace, options.targetName)
			release.Revision += 1

			objects, err := manifests.Generate(manifestSources, options.valuesSources, fullName, clnt, release)
			if err != nil {
				return err
			}

			for _, object := range objects {
				fmt.Printf("---\n%s", util.Must(kyaml.Marshal(object)))
			}

			return nil
		},
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}
			if clnt, err := getClient(c.Flag("kubeconfig").Value.String()); err == nil {
				releaseClient := release.NewClient(fullName, clnt)
				namespace := c.Flag("namespace").Value.String()
				if namespace == "" {
					namespace = "default"
				}
				ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
				defer cancel()
				if releases, err := releaseClient.List(ctx, namespace); err == nil {
					return slices.Collect(releases, func(release *release.Release) string { return release.GetName() }), cobra.ShellCompDirectiveNoFileComp
				}
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVarP(&options.valuesSources, "values", "f", nil, "Path to values file in yaml format (can be repeated, values will be merged in order of appearance)")
	flags.StringVar(&options.targetNamespace, "target-namespace", "", "Target deployment namespace for the release (defaults to the release namespace)")
	flags.StringVar(&options.targetName, "target-name", "", "Target deployment name for the release (defaults to the release name)")

	return cmd
}

/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sap/go-generics/slices"
	"github.com/spf13/cobra"

	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/clm/internal/release"
)

const statusUsage = `Show component status`

type statusOptions struct {
	outputFormat string
}

func newStatusCmd() *cobra.Command {
	options := &statusOptions{}

	cmd := &cobra.Command{
		Use:          "status NAME",
		Short:        "Show component status",
		Long:         statusUsage,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		PreRunE: func(c *cobra.Command, args []string) error {
			switch options.outputFormat {
			case "table", "yaml", "json":
				return nil
			default:
				return fmt.Errorf("invalid value for flag --%s: %s", "output", options.outputFormat)
			}
		},
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			namespace := c.Flag("namespace").Value.String()

			clnt, err := getClient(c.Flag("kubeconfig").Value.String())
			if err != nil {
				return err
			}

			releaseClient := release.NewClient(fullName, clnt)

			release, err := releaseClient.Get(context.TODO(), namespace, name)
			if err != nil {
				return err
			}

			switch options.outputFormat {
			case "table":
				details := getReleaseDetails(release)
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintf(w, "%s:\t%s\t\n", "Namespace", details.Namespace)
				fmt.Fprintf(w, "%s:\t%s\t\n", "Name", details.Name)
				fmt.Fprintf(w, "%s:\t%d\t\n", "Revision", details.Revision)
				fmt.Fprintf(w, "%s:\t%s\t\n", "State", details.State)
				fmt.Fprintf(w, "%s:\t%d\t\n", "Number of objects", details.NumAllObjects)
				fmt.Fprintf(w, "%s:\t%d\t\n", "Number of ready objects", details.NumReadyObjects)
				fmt.Fprintf(w, "%s:\t%d\t\n", "Number of completed objects", details.NumCompletedObjects)
				fmt.Fprintf(w, "%s:\t%s\t\n", "Created at", details.CreatedAt)
				fmt.Fprintf(w, "%s:\t%s\t\n", "Last updated at", details.LastUpdatedAt)
				w.Flush()
			case "yaml":
				fmt.Printf("%s", string(must(kyaml.Marshal(getReleaseDetails(release)))))
			case "json":
				fmt.Printf("%s\n", string(must(json.MarshalIndent(getReleaseDetails(release), "", "  "))))
			}

			return nil
		},
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
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
	flags.StringVarP(&options.outputFormat, "output", "o", "table", "Output format; one of \"table\", \"yaml\" or \"json\"")

	return cmd
}

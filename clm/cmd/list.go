/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sap/go-generics/slices"
	"github.com/spf13/cobra"

	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/clm/internal/release"
)

const listUsage = `List components`

type listOptions struct {
	allNamespaces bool
	outputFormat  string
}

func newListCmd() *cobra.Command {
	options := &listOptions{}

	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List components",
		Long:         listUsage,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		PreRunE: func(c *cobra.Command, args []string) error {
			switch options.outputFormat {
			case "table", "yaml", "yamlstream", "json":
				return nil
			default:
				return fmt.Errorf("invalid value for flag --%s: %s", "output", options.outputFormat)
			}
		},
		RunE: func(c *cobra.Command, args []string) error {
			namespace := c.Flag("namespace").Value.String()
			if options.allNamespaces {
				namespace = ""
			}

			clnt, err := getClient(c.Flag("kubeconfig").Value.String())
			if err != nil {
				return err
			}

			releaseClient := release.NewClient(fullName, clnt)

			releases, err := releaseClient.List(context.TODO(), namespace)
			if err != nil {
				return err
			}

			switch options.outputFormat {
			case "table":
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "NAMESPACE", "NAME", "REVISION", "STATE", "OBJECTS", "READY", "COMPLETED", "CREATED", "UPDATED")
				for _, release := range releases {
					details := getReleaseDetails(release)
					fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%d\t%d\t%s\t%s\t\n",
						details.Namespace,
						details.Name,
						details.Revision,
						details.State,
						details.NumAllObjects,
						details.NumReadyObjects,
						details.NumCompletedObjects,
						details.CreatedAt,
						details.LastUpdatedAt,
					)
				}
				w.Flush()
			case "yaml":
				fmt.Printf("%s", string(must(kyaml.Marshal(slices.Collect(releases, getReleaseDetails)))))
			case "yamlstream":
				for _, release := range releases {
					fmt.Printf("---\n%s", must(kyaml.Marshal(getReleaseDetails(release))))
				}
			case "json":
				fmt.Printf("%s\n", string(must(json.MarshalIndent(slices.Collect(releases, getReleaseDetails), "", "  "))))
			}
			return nil
		},
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List components across all namespaces")
	flags.StringVarP(&options.outputFormat, "output", "o", "table", "Output format; one of \"table\", \"yaml\", \"yamlstream\" or \"json\"")

	return cmd
}

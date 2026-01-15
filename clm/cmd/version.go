/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/version"
)

const versionUsage = `Show component lifecycle manager (clm) version`

type versionOptions struct {
	outputFormat string
}

func newVersionCmd() *cobra.Command {
	options := &versionOptions{}

	cmd := &cobra.Command{
		Use:          "version",
		Short:        "Show version",
		Long:         versionUsage,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		PreRunE: func(c *cobra.Command, args []string) error {
			switch options.outputFormat {
			case "short", "yaml", "json":
				return nil
			default:
				return fmt.Errorf("invalid value for flag --%s: %s", "output", options.outputFormat)
			}
		},
		Run: func(c *cobra.Command, args []string) {
			buildInfo := version.GetBuildInfo()
			switch options.outputFormat {
			case "short":
				fmt.Printf("%s\n", buildInfo.Version)
			case "yaml":
				fmt.Printf("%s", string(must(kyaml.Marshal(buildInfo))))
			case "json":
				fmt.Printf("%s\n", string(must(json.MarshalIndent(buildInfo, "", "  "))))
			default:
				panic("this cannot happen")
			}
		},
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.outputFormat, "output", "o", "short", "Output format; one of \"short\", \"yaml\" or \"json\"")

	return cmd
}

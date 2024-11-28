/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/sap/component-operator-runtime/clm/internal/backoff"
	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
	"github.com/sap/go-generics/slices"
)

const deleteUsage = `Delete component from Kubernetes cluster`

type deleteOptions struct {
	timeout time.Duration
}

func newDeleteCmd() *cobra.Command {
	options := &deleteOptions{}

	cmd := &cobra.Command{
		Use:          "delete NAME",
		Short:        "Delete component",
		Long:         deleteUsage,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		PreRunE: func(c *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(c *cobra.Command, args []string) (err error) {
			name := args[0]
			namespace := c.Flag("namespace").Value.String()

			clnt, err := getClient(c.Flag("kubeconfig").Value.String())
			if err != nil {
				return err
			}

			reconciler := reconciler.NewReconciler(fullName, clnt, reconciler.ReconcilerOptions{
				UpdatePolicy: ref(reconciler.UpdatePolicySsaOverride),
			})

			releaseClient := release.NewClient(fullName, clnt)

			release, err := releaseClient.Get(context.TODO(), namespace, name)
			if err != nil {
				return err
			}

			if ok, msg, err := reconciler.IsDeletionAllowed(context.TODO(), &release.Inventory); err != nil {
				return err
			} else if !ok {
				return fmt.Errorf(msg)
			}

			if err := releaseClient.Delete(context.TODO(), release); err != nil {
				return err
			}
			release, err = releaseClient.Get(context.TODO(), namespace, name)
			if err != nil {
				return err
			}

			backoff := backoff.New()

			var timeout <-chan time.Time
			if options.timeout > 0 {
				timeout = time.After(options.timeout)
			}

			defer func() {
				if err != nil {
					release.State = component.StateError
				}
				if updateErr := releaseClient.Update(context.TODO(), release); updateErr != nil {
					err = utilerrors.NewAggregate([]error{err, updateErr})
				}
			}()

			for {
				release.State = component.StateDeleting
				ok, err := reconciler.Delete(context.TODO(), &release.Inventory)
				if err != nil {
					return err
				}
				if ok {
					break
				}
				if err := releaseClient.Update(context.TODO(), release); err != nil {
					return err
				}
				select {
				case <-time.After(backoff.Next()):
				case <-timeout:
					return fmt.Errorf("timeout deleting release %s/%s", release.GetNamespace(), release.GetName())
				}
			}

			fmt.Printf("Release %s/%s successfully deleted\n", release.GetNamespace(), release.GetName())

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
	flags.DurationVar(&options.timeout, "timeout", 0, "Time to wait for the operation to complete (default is to wait forever)")

	return cmd
}

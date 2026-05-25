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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/clm/api/v1alpha1"
	"github.com/sap/component-operator-runtime/clm/internal/backoff"
	"github.com/sap/component-operator-runtime/clm/internal/manifests"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
)

const applyUsage = `Apply component manifests to Kubernetes cluster`

type applyOptions struct {
	valuesSources   []string
	createNamespace bool
	timeout         time.Duration
}

func newApplyCmd() *cobra.Command {
	options := &applyOptions{}

	cmd := &cobra.Command{
		Use:          "apply NAME SOURCE...",
		Short:        "Apply component",
		Long:         applyUsage,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(2),
		PreRunE: func(c *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(c *cobra.Command, args []string) (err error) {
			name := args[0]
			manifestSources := args[1:]
			namespace := c.Flag("namespace").Value.String()

			mgr, err := getManager(c.Flag("kubeconfig").Value.String())
			if err != nil {
				return err
			}
			clnt := mgr.GetClient()

			values, err := getValuesFromSources(options.valuesSources...)
			if err != nil {
				return err
			}

			if err := clnt.Get(context.TODO(), apitypes.NamespacedName{Name: namespace}, &corev1.Namespace{}); apierrors.IsNotFound(err) && options.createNamespace {
				if err := clnt.Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			release := &v1alpha1.Release{}
			if err := clnt.Get(context.TODO(), apitypes.NamespacedName{Namespace: namespace, Name: name}, release); apierrors.IsNotFound(err) {
				release.Namespace = namespace
				release.Name = name
				if err := clnt.Create(context.TODO(), release); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			if !release.DeletionTimestamp.IsZero() {
				return fmt.Errorf("release %s/%s is being deleted; updates are not allowed in this state", release.Namespace, release.Name)
			}

			release.Generation += 1
			release.Status.LastAttemptedDigest = manifestDigest
			release.Status.LastAttemptedRevision = manifestDigest
			release.Spec.Raw, err = kyaml.Marshal(values)
			if err != nil {
				return err
			}

			generator, err := manifests.NewGenerator(manifestSources)
			if err != nil {
				return err
			}

			// TODO: have a more generic Reconcile() which can deal with the already read release
			reconciler := component.NewReconciler[*v1alpha1.Release](fullName, generator, component.ReconcilerOptions{
				UpdatePolicy: new(reconciler.UpdatePolicySsaOverride),
			})

			if err := reconciler.SetupWithManager(mgr); err != nil {
				return err
			}

			backoff := backoff.New()

			var timeout <-chan time.Time
			if options.timeout > 0 {
				timeout = time.After(options.timeout)
			}

			/*
				defer func() {
					if err != nil {
						release.State = component.StateError
					}
					if updateErr := releaseClient.Update(context.TODO(), release); updateErr != nil {
						err = errors.Join(err, updateErr)
					}
				}()
			*/

			const maxErrCount = 15
			errCount := 0

			for {
				var wait time.Duration
				done := false
				result, err := reconciler.Reconcile(context.TODO(), ctrl.Request{})
				if err != nil {
					if !result.IsZero() {
						// TODO: log warning
					}
					// TODO: detect and handle ephemeral errors
					wait = backoff.Next()
				} else if result.RequeueAfter > 0 {
					backoff.Reset()
					wait = result.RequeueAfter
				} else if result.Requeue {
					wait = backoff.Next()
				} else {
					backoff.Reset()
					release.Status.LastAppliedDigest = release.Status.LastAttemptedDigest
					release.Status.LastAppliedRevision = release.Status.LastAttemptedRevision
					done = true
				}
			}

			/*
				clnt, err := getClient(c.Flag("kubeconfig").Value.String())
				if err != nil {
					return err
				}

				reconciler := reconciler.NewReconciler(fullName, clnt, reconciler.ReconcilerOptions{
					UpdatePolicy: new(reconciler.UpdatePolicySsaOverride),
				})

				releaseClient := release.NewClient(fullName, clnt)

				ownerId := fullName + "/" + namespace + "/" + name

				if err := clnt.Get(context.TODO(), apitypes.NamespacedName{Name: namespace}, &corev1.Namespace{}); apierrors.IsNotFound(err) && options.createNamespace {
					if err := clnt.Create(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
						return err
					}
				} else if err != nil {
					return err
				}

				release, err := releaseClient.Get(context.TODO(), namespace, name)
				if err != nil {
					if apierrors.IsNotFound(err) {
						release, err = releaseClient.Create(context.TODO(), namespace, name)
						if err != nil {
							return err
						}
					} else {
						return err
					}
				}

				if release.IsDeleting() {
					return fmt.Errorf("release %s/%s is being deleted; updates are not allowed in this state", release.GetNamespace(), release.GetName())
				}

				release.Revision += 1

				objects, err := manifests.Generate(manifestSources, options.valuesSources, fullName, clnt, release)
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

				const maxErrCount = 15
				errCount := 0

				for {
					release.State = component.StateProcessing
					ok, err := reconciler.Apply(context.TODO(), &release.Inventory, objects, namespace, ownerId, fmt.Sprintf("%d", release.Revision))
					if err != nil {
						if !isEphmeralError(err) || errCount >= maxErrCount {
							return err
						}
						errCount++
						fmt.Fprintf(os.Stderr, "Error: %s (retrying %d/%d)\n", err, errCount, maxErrCount)
					} else {
						errCount = 0
						if ok {
							release.State = component.StateReady
							break
						}
						if err := releaseClient.Update(context.TODO(), release); err != nil {
							return err
						}
					}
					select {
					case <-time.After(backoff.Next()):
					case <-timeout:
						return fmt.Errorf("timeout applying release %s/%s", release.GetNamespace(), release.GetName())
					}
				}
			*/

			fmt.Printf("Release %s/%s successfully applied\n", release.Namespace, release.Name)

			return nil
		},
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}
			if mgr, err := getManager(c.Flag("kubeconfig").Value.String()); err == nil {
				clnt := mgr.GetClient()
				namespace := c.Flag("namespace").Value.String()
				if namespace == "" {
					namespace = "default"
				}
				ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
				defer cancel()
				releaseList := &v1alpha1.ReleaseList{}
				if err := clnt.List(ctx, releaseList); err == nil {
					return slices.Collect(releaseList.Items, func(release v1alpha1.Release) string { return release.Name }), cobra.ShellCompDirectiveNoFileComp
				}
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVarP(&options.valuesSources, "values", "f", nil, "Path to values file in yaml format (can be repeated, values will be merged in order of appearance)")
	flags.BoolVar(&options.createNamespace, "create-namespace", false, "Create release namespace if not existing")
	flags.DurationVar(&options.timeout, "timeout", 0, "Time to wait for the operation to complete (default is to wait forever)")

	return cmd
}

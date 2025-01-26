/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/sap/go-generics/slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/internal/helm"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// HelmGenerator is a Generator implementation that basically renders a given Helm chart.
// A few restrictions apply to the provided Helm chart: it must not contain any subcharts, some template functions are not supported,
// some bultin variables are not supported, and hooks are processed in a slightly different fashion.
// Note: HelmGenerator's Generate() method expects local client, client and reconciler name to be set in the passed context;
// see: Context.WithLocalClient(), Context.WithClient() and Context.WithReconcilerName() in package pkg/component.
type HelmGenerator struct {
	chart *helm.Chart
}

var _ manifests.Generator = &HelmGenerator{}

// TODO: add a way to pass custom template functions

// Create a new HelmGenerator.
// The client parameter is deprecated (ignored) and will be removed in a future release.
// If fsys is nil, the local operating system filesystem will be used, and chartPath can be an absolute or relative path (in the latter case it will be considered
// relative to the current working directory). If fsys is non-nil, then chartPath should be a relative path; if an absolute path is supplied, it will be turned
// An empty chartPath will be treated like ".".
func NewHelmGenerator(fsys fs.FS, chartPath string, _ client.Client) (*HelmGenerator, error) {
	chart, err := helm.ParseChart(fsys, chartPath, nil)
	if err != nil {
		return nil, err
	}

	return &HelmGenerator{chart: chart}, nil
}

// Create a new HelmGenerator as TransformableGenerator.
func NewTransformableHelmGenerator(fsys fs.FS, chartPath string, clnt client.Client) (manifests.TransformableGenerator, error) {
	g, err := NewHelmGenerator(fsys, chartPath, clnt)
	if err != nil {
		return nil, err
	}
	return manifests.NewGenerator(g), nil
}

// Create a new HelmGenerator with a ParameterTransformer attached (further transformers can be attached to the returned generator object).
func NewHelmGeneratorWithParameterTransformer(fsys fs.FS, chartPath string, clnt client.Client, transformer manifests.ParameterTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableHelmGenerator(fsys, chartPath, clnt)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new HelmGenerator with an ObjectTransformer attached (further transformers can be attached to the returned generator object).
func NewHelmGeneratorWithObjectTransformer(fsys fs.FS, chartPath string, clnt client.Client, transformer manifests.ObjectTransformer) (manifests.TransformableGenerator, error) {
	g, err := NewTransformableHelmGenerator(fsys, chartPath, clnt)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *HelmGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	reconcilerName, err := component.ReconcilerNameFromContext(ctx)
	if err != nil {
		return nil, err
	}
	localClient, err := component.LocalClientFromContext(ctx)
	if err != nil {
		return nil, err
	}
	clnt, err := component.ClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	renderedObjects, err := g.chart.Render(helm.RenderContext{
		LocalClient:     localClient,
		Client:          clnt,
		DiscoveryClient: clnt.DiscoveryClient(),
		Release: &helm.Release{
			Namespace: namespace,
			Name:      name,
			Service:   reconcilerName,
			// TODO: probably IsInstall and IsUpgrade should be set in a more differentiated way;
			// but we don't know how, since this framework does not really distinguish between installation and upgrade ...
			IsInstall: true,
			IsUpgrade: false,
		},
		Values: parameters.ToUnstructured(),
	})
	if err != nil {
		return nil, err
	}

	annotationKeyReconcilePolicy := reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy
	annotationKeyUpdatePolicy := reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy
	annotationKeyApplyOrder := reconcilerName + "/" + types.AnnotationKeySuffixApplyOrder
	annotationKeyPurgeOrder := reconcilerName + "/" + types.AnnotationKeySuffixPurgeOrder

	for _, object := range renderedObjects {
		annotations := object.GetAnnotations()
		for key := range annotations {
			if strings.HasPrefix(key, reconcilerName+"/") {
				return nil, fmt.Errorf("annotation %s must not be set (object: %s)", key, types.ObjectKeyToString(object))
			}
		}
		hookMetadata, err := helm.ParseHookMetadata(object)
		if err != nil {
			return nil, err
		}
		if hookMetadata != nil {
			if slices.Contains(hookMetadata.Types, helm.HookTypePreDelete) {
				return nil, fmt.Errorf("helm hook type %s not supported (object: %s)", helm.HookTypePreDelete, types.ObjectKeyToString(object))
			}
			if slices.Contains(hookMetadata.Types, helm.HookTypePostDelete) {
				return nil, fmt.Errorf("helm hook type %s not supported (object: %s)", helm.HookTypePostDelete, types.ObjectKeyToString(object))
			}
			hookMetadata.Types = slices.Remove(hookMetadata.Types, helm.HookTypePreRollback)
			hookMetadata.Types = slices.Remove(hookMetadata.Types, helm.HookTypePostRollback)
			hookMetadata.Types = slices.Remove(hookMetadata.Types, helm.HookTypeTest)
			hookMetadata.Types = slices.Remove(hookMetadata.Types, helm.HookTypeTestSuccess)
			if len(hookMetadata.Types) == 0 {
				continue
			}
			if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookFailed) {
				return nil, fmt.Errorf("helm delete policy %s is not supported (object: %s)", helm.HookDeletePolicyHookFailed, types.ObjectKeyToString(object))
			}
			if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyBeforeHookCreation) {
				annotations[annotationKeyUpdatePolicy] = types.UpdatePolicyRecreate
			}
			switch {
			case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePreInstall})):
				annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnce
				annotations[annotationKeyApplyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
				if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
					annotations[annotationKeyPurgeOrder] = strconv.Itoa(-1)
				}
			case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePostInstall})):
				annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnce
				annotations[annotationKeyApplyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMinWeight + 1)
				if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
					annotations[annotationKeyPurgeOrder] = strconv.Itoa(helm.HookMaxWeight - helm.HookMinWeight + 1)
				}
			case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePreInstall, helm.HookTypePreUpgrade})):
				annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
				annotations[annotationKeyApplyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
				if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
					annotations[annotationKeyPurgeOrder] = strconv.Itoa(-1)
				}
			case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePostInstall, helm.HookTypePostUpgrade})):
				annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
				annotations[annotationKeyApplyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMinWeight + 1)
				if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
					annotations[annotationKeyPurgeOrder] = strconv.Itoa(helm.HookMaxWeight - helm.HookMinWeight + 1)
				}
			case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePreInstall, helm.HookTypePreUpgrade, helm.HookTypePostInstall, helm.HookTypePostUpgrade})):
				annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
				annotations[annotationKeyApplyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
				if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
					annotations[annotationKeyPurgeOrder] = strconv.Itoa(helm.HookMaxWeight - helm.HookMinWeight + 1)
				}
			default:
				return nil, fmt.Errorf("unsupported helm hook type combination: %s (object: %s)", strings.Join(hookMetadata.Types, ","), types.ObjectKeyToString(object))
			}
			object.SetAnnotations(annotations)
		}
		objects = append(objects, object)
	}

	return objects, nil
}

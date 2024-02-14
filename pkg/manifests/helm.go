/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/sap/go-generics/slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/helm"
	"github.com/sap/component-operator-runtime/internal/reconcile"
	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// HelmGenerator is a Generator implementation that basically renders a given Helm chart.
// A few restrictions apply to the provided Helm chart: it must not contain any subcharts, some template functions are not supported,
// some bultin variables are not supported, and hooks are processed in a slightly different fashion.
type HelmGenerator struct {
	crds      [][]byte
	templates []*template.Template
	data      map[string]any
}

var _ Generator = &HelmGenerator{}

// Create a new HelmGenerator.
// The parameter client should be a client for the local cluster (i.e. the cluster where the component object resides);
// it is used by the localLookup and mustLocalLookup template functions.
// If fsys is nil, the local operating system filesystem will be used, and chartPath can be an absolute or relative path (in the latter case it will be considered
// relative to the current working directory). If fsys is non-nil, then chartPath should be a relative path; if an absolute path is supplied, it will be turned
// An empty chartPath will be treated like ".".
func NewHelmGenerator(fsys fs.FS, chartPath string, client client.Client) (*HelmGenerator, error) {
	g := HelmGenerator{
		data: make(map[string]any),
	}

	if fsys == nil {
		fsys = os.DirFS("/")
		absoluteChartPath, err := filepath.Abs(chartPath)
		if err != nil {
			return nil, err
		}
		chartPath = absoluteChartPath[1:]
	} else if filepath.IsAbs(chartPath) {
		chartPath = chartPath[1:]
	}
	chartPath = filepath.Clean(chartPath)

	chartRaw, err := fs.ReadFile(fsys, filepath.Clean(chartPath+"/Chart.yaml"))
	if err != nil {
		return nil, err
	}
	chartData := &helm.ChartData{}
	if err := kyaml.Unmarshal(chartRaw, chartData); err != nil {
		return nil, err
	}
	if chartData.Type == "" {
		chartData.Type = helm.ChartTypeApplication
	}
	if chartData.Type != helm.ChartTypeApplication {
		return nil, fmt.Errorf("only application charts are supported")
	}
	g.data["Chart"] = chartData

	valuesRaw, err := fs.ReadFile(fsys, filepath.Clean(chartPath+"/values.yaml"))
	if err == nil {
		g.data["Values"] = &map[string]any{}
		if err := kyaml.Unmarshal(valuesRaw, g.data["Values"]); err != nil {
			return nil, err
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		g.data["Values"] = &map[string]any{}
	} else {
		return nil, err
	}

	crds, err := find(fsys, filepath.Clean(chartPath+"/crds"), "*.yaml", fileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	for _, crd := range crds {
		raw, err := fs.ReadFile(fsys, crd)
		if err != nil {
			return nil, err
		}
		g.crds = append(g.crds, raw)
	}

	includes, err := find(fsys, filepath.Clean(chartPath+"/templates"), "_*", fileTypeRegular, 0)
	if err != nil {
		return nil, err
	}

	manifests, err := find(fsys, filepath.Clean(chartPath+"/templates"), "[^_]*.yaml", fileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	if len(manifests) == 0 {
		return &g, nil
	}

	// TODO: for now, one level of library subcharts is supported
	// we should enhance the support of subcharts (nested charts, application charts)
	subChartPaths, err := find(fsys, filepath.Clean(chartPath+"/charts"), "*", fileTypeDir, 1)
	if err != nil {
		return nil, err
	}
	for _, subChartPath := range subChartPaths {
		subChartRaw, err := fs.ReadFile(fsys, subChartPath+"/Chart.yaml")
		if err != nil {
			return nil, err
		}
		var subChartData helm.ChartData
		if err := kyaml.Unmarshal(subChartRaw, &subChartData); err != nil {
			return nil, err
		}
		if subChartData.Type != helm.ChartTypeLibrary {
			return nil, fmt.Errorf("only library subcharts are supported (path: %s)", subChartPath)
		}
		subIncludes, err := find(fsys, filepath.Clean(subChartPath+"/templates"), "_*", fileTypeRegular, 0)
		if err != nil {
			return nil, err
		}
		includes = append(includes, subIncludes...)
	}

	var t *template.Template
	for _, manifest := range manifests {
		raw, err := fs.ReadFile(fsys, manifest)
		if err != nil {
			return nil, err
		}
		// Note: we use absolute paths (instead of relative ones) as template names
		// because the 'Template' builtin needs that to work properly
		if t == nil {
			t = template.New(manifest)
			t.Option("missingkey=zero").
				Funcs(sprig.TxtFuncMap()).
				Funcs(templatex.FuncMap()).
				Funcs(templatex.FuncMapForTemplate(t)).
				Funcs(templatex.FuncMapForLocalClient(client)).
				Funcs(templatex.FuncMapForClient(nil))
		} else {
			t = t.New(manifest)
		}
		if _, err := t.Parse(string(raw)); err != nil {
			return nil, err
		}
		g.templates = append(g.templates, t)
	}
	for _, include := range includes {
		raw, err := fs.ReadFile(fsys, include)
		if err != nil {
			return nil, err
		}
		// Note: we use absolute paths (instead of relative ones) as template names
		// because the 'Template' builtin needs that to work properly
		t = t.New(include)
		if _, err := t.Parse(string(raw)); err != nil {
			return nil, err
		}
	}

	return &g, nil
}

// Create a new HelmGenerator as TransformableGenerator.
func NewTransformableHelmGenerator(fsys fs.FS, chartPath string, client client.Client) (TransformableGenerator, error) {
	g, err := NewHelmGenerator(fsys, chartPath, client)
	if err != nil {
		return nil, err
	}
	return NewGenerator(g), nil
}

// Create a new HelmGenerator with a ParameterTransformer attached (further transformers can be attached to the returned generator object).
func NewHelmGeneratorWithParameterTransformer(fsys fs.FS, chartPath string, client client.Client, transformer ParameterTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableHelmGenerator(fsys, chartPath, client)
	if err != nil {
		return nil, err
	}
	return g.WithParameterTransformer(transformer), nil
}

// Create a new HelmGenerator with an ObjectTransformer attached (further transformers can be attached to the returned generator object).
func NewHelmGeneratorWithObjectTransformer(fsys fs.FS, chartPath string, client client.Client, transformer ObjectTransformer) (TransformableGenerator, error) {
	g, err := NewTransformableHelmGenerator(fsys, chartPath, client)
	if err != nil {
		return nil, err
	}
	return g.WithObjectTransformer(transformer), nil
}

// Generate resource descriptors.
func (g *HelmGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	var objects []client.Object

	reconcilerName, err := reconcile.ReconcilerNameFromContext(ctx)
	if err != nil {
		return nil, err
	}
	client, err := reconcile.ClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	annotationKeyReconcilePolicy := reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy
	annotationKeyUpdatePolicy := reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy
	annotationKeyOrder := reconcilerName + "/" + types.AnnotationKeySuffixOrder
	annotationKeyPurgeOrder := reconcilerName + "/" + types.AnnotationKeySuffixPurgeOrder

	data := make(map[string]any)
	for k, v := range g.data {
		data[k] = v
	}

	capabilities, err := helm.GetCapabilities(client.DiscoveryClient())
	if err != nil {
		return nil, err
	}
	data["Capabilities"] = capabilities

	data["Release"] = &helm.ReleaseData{
		Namespace: namespace,
		Name:      name,
		Service:   reconcilerName,
		// TODO: probably IsInstall and IsUpgrade should be set in a more differentiated way;
		// but we don't know how, since this framework does not really distinguish between installation and upgrade ...
		IsInstall: true,
		IsUpgrade: false,
	}

	data["Values"] = MergeMaps(*data["Values"].(*map[string]any), parameters.ToUnstructured())

	for _, f := range g.crds {
		decoder := utilyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(f))
		for {
			object := &unstructured.Unstructured{}
			if err := decoder.Decode(&object.Object); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			if object.Object == nil {
				continue
			}
			objects = append(objects, object)
		}
	}

	var t0 *template.Template
	if len(g.templates) > 0 {
		t0, err = g.templates[0].Clone()
		if err != nil {
			return nil, err
		}
		t0.Option("missingkey=zero").
			Funcs(templatex.FuncMapForClient(client))
	}
	for _, t := range g.templates {
		data["Template"] = &helm.TemplateData{
			Name: t.Name(),
			BasePath: func(path string) string {
				for path != "." && path != "/" {
					path = filepath.Dir(path)
					if filepath.Base(path) == "templates" {
						return path
					}
				}
				// note: this panic is ok because the way templates were selected ensures that they reside under the 'templates' directory
				panic("this cannot happen")
			}(t.Name()),
		}
		var buf bytes.Buffer
		if err := t0.ExecuteTemplate(&buf, t.Name(), data); err != nil {
			return nil, err
		}
		decoder := utilyaml.NewYAMLToJSONDecoder(&buf)
		for {
			object := &unstructured.Unstructured{}
			if err := decoder.Decode(&object.Object); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			if object.Object == nil {
				continue
			}
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
					annotations[annotationKeyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
					if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
						annotations[annotationKeyPurgeOrder] = strconv.Itoa(-1)
					}
				case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePostInstall})):
					annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnce
					annotations[annotationKeyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMinWeight + 1)
					if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
						annotations[annotationKeyPurgeOrder] = strconv.Itoa(helm.HookMaxWeight - helm.HookMinWeight + 1)
					}
				case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePreInstall, helm.HookTypePreUpgrade})):
					annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
					annotations[annotationKeyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
					if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
						annotations[annotationKeyPurgeOrder] = strconv.Itoa(-1)
					}
				case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePostInstall, helm.HookTypePostUpgrade})):
					annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
					annotations[annotationKeyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMinWeight + 1)
					if slices.Contains(hookMetadata.DeletePolicies, helm.HookDeletePolicyHookSucceeded) {
						annotations[annotationKeyPurgeOrder] = strconv.Itoa(helm.HookMaxWeight - helm.HookMinWeight + 1)
					}
				case slices.Equal(slices.Sort(hookMetadata.Types), slices.Sort([]string{helm.HookTypePreInstall, helm.HookTypePreUpgrade, helm.HookTypePostInstall, helm.HookTypePostUpgrade})):
					annotations[annotationKeyReconcilePolicy] = types.ReconcilePolicyOnObjectOrComponentChange
					annotations[annotationKeyOrder] = strconv.Itoa(hookMetadata.Weight - helm.HookMaxWeight - 1)
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
	}

	return objects, nil
}

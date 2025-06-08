/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/sap/go-generics/slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/fileutils"
	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/manifests"
)

// TODO: give errors more context
// TODO: double-check symlink handling

type RenderContext struct {
	LocalClient     client.Client
	Client          client.Client
	DiscoveryClient discovery.DiscoveryInterface
	Release         *Release
	Values          map[string]any
}

type Chart struct {
	parent    *Chart
	subCharts map[string]*Chart
	metadata  *ChartMetadata
	values    map[string]any
	crds      [][]byte
	t0        *template.Template
	templates []string
	files     Files
}

func ParseChart(fsys fs.FS, chartPath string, parent *Chart) (*Chart, error) {
	chart := &Chart{
		subCharts: make(map[string]*Chart),
	}
	if parent != nil {
		chart.parent = parent
		chart.t0 = parent.t0
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

	// TODO: we should filter out according to .helmignore

	chartRaw, err := fs.ReadFile(fsys, filepath.Clean(chartPath+"/Chart.yaml"))
	if err != nil {
		return nil, err
	}
	chart.metadata = &ChartMetadata{}
	if err := kyaml.Unmarshal(chartRaw, chart.metadata); err != nil {
		return nil, err
	}
	if chart.metadata.Type == "" {
		chart.metadata.Type = ChartTypeApplication
	}
	if chart.parent != nil && chart.parent.metadata.Type == ChartTypeLibrary && chart.metadata.Type == ChartTypeApplication {
		return nil, fmt.Errorf("library chart cannot have application subchart")
	}

	if chart.metadata.Type == ChartTypeApplication {
		crds, err := fileutils.Find(fsys, filepath.Clean(chartPath+"/crds"), "*.yaml", fileutils.FileTypeRegular, 0)
		if err != nil {
			return nil, err
		}
		for _, crd := range crds {
			raw, err := fs.ReadFile(fsys, crd)
			if err != nil {
				return nil, err
			}
			chart.crds = append(chart.crds, raw)
		}

		manifests, err := fileutils.Find(fsys, filepath.Clean(chartPath+"/templates"), "[^_]*.yaml", fileutils.FileTypeRegular, 0)
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			if err := chart.parseTemplate(fsys, manifest, false); err != nil {
				return nil, err
			}
		}
	}

	includes, err := fileutils.Find(fsys, filepath.Clean(chartPath+"/templates"), "_*", fileutils.FileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	for _, include := range includes {
		if err := chart.parseTemplate(fsys, include, true); err != nil {
			return nil, err
		}
	}

	chart.files = Files{}
	files, err := fileutils.Find(fsys, filepath.Clean(chartPath), "", fileutils.FileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		name, err := filepath.Rel(chartPath, file)
		if err != nil {
			// TODO: is it ok to panic here in case of error ?
			panic("this cannot happen")
		}
		chart.files.add(name, raw)
	}

	valuesRaw, err := fs.ReadFile(fsys, filepath.Clean(chartPath+"/values.yaml"))
	if err == nil {
		chart.values = make(map[string]any)
		if err := kyaml.Unmarshal(valuesRaw, &chart.values); err != nil {
			return nil, err
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		chart.values = make(map[string]any)
	} else {
		return nil, err
	}

	subChartPaths, err := fileutils.Find(fsys, filepath.Clean(chartPath+"/charts"), "*", fileutils.FileTypeDir, 1)
	if err != nil {
		return nil, err
	}
	for _, subChartPath := range subChartPaths {
		subChart, err := ParseChart(fsys, subChartPath, chart)
		if err != nil {
			return nil, err
		}
		subChartName := filepath.Base(subChartPath)
		chart.subCharts[subChartName] = subChart

		if slices.None(chart.metadata.Dependencies, func(dep ChartDependency) bool { return dep.Name == subChartName }) {
			chart.metadata.Dependencies = append(chart.metadata.Dependencies, ChartDependency{Name: subChartName})
		}
	}

	for i := 0; i < len(chart.metadata.Dependencies); i++ {
		dep := &chart.metadata.Dependencies[i]
		if dep.Alias == "" {
			dep.Alias = dep.Name
		}
		subChart, ok := chart.subCharts[dep.Name]
		if !ok {
			return nil, fmt.Errorf("dependent chart %s not found", dep.Name)
		}

		// TODO: validate dependency version against actual version in subchart's Chart.yaml
		// TODO: also consider Chart.lock?

		for _, val := range dep.ImportValues {
			if v, _, ok := digMap(subChart.values, val.Child); ok {
				for k, v := range v {
					if err := undig(chart.values, v, val.Parent, k); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return chart, nil
}

func (c *Chart) Render(context RenderContext) ([]client.Object, error) {
	capabilities, err := GetCapabilities(context.DiscoveryClient)
	if err != nil {
		return nil, err
	}

	var t0 *template.Template
	if c.t0 != nil {
		t0, err = c.t0.Clone()
		if err != nil {
			return nil, err
		}
		t0.Option("missingkey=zero").
			Funcs(templatex.FuncMapForTemplate(t0)).
			Funcs(templatex.FuncMapForLocalClient(context.LocalClient)).
			Funcs(templatex.FuncMapForClient(context.Client))
	}

	return c.render("", t0, capabilities, context.Release, context.Values)
}

func (c *Chart) parseTemplate(fsys fs.FS, path string, isInclude bool) error {
	var t *template.Template

	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}

	// Note: we use absolute paths (instead of relative ones) as template names
	// because the 'Template' builtin variable needs that to work properly
	if c.t0 == nil {
		c.t0 = template.New(path)
		c.t0.Option("missingkey=zero").
			Funcs(sprig.TxtFuncMap()).
			Funcs(templatex.FuncMap()).
			Funcs(templatex.FuncMapForTemplate(nil)).
			Funcs(templatex.FuncMapForLocalClient(nil)).
			Funcs(templatex.FuncMapForClient(nil))
		t = c.t0
	} else {
		t = c.t0.New(path)
	}
	if _, err := t.Parse(string(raw)); err != nil {
		return err
	}

	if !isInclude {
		c.templates = append(c.templates, path)
	}

	return nil
}

func (c *Chart) render(name string, t0 *template.Template, capabilities *Capabilities, release *Release, values map[string]any) ([]client.Object, error) {
	var objects []client.Object

	metadata := c.metadata.DeepCopy()
	if name != "" {
		metadata.Name = name
	}
	capabilities = capabilities.DeepCopy()
	release = release.DeepCopy()
	values = manifests.MergeMaps(c.values, values)

	data := make(map[string]any)
	data["Chart"] = metadata
	data["Capabilities"] = capabilities
	data["Release"] = release
	data["Values"] = values
	data["Files"] = c.files

	for _, dep := range c.metadata.Dependencies {
		enabled := true

		// evaluate tags
		// if there is no matching tag, the dependency will be considered enabled
		// otherwise if any of the tags is true, the dependency will be enabled
		// otherwise if any of the tags is false, the dependency will be disabled
		haveMatchingTag := false
		for _, tag := range dep.Tags {
			if v, exists, ok := digBool(values, "tags", tag); ok {
				if v {
					enabled = true
				} else if !haveMatchingTag {
					enabled = false
				}
				haveMatchingTag = true
			} else if exists {
				return nil, fmt.Errorf("tag references a non-boolean value")
			}
		}

		// evaluate condition
		// if a matching condition path is found, its value will win over defaults and tag-based settings
		// and the first matching condition path will terminate the evaluation of further condition paths
		if dep.Condition != "" {
			for _, cond := range strings.Split(dep.Condition, ",") {
				if v, exists, ok := digBool(values, cond); ok {
					enabled = v
					break
				} else if exists {
					return nil, fmt.Errorf("condition references a non-boolean value")
				}
			}
		}

		if !enabled {
			continue
		}

		depChart := c.subCharts[dep.Name]
		depValues, _, _ := getMap(values, dep.Alias)
		if depValues == nil {
			depValues = make(map[string]any)
		}

		if v, exists, ok := getMap(values, "global"); ok {
			if w, exists, ok := getMap(depValues, "global"); ok {
				depValues["global"] = manifests.MergeMaps(w, v)
			} else if exists {
				return nil, fmt.Errorf("values key 'global' exists but is not a map")
			} else {
				depValues["global"] = v
			}
		} else if exists {
			return nil, fmt.Errorf("values key 'global' exists but is not a map")
		}

		if v, exists, ok := getMap(values, "tags"); ok {
			if w, exists, ok := getMap(depValues, "tags"); ok {
				depValues["tags"] = manifests.MergeMaps(w, v)
			} else if exists {
				return nil, fmt.Errorf("values key 'tags' exists but is not a map")
			} else {
				depValues["tags"] = v
			}
		} else if exists {
			return nil, fmt.Errorf("values key 'tags' exists but is not a map")
		}

		depObjects, err := depChart.render(dep.Alias, t0, capabilities, release, depValues)
		if err != nil {
			return nil, err
		}
		objects = append(objects, depObjects...)
	}

	for _, f := range c.crds {
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

	if t0 != nil {
		for _, name := range c.templates {
			data["Template"] = &Template{
				// TODO: review (may be incorrect if templates dir contains a subfolder called templates)
				Name: name,
				BasePath: func(path string) string {
					for path != "." && path != "/" {
						path = filepath.Dir(path)
						if filepath.Base(path) == "templates" {
							return path
						}
					}
					// note: this panic is ok because the way templates were selected ensures that they reside under the 'templates' directory
					panic("this cannot happen")
				}(name),
			}

			var buf bytes.Buffer
			if err := t0.ExecuteTemplate(&buf, name, data); err != nil {
				return nil, err
			}

			decoder := utilyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(templatex.AdjustTemplateOutput(buf.Bytes())))
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
	}

	return objects, nil
}

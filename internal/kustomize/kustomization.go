/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-git/plumbing/format/gitignore"
	"github.com/gobwas/glob"
	"github.com/sap/go-generics/maps"
	"github.com/sap/go-generics/slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	kustfsys "sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/internal/fileutils"
	"github.com/sap/component-operator-runtime/internal/templatex"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
)

// TODO: double-check symlink handling

const (
	componentConfigFilename = ".component-config.yaml"
	componentIgnoreFilename = ".component-ignore"
)

type KustomizationOptions struct {
	TemplateSuffix *string
	// If defined, the given left delimiter will be used to parse go templates; otherwise, defaults to '{{'
	LeftTemplateDelimiter *string
	// If defined, the given right delimiter will be used to parse go templates; otherwise, defaults to '}}'
	RightTemplateDelimiter *string
	// If defined, paths to referenced files or directories outside kustomizationPath
	IncludedFiles []string
	// If defined, paths to referenced kustomizations
	IncludedKustomizations []string
	// If defined, used to decrypt files
	Decryptor manifests.Decryptor
}

type RenderContext struct {
	LocalClient     client.Client
	Client          client.Client
	DiscoveryClient discovery.DiscoveryInterface
	Component       component.Component
	ComponentDigest string
	Namespace       string
	Name            string
	Parameters      map[string]any
}

type Kustomization struct {
	path           string
	files          map[string][]byte
	nonTemplates   map[string][]byte
	templates      map[string]*template.Template
	kustomizations []*Kustomization
}

// TODO: add a way to pass custom template functions

func ParseKustomization(fsys fs.FS, kustomizationPath string, options KustomizationOptions) (*Kustomization, error) {
	kustomization, err := parseKustomization(fsys, kustomizationPath, options, nil)
	if err != nil {
		return nil, err
	}

	return kustomization, nil
}

func parseKustomization(fsys fs.FS, kustomizationPath string, options KustomizationOptions, visitedKustomizationPaths []string) (*Kustomization, error) {
	if options.TemplateSuffix == nil {
		options.TemplateSuffix = ref("")
	}
	if options.LeftTemplateDelimiter == nil {
		options.LeftTemplateDelimiter = ref("")
	}
	if options.RightTemplateDelimiter == nil {
		options.RightTemplateDelimiter = ref("")
	}

	if fsys == nil {
		fsys = os.DirFS("/")
		absoluteKustomizationPath, err := filepath.Abs(kustomizationPath)
		if err != nil {
			return nil, err
		}
		kustomizationPath = absoluteKustomizationPath[1:]
	} else if filepath.IsAbs(kustomizationPath) {
		kustomizationPath = kustomizationPath[1:]
	}
	kustomizationPath = filepath.Clean(kustomizationPath)
	if slices.Any(visitedKustomizationPaths, func(path string) bool {
		return isSubdirectory(kustomizationPath, path)
	}) {
		return nil, fmt.Errorf("path %s part of another referenced kustomization", kustomizationPath)
	}
	visitedKustomizationPaths = append(visitedKustomizationPaths, kustomizationPath)

	if info, err := fs.Stat(fsys, kustomizationPath); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", kustomizationPath)
	}

	k := Kustomization{
		path:         kustomizationPath,
		files:        make(map[string][]byte),
		nonTemplates: make(map[string][]byte),
		templates:    make(map[string]*template.Template),
	}

	if err := readOptions(fsys, filepath.Clean(filepath.Join(kustomizationPath, componentConfigFilename)), &options); err != nil {
		return nil, err
	}

	ignore, err := readIgnore(fsys, filepath.Clean(filepath.Join(kustomizationPath, componentIgnoreFilename)))
	if err != nil {
		return nil, err
	}

	var t *template.Template
	files, err := fileutils.Find(fsys, kustomizationPath, "*", fileutils.FileTypeRegular, 0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		raw, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		if options.Decryptor != nil {
			raw, err = options.Decryptor.Decrypt(raw, file)
			if err != nil {
				return nil, err
			}
		}
		name, err := filepath.Rel(kustomizationPath, file)
		if err != nil {
			// TODO: is it ok to panic here in case of error ?
			panic("this cannot happen")
		}
		k.files[name] = raw
		if filepath.Base(name) == componentConfigFilename || filepath.Base(name) == componentIgnoreFilename {
			continue
		}
		if ignore != nil && ignore.Match(filepath.SplitList(name), false) {
			continue
		}
		if strings.HasSuffix(name, *options.TemplateSuffix) {
			if t == nil {
				t = template.New(name)
				t.Delims(*options.LeftTemplateDelimiter, *options.RightTemplateDelimiter)
				t.Option("missingkey=zero").
					Funcs(sprig.TxtFuncMap()).
					Funcs(templatex.FuncMap()).
					Funcs(templatex.FuncMapForTemplate(nil)).
					Funcs(templatex.FuncMapForLocalClient(nil)).
					Funcs(templatex.FuncMapForClient(nil)).
					Funcs(funcMapForContext(nil, nil, nil, nil, "", "", ""))
			} else {
				t = t.New(name)
			}
			if _, err := t.Parse(string(raw)); err != nil {
				return nil, err
			}
			k.templates[strings.TrimSuffix(name, *options.TemplateSuffix)] = t
		} else {
			k.nonTemplates[name] = raw
		}
	}

	// TODO: check that k.nonTemplates and k.templates are disjoint

	for _, path := range options.IncludedFiles {
		if filepath.IsAbs(path) {
			return nil, fmt.Errorf("include path (%s) must be absolute", path)
		}
		absolutePath := filepath.Clean(filepath.Join(kustomizationPath, path))
		if isSubdirectory(absolutePath, kustomizationPath) {
			return nil, fmt.Errorf("include path (%s) must not be in the kustomization path (%s)", path, kustomizationPath)
		}
		if info, err := fs.Stat(fsys, absolutePath); err != nil {
			return nil, err
		} else if info.IsDir() {
			files, err := fileutils.Find(fsys, absolutePath, "*", fileutils.FileTypeRegular, 0)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				raw, err := fs.ReadFile(fsys, file)
				if err != nil {
					return nil, err
				}
				if options.Decryptor != nil {
					raw, err = options.Decryptor.Decrypt(raw, file)
					if err != nil {
						return nil, err
					}
				}
				k.files[path] = raw
			}
		} else {
			raw, err := fs.ReadFile(fsys, absolutePath)
			if err != nil {
				return nil, err
			}
			if options.Decryptor != nil {
				raw, err = options.Decryptor.Decrypt(raw, absolutePath)
				if err != nil {
					return nil, err
				}
			}
			k.files[path] = raw
		}
	}

	for _, path := range options.IncludedKustomizations {
		if filepath.IsAbs(path) {
			return nil, fmt.Errorf("include path (%s) must be absolute", path)
		}
		absolutePath := filepath.Clean(filepath.Join(kustomizationPath, path))
		if isSubdirectory(absolutePath, kustomizationPath) {
			// this is actually redundant; the same is checked through via visitedKustomizationPaths when calling parseKustomization();
			// but we keep it to maintain symmetry with the IncludedFiles handling, and because of the better error message
			return nil, fmt.Errorf("include path (%s) must not be in the kustomization path (%s)", path, kustomizationPath)
		}
		kustomization, err := parseKustomization(fsys, absolutePath, KustomizationOptions{}, visitedKustomizationPaths)
		if err != nil {
			return nil, err
		}
		k.kustomizations = append(k.kustomizations, kustomization)
	}

	return &k, nil
}

func (k *Kustomization) Path() string {
	return k.path
}

func (k *Kustomization) Render(context RenderContext, fsys kustfsys.FileSystem) error {
	serverVersion, err := context.DiscoveryClient.ServerVersion()
	if err != nil {
		return err
	}
	_, serverGroupsWithResources, err := context.DiscoveryClient.ServerGroupsAndResources()
	if err != nil {
		return err
	}
	serverGroupsWithResources = normalizeServerGroupsWithResources(serverGroupsWithResources)

	data := context.Parameters

	for n, f := range k.nonTemplates {
		if err := fsys.WriteFile(filepath.Join(k.path, n), f); err != nil {
			return err
		}
	}

	var t0 *template.Template
	for n, t := range k.templates {
		if t0 == nil {
			t0, err = t.Clone()
			if err != nil {
				return err
			}
			t0.Option("missingkey=zero").
				Funcs(templatex.FuncMapForTemplate(t0)).
				Funcs(templatex.FuncMapForLocalClient(context.LocalClient)).
				Funcs(templatex.FuncMapForClient(context.Client)).
				Funcs(funcMapForContext(k.files, serverVersion, serverGroupsWithResources, context.Component, context.ComponentDigest, context.Namespace, context.Name))
		}
		var buf bytes.Buffer
		// TODO: templates (accidentally or intentionally) could modify data, or even some of the objects supplied through builtin functions;
		// such as serverVersion or component; this should be hardened, e.g. by deep-copying things upfront, or serializing them; see the comment in
		// funcMapForContext()
		if err := t0.ExecuteTemplate(&buf, t.Name(), data); err != nil {
			return err
		}
		if err := fsys.WriteFile(filepath.Join(k.path, n), templatex.AdjustTemplateOutput(buf.Bytes())); err != nil {
			return err
		}
	}

	haveKustomization := false
	for _, kustomizationName := range konfig.RecognizedKustomizationFileNames() {
		if fsys.Exists(filepath.Join(k.path, kustomizationName)) {
			haveKustomization = true
			break
		}
	}
	if !haveKustomization {
		kustomization, err := generateKustomization(fsys, k.path)
		if err != nil {
			return err
		}
		if err := fsys.WriteFile(filepath.Join(k.path, konfig.DefaultKustomizationFileName()), kustomization); err != nil {
			return err
		}
	}

	for _, kustomization := range k.kustomizations {
		if err := kustomization.Render(context, fsys); err != nil {
			return err
		}
	}

	return nil
}

func funcMapForContext(files map[string][]byte, serverInfo *version.Info, serverGroupsWithResources []*metav1.APIResourceList, component component.Component, componentDigest string, namespace string, name string) template.FuncMap {
	return template.FuncMap{
		// TODO: maybe it would it be better to convert component to unstructured;
		// then calling methods would no longer be possible, and attributes would be in lowercase
		"listFiles":         makeFuncListFiles(files),
		"existsFile":        makeFuncExistsFile(files),
		"readFile":          makeFuncReadFile(files),
		"component":         makeFuncData(component),
		"componentDigest":   func() string { return componentDigest },
		"namespace":         func() string { return namespace },
		"name":              func() string { return name },
		"kubernetesVersion": func() *version.Info { return serverInfo },
		"apiResources":      func() []*metav1.APIResourceList { return serverGroupsWithResources },
	}
}

func makeFuncListFiles(files map[string][]byte) func(pattern string) ([]string, error) {
	return func(pattern string) ([]string, error) {
		g, err := glob.Compile(pattern, '/')
		if err != nil {
			return nil, err
		}
		return slices.Sort(slices.Select(maps.Keys(files), func(path string) bool { return g.Match(path) })), nil
	}
}

func makeFuncExistsFile(files map[string][]byte) func(path string) bool {
	return func(path string) bool {
		_, ok := files[path]
		return ok
	}
}

func makeFuncReadFile(files map[string][]byte) func(path string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return data, nil
	}
}

func makeFuncData(data any) any {
	if data == nil {
		return func() any { return nil }
	}
	ival := reflect.ValueOf(data)
	ityp := ival.Type()
	ftyp := reflect.FuncOf(nil, []reflect.Type{ityp}, false)
	fval := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value { return []reflect.Value{ival} })
	return fval.Interface()
}

// TODO: this could be simplified; the files which will be considered as resources for the generated kustomization.yaml are
// exactly those keys of the k.templates and k.nonTemplates that do not start with '.' and do not end with '.yaml' or '.yml';
// so the fsys.Walk below is basically unnecessary
func generateKustomization(fsys kustfsys.FileSystem, kustomizationPath string) ([]byte, error) {
	var resources []string

	f := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// TODO: IsDir() is false if it is a symlink; is that wanted to be this way?
		if !info.IsDir() && !strings.HasPrefix(filepath.Base(path), ".") && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			resources = append(resources, path)
		}
		return nil
	}

	// TODO: does this work correctly with symlinks?
	if err := fsys.Walk(kustomizationPath, f); err != nil {
		return nil, err
	}

	kustomization := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
		Resources: resources,
	}

	if len(resources) == 0 {
		// if there are no resources, set a dummy namespace to avoid "kustomization.yaml is empty" build error
		kustomization.Namespace = "_dummy"
	}

	rawKustomization, err := kyaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}

	return rawKustomization, nil
}

func readOptions(fsys fs.FS, path string, options *KustomizationOptions) error {
	rawOptions, err := fs.ReadFile(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	if err := kyaml.Unmarshal(rawOptions, options); err != nil {
		return err
	}

	return nil
}

func readIgnore(fsys fs.FS, path string) (gitignore.Matcher, error) {
	var patterns []gitignore.Pattern

	ignoreFile, err := fsys.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer ignoreFile.Close()

	domain := filepath.SplitList(path)
	domain = domain[0 : len(domain)-1]
	scanner := bufio.NewScanner(ignoreFile)
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, "#") && len(strings.TrimSpace(s)) > 0 {
			patterns = append(patterns, gitignore.ParsePattern(s, domain))
		}
	}

	return gitignore.NewMatcher(patterns), nil
}

func normalizeServerGroupsWithResources(serverGroupsWithResources []*metav1.APIResourceList) []*metav1.APIResourceList {
	serverGroupsWithResources = slices.SortBy(serverGroupsWithResources, func(x, y *metav1.APIResourceList) bool { return x.GroupVersion > y.GroupVersion })
	for _, serverGroupWithResources := range serverGroupsWithResources {
		serverGroupWithResources.APIResources = normalizeApiResources(serverGroupWithResources.APIResources)
	}
	return serverGroupsWithResources
}

func normalizeApiResources(apiResources []metav1.APIResource) []metav1.APIResource {
	apiResources = slices.SortBy(apiResources, func(x, y metav1.APIResource) bool { return x.Name > y.Name })
	for i := 0; i < len(apiResources); i++ {
		apiResources[i].Verbs = slices.Sort(apiResources[i].Verbs)
		apiResources[i].ShortNames = slices.Sort(apiResources[i].ShortNames)
		apiResources[i].Categories = slices.Sort(apiResources[i].Categories)
	}
	return apiResources
}

func isSubdirectory(subdir string, dir string) bool {
	return subdir == dir || strings.HasPrefix(subdir, dir+string(filepath.Separator))
}

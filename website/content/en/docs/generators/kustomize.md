---
title: "Kustomize Generator"
linkTitle: "Kustomize Generator"
weight: 10
type: "docs"
description: >
  A resource generator for kustomizations
---

This generator allows to generate the manifests of the component's resources from a given kustomization.
As a special case, one or more simple Kubernetes manifests (without a `kustomization.yaml`) are supported as well.
In addition, all (or selected; see below) files in the kustomization directory can be templatized in a helm'ish way.
That means, they will be considered as a common template group (where all templates are associated with each other),
and the same template function set that is available on Helm can be used; so, all the [sprig](http://masterminds.github.io/sprig) functions, and custom functions such as `include`, `tpl`, `lookup` can be used. In addition:
- parameterless functions `namespace` and `name` are defined, which return the corresponding arguments passed to `Generate()`
- a function `kubernetesVersion` is available, which returns the version information of the target cluster, as a [version.Info](https://pkg.go.dev/k8s.io/apimachinery/pkg/version#Info) structure.

In the generation step, first, all the go templates will be rendered, and the result of this pre-step will be passed to kustomize.

A kustomize generator can be instantiated by calling the following constructor:

```go
package kustomize

func NewKustomizeGenerator(
  fsys fs.FS,
  kustomizationPath string,
  clnt client.Client,
  options KustomizeGeneratorOptions
) (*KustomizeGenerator, error) {
```

Here:
- `fsys` must be an implementation of `fs.FS`, such as `embed.FS`; or it can be passed as nil; then, all file operations will be executed on the current OS filesystem.
- `kustomizationPath` is the directory containing the (potentially templatized) kustomatization; if `fsys` was provided, this has to be a relative path; otherwise, it will be interpreted with respect to the OS filesystem (as an absolute path, or relative to the current working directory of the controller).
- `clnt` should be a client for the local cluster (i.e. the cluster where the component object exists).
- `options` allows to tweak the generator:
  ```go
  package kustomize
  
  type KustomizeGeneratorOptions struct {
    // If defined, only files with that suffix will be subject to templating.
    TemplateSuffix *string
    // If defined, the given left delimiter will be used to parse go templates;
    // otherwise, defaults to '{{'
    LeftTemplateDelimiter *string
    // If defined, the given right delimiter will be used to parse go templates;
    // otherwise, defaults to '}}'
    RightTemplateDelimiter *string
  }
  ```

As of now, the specified kustomization must not reference files or paths outside `kustomizationPath`. Remote references are generally not supported.
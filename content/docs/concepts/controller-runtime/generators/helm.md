---
title: "Helm Generator"
linkTitle: "Helm Generator"
weight: 20
type: "docs"
description: >
  Render a Helm chart into dependent objects
---

If a component already has a productive Helm chart, the `HelmGenerator` implementation
(package `pkg/manifests/helm`) can render it:

```go
package helm

func NewHelmGenerator(fsys fs.FS, chartPath string, clnt client.Client) (*HelmGenerator, error)
```

- `fsys` must be an `fs.FS` implementation (such as an `embed.FS`), or `nil`. If `nil`,
  all file operations are performed on the OS filesystem, and `chartPath` may be an
  absolute path or a path relative to the controller's working directory. If `fsys` is
  non-nil, `chartPath` should be relative (an absolute path is turned into a relative one
  by stripping the leading slash). When using a real filesystem, `os.Root.FS()` is
  recommended over `os.DirFS()` in order to fence symbolic links. An empty `chartPath` is
  treated like `.`.
- `clnt` is **deprecated and ignored**; it will be removed in a future release. The
  clients used at generation time are taken from the context instead.

Transformable variants are available as `NewTransformableHelmGenerator()`,
`NewHelmGeneratorWithParameterTransformer()`, and
`NewHelmGeneratorWithObjectTransformer()` (see
[Transforming Existing Generators](../transformers/)).

`HelmGenerator` does not use the Helm SDK; instead it emulates Helm's behavior as closely
as possible. A few differences and restrictions arise from this:

- Most, but not all Helm template functions are supported. For example, `toToml` is not
  supported; all other functions should work but may behave more strictly on errors.
- Not all builtin variables are supported:
  - `.Release` is supported; `Release.IsInstall` is `true` during the first reconcile
    iteration (that is, when `status.revision` equals 1), and `Release.IsUpgrade` is its
    inverse; `Release.Revision` increases whenever the component manifest or one of its
    references (such as referenced secrets) changes.
  - for `.Chart`, only `.Chart.Name`, `.Chart.Version`, `.Chart.Type`, `.Chart.AppVersion`,
    and `.Chart.Dependencies` are supported.
  - for `.Capabilities`, only `.Capabilities.KubeVersion` and `.Capabilities.APIVersions`
    are supported.
  - `.Template` is fully supported.
  - `.Files` is supported but does not return any of the paths reserved by Helm (such as
    `Chart.yaml` or `templates/`).
- Regarding hooks: `pre-delete` and `post-delete` hooks are not allowed; test and
  rollback hooks are ignored; `pre-install`, `post-install`, `pre-upgrade`, and
  `post-upgrade` hooks may be handled slightly differently:
  - install hooks added later to objects of an already installed release are applied with
    the next reconcile, even though this is not the install case.
  - objects using `pre-install,post-install` or `pre-upgrade,post-upgrade` are applied
    only once per reconcile (early), and, if the deletion policy `hook-succeeded` is set,
    are deleted late.
  - obsolete hook objects (created by a hook, but no longer part of the manifest) are
    deleted immediately, unless they carry `helm.sh/resource-policy: keep` — in which case
    they are never deleted, even when the component is deleted.
  - hook weights are handled compatibly; the `hook-failed` deletion policy is not allowed,
    but `before-hook-creation` and `hook-succeeded` work as expected.
- The `.helmignore` file is currently not evaluated; files can still be accessed through
  `.Files` even if listed in `.helmignore`.

Finally, note that native component-operator-runtime object annotations, such as `mycomponent-operator.mydomain.io/apply-order`, are forbidden in Helm templates.
This is to prevent potential clashes arising from the framework translating certain Helm annotations (such as `helm.sh/hook": post-install`) into component-operator-runtime annotations. 
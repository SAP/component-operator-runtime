---
title: "Helm Generator"
linkTitle: "Helm Generator"
weight: 20
type: "docs"
description: >
  A resource generator for Helm charts
---

Sometimes it is desired to write a component operator (using component-operator-runtime) for some cluster component, which already has a productive Helm chart. Then it can make sense to use the `HelmGenerator` implementation of the `Generator` interface included in this module:

```go
package helm

func NewHelmGenerator(
  fsys                  fs.FS,
  chartPath             string,
  clnt                client.Client,
) (*HelmGenerator, error)
```

Here:
- `fsys` must be an implementation of `fs.FS`, such as `embed.FS`; or it can be passed as nil; then, all file operations will be executed on the current OS filesystem.
- `chartPath` is the directory containing the used Helm chart; if `fsys` was provided, this has to be a relative path; otherwise, it will be interpreted with respect to the OS filesystem (as an absolute path, or relative to the current working directory of the controller).
- `clnt` should be a client for the local cluster (i.e. the cluster where the component object exists).

It should be noted that `HelmGenerator` does not use the Helm SDK; instead it tries to emulate the Helm behavior as good as possible.
A few differences and restrictions arise from this:
- Not all Helm template functions are supported. To be exact, `toToml` is not supported; all other functions should be supported, but may behave more strictly in error situtations.
- Not all builtin variables are supported; the following restrictions apply:
  - the `.Release` builtin is supported; note that `Release.IsInstall` is set to `true` during the first reconcile iteration of the component (precisely, if `status.revision` equals 1), and `Release.IsUpgrade` is set to the inverse of `Release.IsInstall`; also note that `Release.Revision` increases whenever the component manifest, or one of its references (such as referenced secrets) changes
  - for the `.Chart` builtin, only `.Chart.Name`, `.Chart.Version`, `.Chart.Type`, `.Chart.AppVersion`, `.Chart.Dependencies` are supported
  - for the `.Capabilities` builtin, only `.Capabilities.KubeVersion` and `.Capabilities.APIVersions` are supported
  - the `.Template` builtin is fully supported
  - the `.Files` builtin is supported but does not return any of the paths reserved by Helm (such as `Chart.yaml`, `templates/` and so on).
- Regarding hooks, `pre-delete` and `post-delete` hooks are not allowed; test and rollback hooks are ignored, and `pre-install`, `post-install`, `pre-upgrade`, `post-upgrade` hooks might be handled in a sligthly different way:
  - install hooks added later to objects of an already installed release are applied with the next reconcile, although this is not the 'install' case (i.e. `status.revision` not equal to 1)
  - objects using `pre-install,post-install` or `pre-ugprade,post-upgrade` are applied only once per reconcile (early), and, if the deletion policy `hook-succeeded` is set, are deleted late
  - obsolete hook objects (that is, objects created by a hook, which are no longer part of the manifest) are deleted immediately, unless they have `helm.sh/resource-policy: keep`; note that in this case, they will not be deleted at all, even if the component is finally deleted.

  Hook weights will be handled in a compatible way; hook deletion policy `hook-failed` is not allowed, but `before-hook-creation` and `hook-succeeded` should work as expected.
- The `.helmignore` file is currently not evaluated; in particular, files can be accessed through `.Files` altough they are listed in `.helmignore`.
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
  - for the `.Release` builtin, only `.Release.Namespace`, `.Release.Name`, `.Release.Service`, `.Release.IsInstall`, `.Release.IsUpgrade` are supported; note that - since this framework does not really distinguish between installations and upgrades - `Release.IsInstall` is always set to `true`, and `Release.IsUpgrade` is always set to `false`
  - for the `.Chart` builtin, only `.Chart.Name`, `.Chart.Version`, `.Chart.Type`, `.Chart.AppVersion`, `.Chart.Dependencies` are supported
  - for the `.Capabilities` builtin, only `.Capabilities.KubeVersion` and `.Capabilities.APIVersions` are supported
  - the `.Template` builtin is fully supported
  - the `.Files` builtin is supported but does not return any of the paths reserved by Helm (such as `Chart.yaml`, `templates/` and so on)
- Regarding hooks, `pre-delete` and `post-delete` hooks are not allowed; test and rollback hooks are ignored, and `pre-install`, `post-install`, `pre-upgrade`, `post-upgrade` hooks might be handled in a sligthly different way; hook weights will be handled in a compatible way; hook deletion policy `hook-failed` is not allowed, but `before-hook-creation` and `hook-succeeded` should work as expected.
- The `.helmignore` file is currently not evaluated; in particular, files can be accessed through `.Files` altough they are listed in `.helmignore`.
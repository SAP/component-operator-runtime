---
title: "Scaffolder"
linkTitle: "Scaffolder"
weight: 20
type: "docs"
description: >
  A code-generation tool that bootstraps a component operator skeleton
---

## Overview

The **Scaffolder** is a CLI tool that generates a ready-to-build operator skeleton
based on [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).
The generated project is pre-wired to component-operator-runtime — the main
reconciliation loop, the component type scaffolding and status management are all set up
so you can focus on writing your domain logic rather than plumbing.

The Scaffolder is conceptually similar to [kubebuilder](https://book.kubebuilder.io/),
but more opinionated: it makes concrete choices about project layout, API conventions,
and framework integration so you do not have to.

Download the latest binary from the
[GitHub Releases](https://github.com/SAP/component-operator-runtime/releases) page.

---

## Usage

The tool is invoked as `scaffold` and writes the generated project into a target
output directory, which must already exist:

```
scaffold [options] [output directory]
```

Run `scaffold -h` to see the full list of options for the current release.

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--version` | | Show version |
| `--owner` | `SAP SE` | Owner of this project, as written to the license header |
| `--spdx-license-headers` | false | Whether to write license headers in SPDX format |
| `--group-name` | | API group name |
| `--group-version` | `v1alpha1` | API group version |
| `--kind` | | API kind for the component |
| `--resource` | pluralized kind | API resource (plural) for the component |
| `--operator-name` | | Unique name for this operator, used e.g. for leader election and labels; should be a valid DNS hostname |
| `--with-validating-webhook` | false | Whether to scaffold a validating webhook |
| `--with-mutating-webhook` | false | Whether to scaffold a mutating webhook |
| `--go-version` | `1.26.4` | Go version to be used |
| `--go-module` | | Name of the Go module, as written to the `go.mod` file |
| `--kubernetes-version` | `v0.36.0` | Kubernetes go-client version to be used |
| `--controller-runtime-version` | `v0.24.1` | controller-runtime version to be used |
| `--controller-tools-version` | `v0.21.0` | controller-tools version to be used |
| `--code-generator-version` | `v0.36.0` | code-generator version to be used |
| `--admission-webhook-runtime-version` | `v0.1.100` | admission-webhook-runtime version to be used |
| `--envtest-kubernetes-version` | `1.35.0` | Kubernetes version to be used by envtest |
| `--image` | `controller:latest` | Name of the Docker/OCI image produced by this project |
| `--skip-post-processing` | false | Skip post-processing |

### Examples

Scaffold a basic operator (without admission webhooks):

```bash
mkdir scaffold-output
scaffold \
  --group-name example.io \
  --group-version v1alpha1 \
  --kind MyComponent \
  --operator-name mycomponent-operator.example.io \
  --go-module example.io/mycomponent-operator \
  --image mycomponent-operator:latest \
  scaffold-output
```

Scaffold an operator with validating and mutating admission webhooks:

```bash
mkdir scaffold-output
scaffold \
  --group-name example.io \
  --group-version v1alpha1 \
  --kind MyComponent \
  --with-validating-webhook \
  --with-mutating-webhook \
  --operator-name mycomponent-operator.example.io \
  --go-module example.io/mycomponent-operator \
  --image mycomponent-operator:latest \
  scaffold-output
```

---

## Relationship to kubebuilder

Both tools scaffold controller-runtime projects, but the Scaffolder produces a
narrower, more complete starting point:

| | kubebuilder | Scaffolder |
|---|---|---|
| Framework integration | Manual | Pre-wired to component-operator-runtime |
| Generator choice | n/a | Helm / Kustomize / plain YAML |
| Status management | Custom | Built-in component status model |
| Reconciler implementation | Boilerplate | Delegated to `Reconciler[T]` |

If you are starting a new operator that manages a coherent set of Kubernetes resources
and want to skip the framework integration work, the Scaffolder is the fastest path.
See also the [Getting Started](../../getting-started/scaffolder/) guide for a
step-by-step walkthrough.

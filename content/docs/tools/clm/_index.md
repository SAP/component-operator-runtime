---
title: "Component Lifecycle Manager (clm)"
linkTitle: "clm"
weight: 10
type: "docs"
description: >
  A CLI tool that applies component manifests directly to a Kubernetes cluster
---

## Overview

`clm` is a command-line Kubernetes package manager that lets you apply a set of
resource manifests to a cluster without running a full operator. Under the hood it
uses the same [Object Reconciler](../../concepts/reconciler/) that powers
component-operator-runtime operators, and therefore inherits all of its reconciliation
semantics: ordered creation, server-side-apply updates, drift detection, and
inventory-based garbage collection.

State (inventory, revision, lifecycle phase) is persisted in a **ConfigMap** in the
release namespace. The ConfigMap is named `com.sap.cs.clm.release.<name>` and carries
the label `release.clm.cs.sap.com=<name>`.

`clm` is conceptually similar to the Helm CLI. In fact, most Helm charts can be
applied with `clm apply` after downloading them locally.

Download the latest binary from the
[GitHub Releases](https://github.com/SAP/component-operator-runtime/releases) page.
The source code is available at
[github.com/SAP/component-operator-runtime/tree/main/clm](https://github.com/SAP/component-operator-runtime/tree/main/clm).

To explore the tool interactively, run `clm -h` for the global options and the list of
subcommands, and `clm <subcommand> -h` for the details of an individual subcommand.

---

## Source formats

`clm` auto-detects the format of each source directory or file:

| Source  | Generator used |
|--------|-----------|----------------|
| Directory containing `Chart.yaml` | [Helm Generator](../../concepts/controller-runtime/generators/helm/) |
| Any other directory | [Kustomize Generator](../../concepts/controller-runtime/generators/kustomize/) |
| Single file | [Kustomize Generator](../../concepts/controller-runtime/generators/kustomize/) |

Multiple sources can be provided to a single `clm apply` or `clm template` call;
their rendered manifests are merged into one inventory.

---

## Global flags

The global flags apply to every subcommand and follow the standard `kubectl`
conventions:

| Flag | Default | Description |
|------|---------|-------------|
| `-n`, `--namespace` | `default` | Namespace scope for the request |
| `--kubeconfig` | — | Path to the kubeconfig file to use for CLI requests |
| `--context` | — | Name of the kubeconfig context to use |
| `--cluster` | — | Name of the kubeconfig cluster to use |
| `--user` | — | Name of the kubeconfig user to use |
| `-s`, `--server` | — | Address and port of the Kubernetes API server |
| `--token` | — | Bearer token for authentication to the API server |
| `--certificate-authority` | — | Path to a cert file for the certificate authority |
| `--client-certificate` | — | Path to a client certificate file for TLS |
| `--client-key` | — | Path to a client key file for TLS |
| `--tls-server-name` | — | Server name to use for server certificate validation |
| `--insecure-skip-tls-verify` | false | Skip validity check of the server's certificate (insecure) |
| `--disable-compression` | false | Opt out of response compression for all requests |
| `--request-timeout` | `0` | Time to wait before giving up on a single server request (`0` = no timeout) |
| `--cache-dir` | `~/.kube/cache` | Default cache directory |
| `--as` | — | Username to impersonate for the operation |
| `--as-group` | — | Group to impersonate (repeatable) |
| `--as-uid` | — | UID to impersonate |
| `--as-user-extra` | — | User extras to impersonate (repeatable) |

---

## Subcommands

### `apply`

Apply component manifests to the cluster and wait until all objects are ready.

```
clm apply NAME SOURCE... [flags]
```

| Flag | Description |
|------|-------------|
| `-f`, `--values` | Path to a values file in YAML format (repeatable; merged in order of appearance) |
| `--create-namespace` | Create the release namespace if it does not exist |
| `--timeout` | Time to wait for the operation to complete (default is to wait forever) |

`NAME` is a logical release name that uniquely identifies this deployment within the
namespace. `SOURCE` can be one or more local directories or YAML files.

**Examples**

```bash
# Apply a downloaded Helm chart with a custom values file
# helm pull oci://ghcr.io/stefanprodan/charts/podinfo --untar --untardir /tmp
# echo "replicaCount: 2" > /tmp/podinfo-values.yaml
clm -n podinfo apply --create-namespace podinfo /tmp/podinfo -f podinfo-values.yaml

# Apply a plain YAML file
# curl -f -L -o /tmp/cert-manager.yaml https://github.com/cert-manager/cert-manager/releases/download/v1.21.0/cert-manager.yaml
clm -n cert-manager apply --create-namespace cert-manager /tmp/cert-manager.yaml
```

---

### `delete`

Delete a previously applied component from the cluster, waiting until all objects
are fully removed.

```
clm delete NAME [flags]
```

| Flag | Description |
|------|-------------|
| `--timeout` | Time to wait for the operation to complete (default is to wait forever) |

**Example**

```bash
clm -n podinfo delete podinfo
```

---

### `status`

Show the current status of a component.

```
clm status NAME [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | `table` | Output format; one of `table`, `yaml`, or `json` |

The table output includes namespace, name, revision, state, total object count,
number of ready objects, number of completed objects, and creation/update timestamps.

**Examples**

```bash
clm status --namespace cert-manager cert-manager
clm status --namespace cert-manager cert-manager -o json
```

---

### `list` (alias: `ls`)

List all components in the current namespace (or across all namespaces).

```
clm list [flags]
clm ls   [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-A`, `--all-namespaces` | false | List components across all namespaces |
| `-o`, `--output` | `table` | Output format; one of `table`, `yaml`, `yamlstream`, or `json` |

**Examples**

```bash
clm ls --namespace cert-manager
clm ls -A -o json
```

---

### `template`

Render the manifests that would be applied and print them to standard output, without
touching the cluster. Useful for inspecting what `clm apply` would do.

```
clm template NAME SOURCE... [flags]
```

| Flag | Description |
|------|-------------|
| `-f`, `--values` | Path to a values file in YAML format (repeatable; merged in order of appearance) |

**Example**

```bash
clm template podinfo /tmp/podinfo --namespace podinfo
```

---

### `version`

Show the Component Lifecycle Manager (`clm`) version.

```
clm version [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | `short` | Output format; one of `short`, `yaml`, or `json` |

**Examples**

```bash
clm version
clm version -o json
```

---

### `completion`

Generate the autocompletion script for `clm` for the specified shell. See each
sub-command's help for details on how to use the generated script.

```
clm completion [command]
```

Available shells: `bash`, `fish`, `powershell`, `zsh`.

**Example**

```bash
# Load bash completion for the current shell session
. <(clm completion bash)
```
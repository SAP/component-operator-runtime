---
title: "Tools"
linkTitle: "Tools"
weight: 25
type: "docs"
description: >
  Command-line tools that ship with component-operator-runtime
---

component-operator-runtime comes with two standalone CLI tools:

- **[Component Lifecycle Manager (clm)](clm/)** — a Kubernetes package manager that
  applies component manifests from the local file system directly to a cluster,
  without requiring a running operator.

- **[Scaffolder](scaffolder/)** — a code-generation tool that bootstraps a
  fully-wired component operator skeleton, ready to be extended with domain logic.

Both tools are distributed as pre-built binaries and are available for download from
the [GitHub Releases](https://github.com/SAP/component-operator-runtime/releases) page.

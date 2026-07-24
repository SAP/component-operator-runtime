---
title: "Community"
linkTitle: "Community"
weight: 40
type: "docs"
description: >
  How to get support and how to contribute to component-operator-runtime
---

The component-operator-runtime project is hosted on GitHub at
[github.com/SAP/component-operator-runtime](https://github.com/SAP/component-operator-runtime).
Contributions of all kinds are welcome.

## Reporting Issues

If you encounter a bug, have a feature request, or want to suggest an improvement,
please [open an issue](https://github.com/SAP/component-operator-runtime/issues/new/choose)
on GitHub.

When reporting a bug, include:

- A clear description of the problem and the expected behaviour.
- The version of component-operator-runtime you are building against (the
  `github.com/sap/component-operator-runtime` entry in your `go.mod`).
- The Go and Kubernetes versions you are using.
- A minimal reproducer if possible: the relevant `Component` type, its spec, and the
  generator involved.
- Any error messages from your operator's controller logs.

## Asking Questions

For usage questions, design discussions, or anything that is not clearly a bug, please
start a thread in
[GitHub Discussions](https://github.com/SAP/component-operator-runtime/discussions)
rather than opening an issue. This keeps questions searchable for others who run into
the same topic.

## Contributing Code

Contributions are made through
[pull requests](https://github.com/SAP/component-operator-runtime/pulls) on GitHub.

1. **Fork** the repository and create a feature branch from `main`.
2. **Make your changes**, following the existing code style. Run the test suite locally
   before pushing.
3. **Open a pull request** against `main`. Describe what the change does and link any
   related issues.
4. A maintainer will review the pull request. Please respond promptly to review
   comments.

For significant changes (new features, breaking changes, large refactors), it is a good
idea to open an issue first to discuss the approach before investing time in a full
implementation.

## Code of Conduct

This project follows the
[SAP Open Source Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md).
Please be respectful and constructive in all interactions.

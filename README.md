# Kubernetes Component Operator Runtime

[![REUSE status](https://api.reuse.software/badge/github.com/SAP/component-operator-runtime)](https://api.reuse.software/info/github.com/SAP/component-operator-runtime)

## About this project

A framework to support development of Kubernetes operators managing Kubernetes components.

The operators implemented through this framework are strongly opiniated in the sense that instances
of the managed component are described through a dedicated, specific custom resource type.

The key features are:
- Efficient and smart handling of Kubernetes API extensions (custom resource definitions or API aggregation).
- Ability to fully take over rendering of the component's resources by implementing
  ```go
    type Generator interface {
        Generate(ctx context.context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
    }
  ```
  (where parameters correspond to the `spec` of the describing custom resource component object).
- Projects having existing Helm charts or Kustomizations describing the component's lifecylce can reuse these by bundling them into the
  component operator, leveraging the included ready-to-use `HelmGenerator` resp. `KustomizeGenerator` implementations.
- Scaffolding tool to bootstrap new component operators in minutes, see the [Getting Started](https://sap.github.io/component-operator-runtime/docs/getting-started/) documentation.

## Documentation

The project's documentation can be found here: [https://sap.github.io/component-operator-runtime](https://sap.github.io/component-operator-runtime).  
The API reference is here: [https://pkg.go.dev/github.com/sap/component-operator-runtime](https://pkg.go.dev/github.com/sap/component-operator-runtime).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/component-operator-runtime/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/component-operator-runtime).

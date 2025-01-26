---
title: "Kubernetes Clients"
linkTitle: "Kubernetes Clients"
weight: 40
type: "docs"
description: >
  How the framework connects to Kubernetes clusters
---

When a component resource is reconciled, two Kubernetes API clients are constructed:
- The local client; it always points to the cluster where the component resides. If the component implements impersonation (that is, the component type or its spec implements the `ImpersonationConfiguration` interface), and an impersonation user or groups are specified by the component resource, then the specified user and groups are used to impersonate the controller's kubeconfig. Otherwise, if a `DefaultServiceAccount` is defined in the reconciler's options, then that service account (relative to the components `metadata.namespace` ) is used to impersonate the controller's kubeconfig. Otherwise, the controller's kubeconfig itself is used to build the local client. The local client is passed to generators via their context. For example, the `HelmGenerator` and `KustomizeGenerator` provided by component-operator-runtime use the local client to realize the `localLookup` and `mustLocalLookup` template functions.
- The target client; if the component specifies a kubeconfig (by implementing the `ClientConfiguration` interface), then that kubeconfig is used to build the target client. Otherwise, a local client is used (possibly impersonated), created according the the logic described above. The target client is used to manage dependent objects, and is passed to generators via their context. For example, the `HelmGenerator` and `KustomizeGenerator` provided by component-operator-runtime use the target client to realize the `lookup` and `mustLookup` template functions.
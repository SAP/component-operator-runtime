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
and the well-known template function set that is available in Helm can be used; that is, all the [sprig](http://masterminds.github.io/sprig) functions, plus functions like `include`, `tpl`, `lookup`, plus further functions are supported:

| Function name                           | Description |
|-----------------------------------------|------------ |
| `toYaml <input any>`                    | Encode input to YAML string. |
| `mustToYaml <input any>`                | Same as `toYaml`. |
| `fromYaml <input string>`               | Parse YAML string to object. |
| `mustFromYaml <input string>`           | Same as `fromYaml`. |
| `fromYamlArray <input string>`          | Parse YAML string to array. |
| `mustFromYamlArray <input string>`      | Same as `fromYamlArray`. |
| `toJson <input any>`                    | Encode input to JSON string. |
| `mustToJson <input any>`                | Same as `toJson`. |
| `toPrettyJson <input any>`              | Encode object to pretty JSON string. |
| `mustToPrettyJson <input any>`          | Same as `toPrettyJson`. |
| `toRawJson <input any>   `              | Encode input to JSON string with no escaping of HTML characters. |
| `mustToRawJson <input any>   `          | Same as `toRawJson`. |
| `fromJson <input string>`               | Parse JSON string to object. |
| `mustFromJson <input string>`           | Same as `fromJson`. |
| `fromJsonArray <input string>`          | Parse JSON string to array. |
| `mustFromJsonArray <input string>`      | Same as `fromJsonArray`. |
| `required <warn string, input any>`     | Fail with the given error message if the input is `nil` or can be converted to an empty string. |
| `bitwiseShiftLeft <by any, input any>`  | Perform a bitwise left shift on the input by the given number of places. |
| `bitwiseShiftRight <by any, input any>` | Perform a bitwise right shift on the input by the given number of places. |
| `bitwiseAnd <input ...any>`             | Perform a bitwise logical 'and' on the inputs. |
| `bitwiseOr <input ...any>`              | Perform a bitwise logical 'or' on the inputs. |
| `bitwiseXor <input ...any>`             | Perform a bitwise logical 'xor' on the inputs. |
| `parseIPv4Address <input any>`          | Convert a string representation of an IPv4 address into a 32 bit integer. |
| `formatIPv4Address <input any>`         | Convert a 32 bit integer representation of an IPv4 address into a string. |
| `include <name string, input any>`      | Render the given named template with the input as data values. |
| `tpl <template string, input any>`      | Render the given template string with the input as data values. |
| `lookup <apiVersion string, kind string, namespace string, name string>`                                           | Lookup a Kubernetes resource with the target client; return nil in case of 404 (not found) errors. |
| `mustLookup <apiVersion string, kind string, namespace string, name string>`                                       | Lookup a Kubernetes resource with the target client; fail in case of 404 (not found) errors. |
| `localLookup <apiVersion string, kind string, namespace string, name string>`                                      | Lookup a Kubernetes resource with the local client; return nil in case of 404 (not found) errors. |
| `mustLocalLookup <apiVersion string, kind string, namespace string, name string>`                                  | Lookup a Kubernetes resource with the local client; fail in case of 404 (not found) errors. |
| `lookupWithKubeConfig <apiVersion string, kind string, namespace string, name string kubeConfig string> `          | Lookup a Kubernetes resource with the given kubeconfig; return nil in case of 404 (not found) errors. |
| `mustLookupWithKubeConfig <apiVersion string, kind string, namespace string, name string, kubeConfig string>`      | Lookup a Kubernetes resource with the given kubeconfig; fail in case of 404 (not found) errors. |
| `lookupList <apiVersion string, kind string, namespace string, labelSelector string>`                                           | Lookup (list) Kubernetes resources with the target client. |
| `localLookupList <apiVersion string, kind string, namespace string, labelSelector string>`                                      | Lookup (list) Kubernetes resources with the local client. |
| `lookupListWithKubeConfig <apiVersion string, kind string, namespace string, labelSelector string kubeConfig string> `          | Lookup (list) Kubernetes resources with the given kubeconfig. |
| `listFiles <pattern string>`                                                                                       | List files relative to the provided kustomization directory,  matching the given [pattern](https://pkg.go.dev/github.com/gobwas/glob). | 
| `existsFile <path string>`                                                                                         | Check if the given file path exists, relative to the provided kustomization directory. |
| `readFile <path string>`                                                                                           | Read the given file, relative to the provided kustomization directory. |
| `namespace`                                                                                                        | Return the deployment namespace as passed to the generator. |
| `name`                                                                                                             | Return the deployment name as passed to the generator. |
| `kubernetesVersion`                                                                                                | Return a `*version.Info` [struct](https://pkg.go.dev/k8s.io/apimachinery/pkg/version#Info) containing Kubernetes version details about the deployment target. |
| `apiResources`                                                                                                     | Return a slice of `[]*metav1.APIResourceList` [structs](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#APIResourceList) containing Kubernetes API discovery details about the deployment target. |

Notes:
- in case a function is provided by sprig and by us, our implementation is used
- all functions (unless otherwise stated) make the using template fail in case of errors
- as for the lookup functions, the target client points to the deployment target, the local client points to the cluster where the executing controller is running.

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
  TemplateSuffix *string
  // If defined, the given left delimiter will be used to parse go templates;
  // otherwise, defaults to '{{'
  LeftTemplateDelimiter *string
  // If defined, the given right delimiter will be used to parse go templates;
  // otherwise, defaults to '}}'
  RightTemplateDelimiter *string
  // If defined, used to decrypt files
  Decryptor manifests.Decryptor
  }
  ```

  The generator options can be overridden on source level by creating a file `.component-config.yaml` in the specified `kustomizationPath`; the file can contain JSON or YAML, compatible with the  `KustomizationsOptions` struct:

  ```go
  package kustomize

  type KustomizationOptions struct {
    TemplateSuffix *string
    // If defined, the given left delimiter will be used to parse go templates; otherwise, defaults to '{{'
    LeftTemplateDelimiter *string
    // If defined, the given right delimiter will be used to parse go templates; otherwise, defaults to '}}'
    RightTemplateDelimiter *string
    // If defined, paths to referenced files or directories outside kustomizationPath
    IncludedFiles []string
    // If defined, paths to referenced kustomizations
    IncludedKustomizations []string
    // If defined, used to decrypt files
    Decryptor manifests.Decryptor
  }
  ```

By default, the specified kustomization cannot reference files or paths on `fsys` outside `kustomizationPath`.
By default, all `.yaml` or `.yml` files in `kustomizationPath`, and its subdirectories, are subject to templating, and are considered if a `kustomization.yaml` is auto-generated. It is possible to exclude certain files from templating by creating a file `.component-ignore` in `kustomizationPath`; this `.component-ignore` file uses the common `.gitignore` syntax. Note that excluded files are still visible to the `readFile` template function. Furthermore, additional file outside the `kustomizationPath` can be referenced if the according paths are declared in `.component-config.yaml` as:
- `includedKustomizations: []string`: a list of directory paths relative to `kustomizationPath`; targeted directories are treated as own components, rendered with the including component's parameters (values), and then supplied to kustomize at the identical path
- `includedFiles: []string`: a list of paths relative to `kustomizationPath` (single files or directories); all referenced files (recursively in case a directory is specified) can be used with `readFile`.

Recursive inclusions are possible, but must not lead to cycles (there is a circuit breaking logic that will fail the generator in case of cycles).

Finally, note that remote references are not supported at all.
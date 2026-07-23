---
title: "Kustomize Generator"
linkTitle: "Kustomize Generator"
weight: 30
type: "docs"
description: >
  Render a (templatized) kustomization into dependent objects
---

The `KustomizeGenerator` (package `pkg/manifests/kustomize`) renders the component's
resources from a given kustomization. As a special case, one or more plain Kubernetes
manifests (without a `kustomization.yaml`) are supported as well. In addition, all (or
selected) files in the kustomization directory can be templatized in a Helm-ish way:
they are treated as a common template group, and the well-known template function set
available in Helm can be used — that is, all [sprig](http://masterminds.github.io/sprig)
functions, plus functions like `include`, `tpl`, `lookup`, and the following:

| Function name | Description |
|---|---|
| `toYaml <input any>` | Encode input to YAML string. |
| `mustToYaml <input any>` | Same as `toYaml`. |
| `fromYaml <input string>` | Parse YAML string to object. |
| `mustFromYaml <input string>` | Same as `fromYaml`. |
| `fromYamlArray <input string>` | Parse YAML string to array. |
| `mustFromYamlArray <input string>` | Same as `fromYamlArray`. |
| `toJson <input any>` | Encode input to JSON string. |
| `mustToJson <input any>` | Same as `toJson`. |
| `toPrettyJson <input any>` | Encode object to pretty JSON string. |
| `mustToPrettyJson <input any>` | Same as `toPrettyJson`. |
| `toRawJson <input any>` | Encode input to JSON string with no escaping of HTML characters. |
| `mustToRawJson <input any>` | Same as `toRawJson`. |
| `fromJson <input string>` | Parse JSON string to object. |
| `mustFromJson <input string>` | Same as `fromJson`. |
| `fromJsonArray <input string>` | Parse JSON string to array. |
| `mustFromJsonArray <input string>` | Same as `fromJsonArray`. |
| `required <warn string, input any>` | Fail with the given error message if the input is `nil` or converts to an empty string. |
| `bitwiseShiftLeft <by any, input any>` | Perform a bitwise left shift on the input by the given number of places. |
| `bitwiseShiftRight <by any, input any>` | Perform a bitwise right shift on the input by the given number of places. |
| `bitwiseAnd <input ...any>` | Perform a bitwise logical 'and' on the inputs. |
| `bitwiseOr <input ...any>` | Perform a bitwise logical 'or' on the inputs. |
| `bitwiseXor <input ...any>` | Perform a bitwise logical 'xor' on the inputs. |
| `parseIPv4Address <input any>` | Convert a string representation of an IPv4 address into a 32 bit integer. |
| `formatIPv4Address <input any>` | Convert a 32 bit integer representation of an IPv4 address into a string. |
| `include <name string, input any>` | Render the given named template with the input as data values. |
| `tpl <template string, input any>` | Render the given template string with the input as data values. |
| `lookup <apiVersion, kind, namespace, name string>` | Lookup a resource with the target client; return nil on 404. |
| `mustLookup <apiVersion, kind, namespace, name string>` | Lookup a resource with the target client; fail on 404. |
| `localLookup <apiVersion, kind, namespace, name string>` | Lookup a resource with the local client; return nil on 404. |
| `mustLocalLookup <apiVersion, kind, namespace, name string>` | Lookup a resource with the local client; fail on 404. |
| `lookupWithKubeConfig <apiVersion, kind, namespace, name, kubeConfig string>` | Lookup a resource with the given kubeconfig; return nil on 404. |
| `mustLookupWithKubeConfig <apiVersion, kind, namespace, name, kubeConfig string>` | Lookup a resource with the given kubeconfig; fail on 404. |
| `lookupList <apiVersion, kind, namespace, labelSelector string>` | Lookup (list) resources with the target client. |
| `localLookupList <apiVersion, kind, namespace, labelSelector string>` | Lookup (list) resources with the local client. |
| `lookupListWithKubeConfig <apiVersion, kind, namespace, labelSelector, kubeConfig string>` | Lookup (list) resources with the given kubeconfig. |
| `listFiles <pattern string>` | List files relative to the kustomization directory, matching the given [pattern](https://pkg.go.dev/github.com/gobwas/glob). |
| `existsFile <path string>` | Check if the given file path exists, relative to the kustomization directory. |
| `readFile <path string>` | Read the given file, relative to the kustomization directory. |
| `componentDigest` | Return the digest of the component (considering spec, annotations, and references such as secrets). |
| `componentRevision` | Return the revision of the component; a counter increased whenever a new digest is applied to the cluster. |
| `namespace` | Return the deployment namespace as passed to the generator. |
| `name` | Return the deployment name as passed to the generator. |
| `kubernetesVersion` | Return a `*version.Info` [struct](https://pkg.go.dev/k8s.io/apimachinery/pkg/version#Info) with Kubernetes version details about the target. |
| `apiResources` | Return a slice of `[]*metav1.APIResourceList` [structs](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#APIResourceList) with API discovery details about the target. |

Notes:

- if a function is provided by both sprig and this framework, the framework's
  implementation is used;
- unless otherwise stated, all functions make the template fail on errors;
- for the lookup functions, the *target* client points to the deployment target, while
  the *local* client points to the cluster where the controller runs.

During generation, all Go templates are rendered first, and the result of this pre-step
is then passed to kustomize.

The generator is instantiated with:

```go
package kustomize

func NewKustomizeGenerator(fsys fs.FS, kustomizationPath string, clnt client.Client, options KustomizeGeneratorOptions) (*KustomizeGenerator, error)
```

- `fsys` and `kustomizationPath` behave exactly like `fsys`/`chartPath` for the
  [Helm generator](../helm/) (nil filesystem means the OS filesystem, absolute paths are
  normalized, `os.Root.FS()` is recommended, empty path means `.`).
- `clnt` is **deprecated and ignored**; the clients used at generation time are taken from
  the context.
- `options` tweaks the generator:

  ```go
  package kustomize

  type KustomizeGeneratorOptions struct {
  	// If defined, only files with the given suffix are considered as templates.
  	TemplateSuffix *string
  	// If defined, the given left delimiter is used to parse go templates; defaults to '{{'.
  	LeftTemplateDelimiter *string
  	// If defined, the given right delimiter is used to parse go templates; defaults to '}}'.
  	RightTemplateDelimiter *string
  	// If defined, used to decrypt files.
  	Decryptor manifests.Decryptor
  }
  ```

Transformable variants are available as `NewTransformableKustomizeGenerator()`,
`NewKustomizeGeneratorWithParameterTransformer()`, and
`NewKustomizeGeneratorWithObjectTransformer()` (see
[Transforming Existing Generators](../transformers/)).

The generator can additionally be tuned on source level by placing a
`.component-config.yaml` file (JSON or YAML) in the `kustomizationPath`, compatible with:

```go
package kustomize

type KustomizationConfiguration struct {
	// If defined, only files with the given suffix are considered as templates.
	TemplateSuffix *string
	// If defined, the given left delimiter is used to parse go templates; defaults to '{{'.
	LeftTemplateDelimiter *string
	// If defined, the given right delimiter is used to parse go templates; defaults to '}}'.
	RightTemplateDelimiter *string
	// If defined, paths to referenced files or directories outside kustomizationPath.
	IncludedFiles []string
	// If defined, paths to referenced kustomizations.
	IncludedKustomizations []string
	// If defined, default values for the templates.
	Values map[string]any
}
```

Where a property appears both in `options` and in `.component-config.yaml`, the config
file takes precedence.

By default, the kustomization cannot reference files or paths on `fsys` outside
`kustomizationPath`, and all `.yaml`/`.yml` files in `kustomizationPath` (and its
subdirectories) are subject to templating and considered when a `kustomization.yaml` is
auto-generated. Specific files can be excluded from templating by creating a
`.component-ignore` file (using `.gitignore` syntax) in `kustomizationPath`; excluded
files remain visible to the `readFile` function. Files outside `kustomizationPath` can be
referenced by declaring them in `.component-config.yaml`:

- `includedKustomizations`: directory paths (relative to `kustomizationPath`) that are
  treated as own components, rendered with the including component's values, and then
  supplied to kustomize at the identical path; recursive inclusions are possible but must
  not form cycles (a circuit breaker fails the generator on cycles).
- `includedFiles`: paths (relative to `kustomizationPath`) to single files or directories,
  whose contents can then be used with `readFile`.

Remote references are not supported.

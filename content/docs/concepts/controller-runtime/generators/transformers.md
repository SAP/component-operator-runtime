---
title: "Transforming Existing Generators"
linkTitle: "Transforming Generators"
weight: 10
type: "docs"
description: >
  Wrap an existing generator to adjust its input parameters or output objects
---

Sometimes it is desirable to reuse an existing [generator](../), but adjust its *input*
(the parameters passed to the generation step) or its *output* (the object manifests it
returns) — for example to inject computed values, or to patch rendered objects. This is
done by wrapping any `Generator` into a `TransformableGenerator`:

```go
package manifests

// Interface for generators that can be enhanced with parameter/object transformers.
type TransformableGenerator interface {
	Generator
	WithParameterTransformer(transformer ParameterTransformer) TransformableGenerator
	WithObjectTransformer(transformer ObjectTransformer) TransformableGenerator
}

// Wrap an existing generator into a TransformableGenerator.
func NewGenerator(generator Generator) TransformableGenerator
```

Transformers are attached through the fluent `WithParameterTransformer()` and
`WithObjectTransformer()` methods (which return the generator again, so calls can be
chained). Multiple transformers of each kind can be attached and are applied in order.
The transformation logic itself is provided by implementing:

```go
package manifests

// Parameter transformer interface.
// Allows to manipulate the parameters passed to an existing generator.
type ParameterTransformer interface {
	TransformParameters(namespace string, name string, parameters types.Unstructurable) (types.Unstructurable, error)
}

// Object transformer interface.
// Allows to manipulate the objects returned by an existing generator.
type ObjectTransformer interface {
	TransformObjects(namespace string, name string, objects []client.Object) ([]client.Object, error)
}
```

Parameter transformers run before the wrapped generator's `Generate()` and may rewrite
the incoming parameters; object transformers run afterwards and may rewrite the produced
objects.

The framework also ships a few ready-made transformers:

- `TemplateParameterTransformer` (`NewTemplateParameterTransformer(fsys, path)`) —
  transforms the parameters through a Go template (with the sprig function set plus
  `toYaml`, `fromYaml`, `toJson`, `fromJson`, `required`, and related helpers; see `FuncMap()` in `internal/templatex`).
- `SubstitutionObjectTransformer` (`NewSubstitutionObjectTransformer(substitutions, selector)`) —
  applies environment-variable-style substitutions to the rendered objects matching the
  given selector.
- `KustomizeObjectTransformer` (`NewKustomizeObjectTransformer(patches, images)`) —
  post-processes the rendered objects by applying kustomize patches and image overrides.

For convenience, both built-in generators expose `NewTransformable…Generator()` and
`New…GeneratorWith(Parameter|Object)Transformer()` constructors, so a transformable
generator can be created in a single call.

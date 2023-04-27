---
title: "Enhance Existing Generators"
linkTitle: "Enhance Existing Generators"
weight: 30
type: "docs"
description: >
  How to construct generators from existing generators
---

In some cases it is desirable to modify the behaviour of an existing generator by transforming the
input parameters passed to the generation step, or by transforming the object manifests returned by the
generation step. This can be achieved by wrapping an existing generator into a

```go
package manifests

type TransformableGenerator interface {
	Generator
	WithParameterTransformer(transformer ParameterTransformer) TransformableGenerator
	WithObjectTransformer(transformer ObjectTransformer) TransformableGenerator
}
```

object tby calling

```go
package manifests

func NewGenerator(generator Generator) TransformableGenerator
```

The generator obtained this way can now be extended by calling its methods `WithParameterTransformer()` and `WithObjectTransformer()`. The actual modification logic happens by implementing the respective interfaces

```go
package manifests

type ParameterTransformer interface {
	TransformParameters(parameters types.Unstructurable) (types.Unstructurable, error)
}

type ObjectTransformer interface {
	TransformObjects(objects []client.Object) ([]client.Object, error)
}
```
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {% name >>
data:
  foo: {% .foo >>
  bar: {% .bar >>
  baz: {% .baz >>
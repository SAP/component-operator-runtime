apiVersion: v2
name: {{ .operatorName | splitList "." | first }}
description: A Helm chart for https://{{ .goModule }}
type: application
version: 0.1.0
appVersion: v0.1.0

{{- define "selectorLabels" -}}
app: complex-app
{{- end -}}

{{- define "commonLabels" -}}
{{ include "selectorLabels" . }}
purpose: testing
{{- end -}}
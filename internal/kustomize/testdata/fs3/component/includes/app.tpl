{{- define "image" -}}
{{- with .image -}}
{{ .repository | default "complex-app" }}:{{ .tag | default (include "version" .) }}
{{- else -}}
nginx:latest
{{- end -}}
{{- end -}}

{{- define "replicas" -}}
{{ .replicas | default 1 }}
{{- end -}}
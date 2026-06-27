{{- define "stage" -}}
{{ .stage | default "prod" }}
{{- end -}}

{{- define "logLevel" -}}
{{ (include "stage" . | printf "../config/%s.yaml" | readFile | toString | fromYaml).logLevel }}
{{- end -}}
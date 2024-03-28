{{- define "sub12.t1" -}}
val-from-sub12.t1
{{- end -}}

{{- define "templateName" -}}
{{ regexReplaceAll "^(.*)(templates(?:/[^/]+)?)$" .Template.Name (printf "%s/${2}" .Chart.Name) }}
{{- end }}

{{- define "templateBasePath" -}}
{{ regexReplaceAll "^(.*)(templates(?:/[^/]+)?)$" .Template.BasePath (printf "%s/${2}" .Chart.Name) }}
{{- end }}
{{- define "version" -}}
{{ readFile "../../version" | toString }}
{{- end -}}
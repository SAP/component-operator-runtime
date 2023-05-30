{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
{
  "CN": "local-ca",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
{{- end }}

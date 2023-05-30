{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
{{- $name := (split "." .operatorName)._0 }}
{
  "CN": "{{ $name }}-webhook.default.svc.cluster.local",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "hosts": [
    "{{ $name }}-webhook.default.svc.cluster.local",
    "{{ $name }}-webhook.default.svc",
    "{{ $name }}-webhook.default",
    "*.internal",
    "localhost"
  ]
}
{{- end }}

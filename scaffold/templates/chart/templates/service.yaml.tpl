{{- $operator := (splitList "." .operatorName | first) -}}
{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
---
apiVersion: v1
kind: Service
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
spec:
  type: {{`{{`}} .Values.service.type {{`}}`}}
  ports:
    - port: {{`{{`}} .Values.service.port {{`}}`}}
      targetPort: webhook
      protocol: TCP
      name: https
  selector:
    {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 4 {{`}}`}}
{{- end }}

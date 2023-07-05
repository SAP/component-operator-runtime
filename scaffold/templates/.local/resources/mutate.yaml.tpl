{{- if .mutatingWebhookEnabled -}}
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ (split "." .operatorName)._0 }}
webhooks:
- name: mutate.{{ .resource }}.{{ .groupName }}
  admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${WEBHOOK_CA_CERT}
    url: https://${WEBHOOK_HOSTNAME}:2443/admission/{{ .groupName }}/{{ .groupVersion }}/{{ .resource }}/mutate
  rules:
  - apiGroups:
    - {{ .groupName }}
    apiVersions:
    - {{ .groupVersion }}
    operations:
    - CREATE
    - UPDATE
    resources:
    - {{ .resource }}
    scope: Namespaced
  matchPolicy: Equivalent
  sideEffects: None
  timeoutSeconds: 10
  failurePolicy: Fail
  reinvocationPolicy: Never
  {{- end }}

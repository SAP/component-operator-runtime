{{- if .validatingWebhookEnabled -}}
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ (split "." .operatorName)._0 }}
webhooks:
- name: {{ .operatorName }}
  admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: ${WEBHOOK_CA_CERT}
    url: https://${WEBHOOK_HOSTNAME}:2443/admission/{{ .groupName }}/{{ .groupVersion }}/{{ .resource }}/validate
  rules:
  - apiGroups:
    - {{ .groupName }}
    apiVersions:
    - {{ .groupVersion }}
    operations:
    - CREATE
    - UPDATE
    - DELETE
    resources:
    - {{ .resource }}
    scope: Namespaced
  matchPolicy: Equivalent
  sideEffects: None
  timeoutSeconds: 10
  failurePolicy: Fail
  {{- end }}

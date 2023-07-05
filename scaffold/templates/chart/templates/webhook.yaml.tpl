{{- $operator := (splitList "." .operatorName | first) -}}
{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
{{`{{`}}- $caCert := "" {{`}}`}}
{{`{{`}}- if .Values.webhook.certManager.enabled {{`}}`}}
{{`{{`}}- if not .Values.webhook.certManager.issuerName {{`}}`}}
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
spec:
  selfSigned: {}
{{`{{`}}- end {{`}}`}}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
spec:
  dnsNames:
  - {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  - {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}.{{`{{`}} .Release.Namespace {{`}}`}}
  - {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}.{{`{{`}} .Release.Namespace {{`}}`}}.svc
  - {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}.{{`{{`}} .Release.Namespace {{`}}`}}.svc.cluster.local
  issuerRef:
    {{`{{`}}- if .Values.webhook.certManager.issuerName {{`}}`}}
    {{`{{`}}- with .Values.webhook.certManager.issuerGroup {{`}}`}}
    group: {{`{{`}} . {{`}}`}}
    {{`{{`}}- end {{`}}`}}
    {{`{{`}}- with .Values.webhook.certManager.issuerKind {{`}}`}}
    kind: {{`{{`}} . {{`}}`}}
    {{`{{`}}- end {{`}}`}}
    name: {{`{{`}} .Values.webhook.certManager.issuerName {{`}}`}}
    {{`{{`}}- else {{`}}`}}
    name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
    {{`{{`}}- end {{`}}`}}
  secretName: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}-tls-managed
{{`{{`}}- else {{`}}`}}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}-tls
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
type: Opaque
data:
  {{`{{`}}- $data := (lookup "v1" "Secret" .Release.Namespace (printf "%s-tls" (include "{{ $operator }}.fullname" .))).data {{`}}`}}
  {{`{{`}}- if $data {{`}}`}}
  {{`{{`}} $data | toYaml | nindent 2 {{`}}`}}
  {{`{{`}}- $caCert = index $data "ca.crt" {{`}}`}}
  {{`{{`}}- else {{`}}`}}
  {{`{{`}}- $cn := printf "%s.%s.svc" (include "{{ $operator }}.fullname" .) .Release.Namespace {{`}}`}}
  {{`{{`}}- $ca := genCA (printf "%s-ca" (include "{{ $operator }}.fullname" .)) 36500 {{`}}`}}
  {{`{{`}}- $cert := genSignedCert $cn nil (list $cn) 36500 $ca {{`}}`}}
  ca.crt: {{`{{`}} $ca.Cert | b64enc {{`}}`}}
  tls.crt: {{`{{`}} $cert.Cert | b64enc {{`}}`}}
  tls.key: {{`{{`}} $cert.Key | b64enc {{`}}`}}
  {{`{{`}}- $caCert = $ca.Cert | b64enc {{`}}`}}
  {{`{{`}}- end {{`}}`}}
{{`{{`}}- end {{`}}`}}
{{- end }}
{{- if .validatingWebhookEnabled }}
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
  {{`{{`}}- if .Values.webhook.certManager.enabled {{`}}`}}
  annotations:
    cert-manager.io/inject-ca-from: {{`{{`}} .Release.Namespace {{`}}`}}/{{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  {{`{{`}}- end {{`}}`}}
webhooks:
- name: validate.{{ .resource }}.{{ .groupName }}
  admissionReviewVersions:
  - v1
  clientConfig:
    {{`{{`}}- if not .Values.webhook.certManager.enabled {{`}}`}}
    caBundle: {{`{{`}} $caCert {{`}}`}}
    {{`{{`}}- end {{`}}`}}
    service:
      namespace: {{`{{`}} .Release.Namespace {{`}}`}}
      name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
      port: {{`{{`}} .Values.service.port {{`}}`}}
      path: /admission/{{ .groupName }}/{{ .groupVersion }}/{{ .resource }}/validate
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
{{- if .mutatingWebhookEnabled }}
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
  {{`{{`}}- if .Values.webhook.certManager.enabled {{`}}`}}
  annotations:
    cert-manager.io/inject-ca-from: {{`{{`}} .Release.Namespace {{`}}`}}/{{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  {{`{{`}}- end {{`}}`}}
webhooks:
- name: mutate.{{ .resource }}.{{ .groupName }}
  admissionReviewVersions:
  - v1
  clientConfig:
    {{`{{`}}- if not .Values.webhook.certManager.enabled {{`}}`}}
    caBundle: {{`{{`}} $caCert {{`}}`}}
    {{`{{`}}- end {{`}}`}}
    service:
      namespace: {{`{{`}} .Release.Namespace {{`}}`}}
      name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
      port: {{`{{`}} .Values.service.port {{`}}`}}
      path: /admission/{{ .groupName }}/{{ .groupVersion }}/{{ .resource }}/mutate
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

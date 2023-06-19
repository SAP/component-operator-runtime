{{- $operator := (splitList "." .operatorName | first) -}}
{{- $webhooksEnabled := or .validatingWebhookEnabled .mutatingWebhookEnabled -}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
    {{ .operatorName }}/ignored: "true"
spec:
  replicas: {{`{{`}} .Values.replicaCount {{`}}`}}
  selector:
    matchLabels:
      {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 6 {{`}}`}}
  template:
    metadata:
      annotations:
        {{`{{`}}- with .Values.podAnnotations {{`}}`}}
        {{`{{`}}- toYaml . | nindent 8 {{`}}`}}
        {{`{{`}}- end {{`}}`}}
      labels:
        {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 8 {{`}}`}}
        {{`{{`}}- with .Values.podLabels {{`}}`}}
        {{`{{`}}- toYaml . | nindent 8 {{`}}`}}
        {{`{{`}}- end {{`}}`}}
    spec:
      {{`{{`}}- with .Values.imagePullSecrets {{`}}`}}
      imagePullSecrets:
      {{`{{`}}- toYaml . | nindent 6 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.podSecurityContext {{`}}`}}
      securityContext:
        {{`{{`}}- toYaml . | nindent 8 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.nodeSelector {{`}}`}}
      nodeSelector:
        {{`{{`}}- toYaml . | nindent 8 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.affinity {{`}}`}}
      affinity:
        {{`{{`}}- toYaml . | nindent 8 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.topologySpreadConstraints {{`}}`}}
      topologySpreadConstraints:
      {{`{{`}}- range . {{`}}`}}
      - {{`{{`}} toYaml . | trim | nindent 8 {{`}}`}}
        {{`{{`}}- if not .labelSelector {{`}}`}}
        labelSelector:
          matchLabels:
            {{`{{`}}- include "{{ $operator }}.selectorLabels" $ | nindent 12 {{`}}`}}
        {{`{{`}}- end {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- else {{`}}`}}
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        nodeTaintsPolicy: Honor
        whenUnsatisfiable: {{`{{`}} .Values.defaultHostNameSpreadPolicy  {{`}}`}}
        labelSelector:
          matchLabels:
            {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 12 {{`}}`}}
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        nodeTaintsPolicy: Honor
        whenUnsatisfiable: {{`{{`}} .Values.defaultZoneSpreadPolicy  {{`}}`}}
        labelSelector:
          matchLabels:
            {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 12 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.tolerations {{`}}`}}
      tolerations:
      {{`{{`}}- toYaml . | nindent 6 {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      {{`{{`}}- with .Values.priorityClassName {{`}}`}}
      priorityClassName: {{`{{`}} . {{`}}`}}
      {{`{{`}}- end {{`}}`}}
      serviceAccountName: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
      automountServiceAccountToken: true
      containers:
      - name: manager
        image: {{`{{`}} .Values.image.repository {{`}}`}}:{{`{{`}} .Values.image.tag | default .Chart.AppVersion {{`}}`}}
        imagePullPolicy: {{`{{`}} .Values.image.pullPolicy {{`}}`}}
        args:
        - --leader-elect
        ports:
        {{- if $webhooksEnabled }}
        - name: webhook
          containerPort: 2443
          protocol: TCP
        {{- end }}
        - name: metrics
          containerPort: 8080
          protocol: TCP
        - name: probes
          containerPort: 8081
          protocol: TCP
        {{`{{`}}- with .Values.securityContext {{`}}`}}
        securityContext:
          {{`{{`}}- toYaml . | nindent 12 {{`}}`}}
        {{`{{`}}- end {{`}}`}}
        resources:
          {{`{{`}}- toYaml .Values.resources | nindent 12 {{`}}`}}
        livenessProbe:
          httpGet:
            port: probes
            scheme: HTTP
            path: /healthz
        readinessProbe:
          httpGet:
            port: probes
            scheme: HTTP
            path: /readyz
        {{- if $webhooksEnabled }}
        volumeMounts:
        - name: tls
          mountPath: /tmp/k8s-webhook-server/serving-certs
        {{- end }}
      {{- if $webhooksEnabled }}
      volumes:
      - name: tls
        secret:
          secretName: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}-{{`{{`}} ternary "tls-managed" "tls" .Values.webhook.certManager.enabled {{`}}`}}
      {{- end }}

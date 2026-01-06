{{- $operator := (splitList "." .operatorName | first) -}}
{{`{{`}}- if .Values.pdb.enabled {{`}}`}}
{{`{{`}}- if ge (int .Values.replicaCount) 2 {{`}}`}}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  {{`{{`}}- with .Values.pdb.annotations {{`}}`}}
  annotations:
    {{`{{`}}- toYaml . | nindent 4 {{`}}`}}
  {{`{{`}}- end {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
    {{`{{`}}- with .Values.pdb.labels {{`}}`}}
    {{`{{`}}- toYaml . | nindent 4 {{`}}`}}
    {{`{{`}}- end {{`}}`}}
spec:
  {{`{{`}}- if .Values.pdb.maxUnavailable {{`}}`}}
  maxUnavailable: {{ .Values.pdb.maxUnavailable {{`}}`}}
  {{`{{`}}- else }}
  minAvailable: {{ .Values.pdb.minAvailable {{`}}`}}
  {{`{{`}}- end {{`}}`}}
  selector:
    matchLabels:
      {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 6 {{`}}`}}
{{`{{`}}- end {{`}}`}}
{{`{{`}}- end {{`}}`}}
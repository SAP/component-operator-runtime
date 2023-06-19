{{- $operator := (splitList "." .operatorName | first) -}}
{{`{{`}}- if ge (int .Values.replicaCount) 2 {{`}}`}}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
spec:
  minAvailable: 1
  selector:
    matchLabels:
      {{`{{`}}- include "{{ $operator }}.selectorLabels" . | nindent 6 {{`}}`}}
{{`{{`}}- end {{`}}`}}

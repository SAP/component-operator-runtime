apiVersion: {{ .groupName }}/{{ .groupVersion }}
kind: {{ .kind }}
metadata:
  name: {{ .operatorName | splitList "." | first | trimSuffix "-cop" }}

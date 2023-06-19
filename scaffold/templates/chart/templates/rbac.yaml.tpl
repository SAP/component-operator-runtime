{{- $operator := (splitList "." .operatorName | first) -}}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - coordination.k8s.io
  resources:
  - leases
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
subjects:
- kind: ServiceAccount
  namespace: {{`{{`}} .Release.Namespace {{`}}`}}
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
  labels:
    {{`{{`}}- include "{{ $operator }}.labels" . | nindent 4 {{`}}`}}
subjects:
- kind: ServiceAccount
  namespace: {{`{{`}} .Release.Namespace {{`}}`}}
  name: {{`{{`}} include "{{ $operator }}.fullname" . {{`}}`}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin

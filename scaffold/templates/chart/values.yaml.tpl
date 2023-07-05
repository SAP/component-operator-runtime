# -- Override full name
fullnameOverride: ""
# -- Override name
nameOverride: ""

# -- Replica count
replicaCount: 1

image:
  # -- Image repository
  repository: {{ regexReplaceAll "^([^/]+/)?([^:]+)(:[^:]+)?$" .image "${1}${2}" }}
  # -- Image tag (defauls to .Chart.AppVersion)
  tag: ""
  # -- Image pull policy
  pullPolicy: IfNotPresent

# -- Image pull secrets
imagePullSecrets: []

# -- Additional pod labels
podLabels: {}

# -- Additional pod annotations
podAnnotations: {}

# -- Node selector
nodeSelector: {}

# -- Affinity settings
affinity: {}

# -- Topology spread constraints (if unspecified, default constraints for hostname and zone will be generated)
topologySpreadConstraints: []

# -- Default topology spread policy for hostname
defaultHostNameSpreadPolicy: ScheduleAnyway

# -- Default topology spread policy for zone
defaultZoneSpreadPolicy: ScheduleAnyway

# -- Tolerations
tolerations: []

# -- Priority class
priorityClassName: ""

# -- Pod security context
podSecurityContext: {}

# -- Container security context
securityContext: {}

resources:
  limits:
    # -- CPU limit
    cpu: 100m
    # -- Memory limit
    memory: 128Mi
  requests:
    # -- CPU request
    cpu: 100m
    # -- Memory request
    memory: 128Mi

{{- if or .validatingWebhookEnabled .mutatingWebhookEnabled }}

service:
  # -- Service type
  type: ClusterIP
  # -- Service port
  port: 443

webhook:
  certManager:
    # -- Whether to use cert-manager to manage webhook tls
    enabled: false
    # -- Issuer group (only relevant if enabled is true; if unset, the default cert-manager group is used)
    issuerGroup: ""
    # -- Issuer kind (only relevant if enabled is true; if unset, the default cert-manager type 'Issuer' is used)
    issuerKind: ""
    # -- Issuer name (only relevant if enabled is true; if unset, a self-signed issuer is used)
    issuerName: ""
{{- end }}

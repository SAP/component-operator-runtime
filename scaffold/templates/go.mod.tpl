module {{ .goModule }}

go {{ .goVersion }}

{{- if contains "/" .version }}

replace github.com/sap/component-operator-runtime => {{ .version }}
{{- end }}

require (
	{{- if contains "/" .version }}
	github.com/sap/component-operator-runtime v0.0.0
	{{- else }}
	github.com/sap/component-operator-runtime {{ .version }}
	{{- end }}
	k8s.io/apiextensions-apiserver {{ .kubernetesVersion }}
	k8s.io/apimachinery {{ .kubernetesVersion }}
	k8s.io/client-go {{ .kubernetesVersion }}
	k8s.io/kube-aggregator {{ .kubernetesVersion }}
	sigs.k8s.io/controller-runtime {{ .controllerRuntimeVersion }}
)

module {{ .goModule }}

go {{ .goVersion }}

require (
	github.com/sap/component-operator-runtime {{ .version }}
	k8s.io/apiextensions-apiserver {{ .kubernetesVersion }}
	k8s.io/apimachinery {{ .kubernetesVersion }}
	k8s.io/client-go {{ .kubernetesVersion }}
	k8s.io/kube-aggregator {{ .kubernetesVersion }}
	sigs.k8s.io/controller-runtime {{ .controllerRuntimeVersion }}
)

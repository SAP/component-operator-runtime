/*
Copyright {{ now.Year }} {{ .owner }}.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"flag"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/operator"

	operatorv1alpha1 "{{ .goModule }}/api/v1alpha1"
)

const Name = "{{ .operatorName }}"

type Options struct {
	Name       string
	FlagPrefix string
}

type Operator struct {
	options Options
}

var defaultOperator operator.Operator = New()

func GetName() string {
	return defaultOperator.GetName()
}

func InitScheme(scheme *runtime.Scheme) {
	defaultOperator.InitScheme(scheme)
}

func InitFlags(flagset *flag.FlagSet) {
	defaultOperator.InitFlags(flagset)
}

func ValidateFlags() error {
	return defaultOperator.ValidateFlags()
}

func GetUncacheableTypes() []client.Object {
	return defaultOperator.GetUncacheableTypes()
}

func Setup(mgr ctrl.Manager, discoveryClient discovery.DiscoveryInterface) error {
	return defaultOperator.Setup(mgr, discoveryClient)
}

func New() *Operator {
	return NewWithOptions(Options{})
}

func NewWithOptions(options Options) *Operator {
	operator := &Operator{options: options}
	if operator.options.Name == "" {
		operator.options.Name = Name
	}
	return operator
}

func (o *Operator) GetName() string {
	return o.options.Name
}

func (o *Operator) InitScheme(scheme *runtime.Scheme) {
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
}

func (o *Operator) InitFlags(flagset *flag.FlagSet) {
	// Add logic to initialize flags (if running in a combined controller you might want to evaluate o.options.FlagPrefix)
}

func (o *Operator) ValidateFlags() error {
	// Add logic to validate flags (if running in a combined controller you might want to evaluate o.options.FlagPrefix)
	return nil
}

func (o *Operator) GetUncacheableTypes() []client.Object {
	// Add types which should bypass informer caching
	return []client.Object{&operatorv1alpha1.{{ .kind }}{}}
}

func (o *Operator) Setup(mgr ctrl.Manager, discoveryClient discovery.DiscoveryInterface) error {
	// Replace this by a real resource generator (e.g. manifests.HelmGenerator, or your own one)
	resourceGenerator, err := manifests.NewDummyGenerator()
	if err != nil {
		return errors.Wrap(err, "error initializing resource generator")
	}
	if err := component.NewReconciler[*operatorv1alpha1.{{ .kind }}](
		o.options.Name,
		mgr.GetClient(),
		discoveryClient,
		mgr.GetEventRecorderFor(o.options.Name),
		mgr.GetScheme(),
		resourceGenerator,
	).SetupWithManager(mgr); err != nil {
		return errors.Wrapf(err, "unable to create controller")
	}
	return nil
}

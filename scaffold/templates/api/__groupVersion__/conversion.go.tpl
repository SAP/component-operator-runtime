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

package {{ .groupVersion }}

// Uncomment the following block if conversion is used, and this api version is the conversion hub;
// see https://book.kubebuilder.io/multiversion-tutorial/conversion.html to learn about the concept of hubs and spokes.
/*
import "sigs.k8s.io/controller-runtime/pkg/conversion"

var _ conversion.Hub = &{{ .kind }}{}

func (c *{{ .kind }}) Hub() {}
*/

// Uncomment the following block if conversion is used, and this api version is a conversion spoke,
// and replace _HUB_API_VERSION_ with the api version of the conversion hub;
// see https://book.kubebuilder.io/multiversion-tutorial/conversion.html to learn about the concept of hubs and spokes.
/*
import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"{{ .goModule }}/api/_HUB_API_VERSION_"
)

var _ conversion.Convertible = &{{ .kind }}{}

func (src *{{ .kind }}) ConvertTo(dstHub conversion.Hub) error {
	dst := dstHub.(*_HUB_API_VERSION_.{{ .kind }})
	dst.ObjectMeta = src.ObjectMeta
	// Add logic here to convert src.Spec into dst.Spec.
	return nil
}

func (dst *{{ .kind }}) ConvertFrom(srcHub conversion.Hub) error {
	src := srcHub.(*_HUB_API_VERSION_.{{ .kind }})
	dst.ObjectMeta = src.ObjectMeta
	// Add logic here to convert src.Spec into dst.Spec.
	return nil
}
*/
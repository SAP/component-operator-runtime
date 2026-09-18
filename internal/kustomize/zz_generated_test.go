/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize_test

import (
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Deep copy functions for Component and ComponentSpec
// Note: manually added, because controller-gen seems to not support test packages/files

func (in *Component) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *Component) DeepCopyInto(out *Component) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *Component) DeepCopy() *Component {
	if in == nil {
		return nil
	}
	out := new(Component)
	in.DeepCopyInto(out)
	return out
}

func (in *ComponentSpec) DeepCopyInto(out *ComponentSpec) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = new(v1.JSON)
		(*in).DeepCopyInto(*out)
	}
}

func (in *ComponentSpec) DeepCopy() *ComponentSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentSpec)
	in.DeepCopyInto(out)
	return out
}

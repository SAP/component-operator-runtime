//go:build !ignore_autogenerated

/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by controller-gen. DO NOT EDIT.

package helm

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ApiVersions) DeepCopyInto(out *ApiVersions) {
	{
		in := &in
		*out = make(ApiVersions, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApiVersions.
func (in ApiVersions) DeepCopy() ApiVersions {
	if in == nil {
		return nil
	}
	out := new(ApiVersions)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Capabilities) DeepCopyInto(out *Capabilities) {
	*out = *in
	out.KubeVersion = in.KubeVersion
	if in.APIVersions != nil {
		in, out := &in.APIVersions, &out.APIVersions
		*out = make(ApiVersions, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Capabilities.
func (in *Capabilities) DeepCopy() *Capabilities {
	if in == nil {
		return nil
	}
	out := new(Capabilities)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartDependency) DeepCopyInto(out *ChartDependency) {
	*out = *in
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ImportValues != nil {
		in, out := &in.ImportValues, &out.ImportValues
		*out = make([]ChartImportValue, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartDependency.
func (in *ChartDependency) DeepCopy() *ChartDependency {
	if in == nil {
		return nil
	}
	out := new(ChartDependency)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartImportValue) DeepCopyInto(out *ChartImportValue) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartImportValue.
func (in *ChartImportValue) DeepCopy() *ChartImportValue {
	if in == nil {
		return nil
	}
	out := new(ChartImportValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartMetadata) DeepCopyInto(out *ChartMetadata) {
	*out = *in
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]ChartDependency, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartMetadata.
func (in *ChartMetadata) DeepCopy() *ChartMetadata {
	if in == nil {
		return nil
	}
	out := new(ChartMetadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeVersion) DeepCopyInto(out *KubeVersion) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeVersion.
func (in *KubeVersion) DeepCopy() *KubeVersion {
	if in == nil {
		return nil
	}
	out := new(KubeVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Release) DeepCopyInto(out *Release) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Release.
func (in *Release) DeepCopy() *Release {
	if in == nil {
		return nil
	}
	out := new(Release)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Template) DeepCopyInto(out *Template) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Template.
func (in *Template) DeepCopy() *Template {
	if in == nil {
		return nil
	}
	out := new(Template)
	in.DeepCopyInto(out)
	return out
}

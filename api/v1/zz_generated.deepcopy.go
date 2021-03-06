//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
This file was generated with "make generate".
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dogu) DeepCopyInto(out *Dogu) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dogu.
func (in *Dogu) DeepCopy() *Dogu {
	if in == nil {
		return nil
	}
	out := new(Dogu)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Dogu) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DoguList) DeepCopyInto(out *DoguList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Dogu, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DoguList.
func (in *DoguList) DeepCopy() *DoguList {
	if in == nil {
		return nil
	}
	out := new(DoguList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DoguList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DoguSpec) DeepCopyInto(out *DoguSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DoguSpec.
func (in *DoguSpec) DeepCopy() *DoguSpec {
	if in == nil {
		return nil
	}
	out := new(DoguSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DoguStatus) DeepCopyInto(out *DoguStatus) {
	*out = *in
	if in.StatusMessages != nil {
		in, out := &in.StatusMessages, &out.StatusMessages
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DoguStatus.
func (in *DoguStatus) DeepCopy() *DoguStatus {
	if in == nil {
		return nil
	}
	out := new(DoguStatus)
	in.DeepCopyInto(out)
	return out
}

// Code generated by mockery v2.13.0-beta.1. DO NOT EDIT.

package mocks

import (
	apply "github.com/cloudogu/k8s-apply-lib/apply"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Applier is an autogenerated mock type for the Applier type
type Applier struct {
	mock.Mock
}

// ApplyWithOwner provides a mock function with given fields: doc, namespace, resource
func (_m *Applier) ApplyWithOwner(doc apply.YamlDocument, namespace string, resource v1.Object) error {
	ret := _m.Called(doc, namespace, resource)

	var r0 error
	if rf, ok := ret.Get(0).(func(apply.YamlDocument, string, v1.Object) error); ok {
		r0 = rf(doc, namespace, resource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type newApplierT interface {
	mock.TestingT
	Cleanup(func())
}

// newApplier creates a new instance of Applier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newApplier(t newApplierT) *Applier {
	mock := &Applier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

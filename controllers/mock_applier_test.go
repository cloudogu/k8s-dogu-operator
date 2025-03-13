// Code generated by mockery v2.53.2. DO NOT EDIT.

package controllers

import (
	apply "github.com/cloudogu/k8s-apply-lib/apply"
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockApplier is an autogenerated mock type for the applier type
type mockApplier struct {
	mock.Mock
}

type mockApplier_Expecter struct {
	mock *mock.Mock
}

func (_m *mockApplier) EXPECT() *mockApplier_Expecter {
	return &mockApplier_Expecter{mock: &_m.Mock}
}

// ApplyWithOwner provides a mock function with given fields: doc, namespace, resource
func (_m *mockApplier) ApplyWithOwner(doc apply.YamlDocument, namespace string, resource v1.Object) error {
	ret := _m.Called(doc, namespace, resource)

	if len(ret) == 0 {
		panic("no return value specified for ApplyWithOwner")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(apply.YamlDocument, string, v1.Object) error); ok {
		r0 = rf(doc, namespace, resource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockApplier_ApplyWithOwner_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyWithOwner'
type mockApplier_ApplyWithOwner_Call struct {
	*mock.Call
}

// ApplyWithOwner is a helper method to define mock.On call
//   - doc apply.YamlDocument
//   - namespace string
//   - resource v1.Object
func (_e *mockApplier_Expecter) ApplyWithOwner(doc interface{}, namespace interface{}, resource interface{}) *mockApplier_ApplyWithOwner_Call {
	return &mockApplier_ApplyWithOwner_Call{Call: _e.mock.On("ApplyWithOwner", doc, namespace, resource)}
}

func (_c *mockApplier_ApplyWithOwner_Call) Run(run func(doc apply.YamlDocument, namespace string, resource v1.Object)) *mockApplier_ApplyWithOwner_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(apply.YamlDocument), args[1].(string), args[2].(v1.Object))
	})
	return _c
}

func (_c *mockApplier_ApplyWithOwner_Call) Return(_a0 error) *mockApplier_ApplyWithOwner_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockApplier_ApplyWithOwner_Call) RunAndReturn(run func(apply.YamlDocument, string, v1.Object) error) *mockApplier_ApplyWithOwner_Call {
	_c.Call.Return(run)
	return _c
}

// newMockApplier creates a new instance of mockApplier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockApplier(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockApplier {
	mock := &mockApplier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

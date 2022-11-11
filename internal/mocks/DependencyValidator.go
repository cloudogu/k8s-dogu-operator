// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"
)

// DependencyValidator is an autogenerated mock type for the DependencyValidator type
type DependencyValidator struct {
	mock.Mock
}

// ValidateDependencies provides a mock function with given fields: ctx, dogu
func (_m *DependencyValidator) ValidateDependencies(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewDependencyValidator interface {
	mock.TestingT
	Cleanup(func())
}

// NewDependencyValidator creates a new instance of DependencyValidator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDependencyValidator(t mockConstructorTestingTNewDependencyValidator) *DependencyValidator {
	mock := &DependencyValidator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	context "context"

	internal "github.com/cloudogu/k8s-dogu-operator/internal"
	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"
)

// ExecPod is an autogenerated mock type for the ExecPod type
type ExecPod struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx
func (_m *ExecPod) Create(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete provides a mock function with given fields: ctx
func (_m *ExecPod) Delete(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Exec provides a mock function with given fields: ctx, cmd
func (_m *ExecPod) Exec(ctx context.Context, cmd internal.ShellCommand) (string, error) {
	ret := _m.Called(ctx, cmd)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, internal.ShellCommand) string); ok {
		r0 = rf(ctx, cmd)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, internal.ShellCommand) error); ok {
		r1 = rf(ctx, cmd)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ObjectKey provides a mock function with given fields:
func (_m *ExecPod) ObjectKey() *types.NamespacedName {
	ret := _m.Called()

	var r0 *types.NamespacedName
	if rf, ok := ret.Get(0).(func() *types.NamespacedName); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NamespacedName)
		}
	}

	return r0
}

// PodName provides a mock function with given fields:
func (_m *ExecPod) PodName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewExecPod interface {
	mock.TestingT
	Cleanup(func())
}

// NewExecPod creates a new instance of ExecPod. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewExecPod(t mockConstructorTestingTNewExecPod) *ExecPod {
	mock := &ExecPod{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

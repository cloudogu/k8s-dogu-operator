// Code generated by mockery v2.53.3. DO NOT EDIT.

package controllers

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

// mockDoguAdditionalMountsValidator is an autogenerated mock type for the doguAdditionalMountsValidator type
type mockDoguAdditionalMountsValidator struct {
	mock.Mock
}

type mockDoguAdditionalMountsValidator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockDoguAdditionalMountsValidator) EXPECT() *mockDoguAdditionalMountsValidator_Expecter {
	return &mockDoguAdditionalMountsValidator_Expecter{mock: &_m.Mock}
}

// ValidateAdditionalMounts provides a mock function with given fields: ctx, doguDescriptor, doguResource
func (_m *mockDoguAdditionalMountsValidator) ValidateAdditionalMounts(ctx context.Context, doguDescriptor *core.Dogu, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguDescriptor, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for ValidateAdditionalMounts")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguDescriptor, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateAdditionalMounts'
type mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call struct {
	*mock.Call
}

// ValidateAdditionalMounts is a helper method to define mock.On call
//   - ctx context.Context
//   - doguDescriptor *core.Dogu
//   - doguResource *v2.Dogu
func (_e *mockDoguAdditionalMountsValidator_Expecter) ValidateAdditionalMounts(ctx interface{}, doguDescriptor interface{}, doguResource interface{}) *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call {
	return &mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call{Call: _e.mock.On("ValidateAdditionalMounts", ctx, doguDescriptor, doguResource)}
}

func (_c *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call) Run(run func(ctx context.Context, doguDescriptor *core.Dogu, doguResource *v2.Dogu)) *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu), args[2].(*v2.Dogu))
	})
	return _c
}

func (_c *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call) Return(_a0 error) *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call) RunAndReturn(run func(context.Context, *core.Dogu, *v2.Dogu) error) *mockDoguAdditionalMountsValidator_ValidateAdditionalMounts_Call {
	_c.Call.Return(run)
	return _c
}

// newMockDoguAdditionalMountsValidator creates a new instance of mockDoguAdditionalMountsValidator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDoguAdditionalMountsValidator(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockDoguAdditionalMountsValidator {
	mock := &mockDoguAdditionalMountsValidator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

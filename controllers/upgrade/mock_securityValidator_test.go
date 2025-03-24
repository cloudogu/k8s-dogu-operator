// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

// mockSecurityValidator is an autogenerated mock type for the securityValidator type
type mockSecurityValidator struct {
	mock.Mock
}

type mockSecurityValidator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSecurityValidator) EXPECT() *mockSecurityValidator_Expecter {
	return &mockSecurityValidator_Expecter{mock: &_m.Mock}
}

// ValidateSecurity provides a mock function with given fields: doguDescriptor, doguResource
func (_m *mockSecurityValidator) ValidateSecurity(doguDescriptor *core.Dogu, doguResource *v2.Dogu) error {
	ret := _m.Called(doguDescriptor, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for ValidateSecurity")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*core.Dogu, *v2.Dogu) error); ok {
		r0 = rf(doguDescriptor, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockSecurityValidator_ValidateSecurity_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateSecurity'
type mockSecurityValidator_ValidateSecurity_Call struct {
	*mock.Call
}

// ValidateSecurity is a helper method to define mock.On call
//   - doguDescriptor *core.Dogu
//   - doguResource *v2.Dogu
func (_e *mockSecurityValidator_Expecter) ValidateSecurity(doguDescriptor interface{}, doguResource interface{}) *mockSecurityValidator_ValidateSecurity_Call {
	return &mockSecurityValidator_ValidateSecurity_Call{Call: _e.mock.On("ValidateSecurity", doguDescriptor, doguResource)}
}

func (_c *mockSecurityValidator_ValidateSecurity_Call) Run(run func(doguDescriptor *core.Dogu, doguResource *v2.Dogu)) *mockSecurityValidator_ValidateSecurity_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*core.Dogu), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockSecurityValidator_ValidateSecurity_Call) Return(_a0 error) *mockSecurityValidator_ValidateSecurity_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockSecurityValidator_ValidateSecurity_Call) RunAndReturn(run func(*core.Dogu, *v2.Dogu) error) *mockSecurityValidator_ValidateSecurity_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSecurityValidator creates a new instance of mockSecurityValidator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSecurityValidator(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSecurityValidator {
	mock := &mockSecurityValidator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

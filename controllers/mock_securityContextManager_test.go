// Code generated by mockery v2.53.2. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// mockSecurityContextManager is an autogenerated mock type for the securityContextManager type
type mockSecurityContextManager struct {
	mock.Mock
}

type mockSecurityContextManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSecurityContextManager) EXPECT() *mockSecurityContextManager_Expecter {
	return &mockSecurityContextManager_Expecter{mock: &_m.Mock}
}

// UpdateDeploymentWithSecurityContext provides a mock function with given fields: ctx, doguResource
func (_m *mockSecurityContextManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for UpdateDeploymentWithSecurityContext")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateDeploymentWithSecurityContext'
type mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call struct {
	*mock.Call
}

// UpdateDeploymentWithSecurityContext is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockSecurityContextManager_Expecter) UpdateDeploymentWithSecurityContext(ctx interface{}, doguResource interface{}) *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call {
	return &mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call{Call: _e.mock.On("UpdateDeploymentWithSecurityContext", ctx, doguResource)}
}

func (_c *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call) Return(_a0 error) *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *mockSecurityContextManager_UpdateDeploymentWithSecurityContext_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSecurityContextManager creates a new instance of mockSecurityContextManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSecurityContextManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSecurityContextManager {
	mock := &mockSecurityContextManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// InstallManager is an autogenerated mock type for the InstallManager type
type InstallManager struct {
	mock.Mock
}

type InstallManager_Expecter struct {
	mock *mock.Mock
}

func (_m *InstallManager) EXPECT() *InstallManager_Expecter {
	return &InstallManager_Expecter{mock: &_m.Mock}
}

// Install provides a mock function with given fields: ctx, doguResource
func (_m *InstallManager) Install(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InstallManager_Install_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Install'
type InstallManager_Install_Call struct {
	*mock.Call
}

// Install is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *InstallManager_Expecter) Install(ctx interface{}, doguResource interface{}) *InstallManager_Install_Call {
	return &InstallManager_Install_Call{Call: _e.mock.On("Install", ctx, doguResource)}
}

func (_c *InstallManager_Install_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *InstallManager_Install_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *InstallManager_Install_Call) Return(_a0 error) *InstallManager_Install_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InstallManager_Install_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *InstallManager_Install_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewInstallManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewInstallManager creates a new instance of InstallManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewInstallManager(t mockConstructorTestingTNewInstallManager) *InstallManager {
	mock := &InstallManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
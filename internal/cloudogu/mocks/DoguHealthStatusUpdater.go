// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/api/apps/v1"
)

// DoguHealthStatusUpdater is an autogenerated mock type for the DoguHealthStatusUpdater type
type DoguHealthStatusUpdater struct {
	mock.Mock
}

type DoguHealthStatusUpdater_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguHealthStatusUpdater) EXPECT() *DoguHealthStatusUpdater_Expecter {
	return &DoguHealthStatusUpdater_Expecter{mock: &_m.Mock}
}

// UpdateHealthConfigMap provides a mock function with given fields: ctx, deployment, doguJson
func (_m *DoguHealthStatusUpdater) UpdateHealthConfigMap(ctx context.Context, deployment *v1.Deployment, doguJson *core.Dogu) error {
	ret := _m.Called(ctx, deployment, doguJson)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Deployment, *core.Dogu) error); ok {
		r0 = rf(ctx, deployment, doguJson)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguHealthStatusUpdater_UpdateHealthConfigMap_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateHealthConfigMap'
type DoguHealthStatusUpdater_UpdateHealthConfigMap_Call struct {
	*mock.Call
}

// UpdateHealthConfigMap is a helper method to define mock.On call
//   - ctx context.Context
//   - deployment *v1.Deployment
//   - doguJson *core.Dogu
func (_e *DoguHealthStatusUpdater_Expecter) UpdateHealthConfigMap(ctx interface{}, deployment interface{}, doguJson interface{}) *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call {
	return &DoguHealthStatusUpdater_UpdateHealthConfigMap_Call{Call: _e.mock.On("UpdateHealthConfigMap", ctx, deployment, doguJson)}
}

func (_c *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call) Run(run func(ctx context.Context, deployment *v1.Deployment, doguJson *core.Dogu)) *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Deployment), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call) Return(_a0 error) *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call) RunAndReturn(run func(context.Context, *v1.Deployment, *core.Dogu) error) *DoguHealthStatusUpdater_UpdateHealthConfigMap_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, doguName, available
func (_m *DoguHealthStatusUpdater) UpdateStatus(ctx context.Context, doguName types.NamespacedName, available bool) error {
	ret := _m.Called(ctx, doguName, available)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName, bool) error); ok {
		r0 = rf(ctx, doguName, available)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguHealthStatusUpdater_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type DoguHealthStatusUpdater_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - doguName types.NamespacedName
//   - available bool
func (_e *DoguHealthStatusUpdater_Expecter) UpdateStatus(ctx interface{}, doguName interface{}, available interface{}) *DoguHealthStatusUpdater_UpdateStatus_Call {
	return &DoguHealthStatusUpdater_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, doguName, available)}
}

func (_c *DoguHealthStatusUpdater_UpdateStatus_Call) Run(run func(ctx context.Context, doguName types.NamespacedName, available bool)) *DoguHealthStatusUpdater_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(types.NamespacedName), args[2].(bool))
	})
	return _c
}

func (_c *DoguHealthStatusUpdater_UpdateStatus_Call) Return(_a0 error) *DoguHealthStatusUpdater_UpdateStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguHealthStatusUpdater_UpdateStatus_Call) RunAndReturn(run func(context.Context, types.NamespacedName, bool) error) *DoguHealthStatusUpdater_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguHealthStatusUpdater interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguHealthStatusUpdater creates a new instance of DoguHealthStatusUpdater. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguHealthStatusUpdater(t mockConstructorTestingTNewDoguHealthStatusUpdater) *DoguHealthStatusUpdater {
	mock := &DoguHealthStatusUpdater{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

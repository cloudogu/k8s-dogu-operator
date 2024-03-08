// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// StopDoguManager is an autogenerated mock type for the StopDoguManager type
type StopDoguManager struct {
	mock.Mock
}

type StopDoguManager_Expecter struct {
	mock *mock.Mock
}

func (_m *StopDoguManager) EXPECT() *StopDoguManager_Expecter {
	return &StopDoguManager_Expecter{mock: &_m.Mock}
}

// StopDogu provides a mock function with given fields: ctx, doguResource
func (_m *StopDoguManager) StopDogu(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopDoguManager_StopDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StopDogu'
type StopDoguManager_StopDogu_Call struct {
	*mock.Call
}

// StopDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
func (_e *StopDoguManager_Expecter) StopDogu(ctx interface{}, doguResource interface{}) *StopDoguManager_StopDogu_Call {
	return &StopDoguManager_StopDogu_Call{Call: _e.mock.On("StopDogu", ctx, doguResource)}
}

func (_c *StopDoguManager_StopDogu_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *StopDoguManager_StopDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *StopDoguManager_StopDogu_Call) Return(_a0 error) *StopDoguManager_StopDogu_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *StopDoguManager_StopDogu_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *StopDoguManager_StopDogu_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewStopDoguManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewStopDoguManager creates a new instance of StopDoguManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewStopDoguManager(t mockConstructorTestingTNewStopDoguManager) *StopDoguManager {
	mock := &StopDoguManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
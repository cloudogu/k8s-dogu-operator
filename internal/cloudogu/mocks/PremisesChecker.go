// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// PremisesChecker is an autogenerated mock type for the PremisesChecker type
type PremisesChecker struct {
	mock.Mock
}

type PremisesChecker_Expecter struct {
	mock *mock.Mock
}

func (_m *PremisesChecker) EXPECT() *PremisesChecker_Expecter {
	return &PremisesChecker_Expecter{mock: &_m.Mock}
}

// Check provides a mock function with given fields: ctx, toDoguResource, fromDogu, toDogu
func (_m *PremisesChecker) Check(ctx context.Context, toDoguResource *v1.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu) error {
	ret := _m.Called(ctx, toDoguResource, fromDogu, toDogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu, *core.Dogu) error); ok {
		r0 = rf(ctx, toDoguResource, fromDogu, toDogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PremisesChecker_Check_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Check'
type PremisesChecker_Check_Call struct {
	*mock.Call
}

// Check is a helper method to define mock.On call
//   - ctx context.Context
//   - toDoguResource *v1.Dogu
//   - fromDogu *core.Dogu
//   - toDogu *core.Dogu
func (_e *PremisesChecker_Expecter) Check(ctx interface{}, toDoguResource interface{}, fromDogu interface{}, toDogu interface{}) *PremisesChecker_Check_Call {
	return &PremisesChecker_Check_Call{Call: _e.mock.On("Check", ctx, toDoguResource, fromDogu, toDogu)}
}

func (_c *PremisesChecker_Check_Call) Run(run func(ctx context.Context, toDoguResource *v1.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu)) *PremisesChecker_Check_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(*core.Dogu), args[3].(*core.Dogu))
	})
	return _c
}

func (_c *PremisesChecker_Check_Call) Return(_a0 error) *PremisesChecker_Check_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *PremisesChecker_Check_Call) RunAndReturn(run func(context.Context, *v1.Dogu, *core.Dogu, *core.Dogu) error) *PremisesChecker_Check_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewPremisesChecker interface {
	mock.TestingT
	Cleanup(func())
}

// NewPremisesChecker creates a new instance of PremisesChecker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPremisesChecker(t mockConstructorTestingTNewPremisesChecker) *PremisesChecker {
	mock := &PremisesChecker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

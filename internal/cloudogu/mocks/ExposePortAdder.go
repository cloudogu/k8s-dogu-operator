// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// ExposePortAdder is an autogenerated mock type for the ExposePortAdder type
type ExposePortAdder struct {
	mock.Mock
}

type ExposePortAdder_Expecter struct {
	mock *mock.Mock
}

func (_m *ExposePortAdder) EXPECT() *ExposePortAdder_Expecter {
	return &ExposePortAdder_Expecter{mock: &_m.Mock}
}

// CreateOrUpdateCesLoadbalancerService provides a mock function with given fields: ctx, doguResource, dogu
func (_m *ExposePortAdder) CreateOrUpdateCesLoadbalancerService(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu) (*corev1.Service, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	var r0 *corev1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) (*corev1.Service, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) *corev1.Service); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateOrUpdateCesLoadbalancerService'
type ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call struct {
	*mock.Call
}

// CreateOrUpdateCesLoadbalancerService is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v1.Dogu
//   - dogu *core.Dogu
func (_e *ExposePortAdder_Expecter) CreateOrUpdateCesLoadbalancerService(ctx interface{}, doguResource interface{}, dogu interface{}) *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	return &ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call{Call: _e.mock.On("CreateOrUpdateCesLoadbalancerService", ctx, doguResource, dogu)}
}

func (_c *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu)) *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) Return(_a0 *corev1.Service, _a1 error) *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) RunAndReturn(run func(context.Context, *v1.Dogu, *core.Dogu) (*corev1.Service, error)) *ExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewExposePortAdder interface {
	mock.TestingT
	Cleanup(func())
}

// NewExposePortAdder creates a new instance of ExposePortAdder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewExposePortAdder(t mockConstructorTestingTNewExposePortAdder) *ExposePortAdder {
	mock := &ExposePortAdder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
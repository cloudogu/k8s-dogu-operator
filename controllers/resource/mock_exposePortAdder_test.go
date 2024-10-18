// Code generated by mockery v2.46.2. DO NOT EDIT.

package resource

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// mockExposePortAdder is an autogenerated mock type for the exposePortAdder type
type mockExposePortAdder struct {
	mock.Mock
}

type mockExposePortAdder_Expecter struct {
	mock *mock.Mock
}

func (_m *mockExposePortAdder) EXPECT() *mockExposePortAdder_Expecter {
	return &mockExposePortAdder_Expecter{mock: &_m.Mock}
}

// CreateOrUpdateCesLoadbalancerService provides a mock function with given fields: ctx, doguResource, dogu
func (_m *mockExposePortAdder) CreateOrUpdateCesLoadbalancerService(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu) (*v1.Service, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for CreateOrUpdateCesLoadbalancerService")
	}

	var r0 *v1.Service
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) (*v1.Service, error)); ok {
		return rf(ctx, doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, *core.Dogu) *v1.Service); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Service)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateOrUpdateCesLoadbalancerService'
type mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call struct {
	*mock.Call
}

// CreateOrUpdateCesLoadbalancerService is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *mockExposePortAdder_Expecter) CreateOrUpdateCesLoadbalancerService(ctx interface{}, doguResource interface{}, dogu interface{}) *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	return &mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call{Call: _e.mock.On("CreateOrUpdateCesLoadbalancerService", ctx, doguResource, dogu)}
}

func (_c *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu, dogu *core.Dogu)) *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) Return(_a0 *v1.Service, _a1 error) *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call) RunAndReturn(run func(context.Context, *v2.Dogu, *core.Dogu) (*v1.Service, error)) *mockExposePortAdder_CreateOrUpdateCesLoadbalancerService_Call {
	_c.Call.Return(run)
	return _c
}

// newMockExposePortAdder creates a new instance of mockExposePortAdder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockExposePortAdder(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockExposePortAdder {
	mock := &mockExposePortAdder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// mockResourceDoguFetcher is an autogenerated mock type for the resourceDoguFetcher type
type mockResourceDoguFetcher struct {
	mock.Mock
}

type mockResourceDoguFetcher_Expecter struct {
	mock *mock.Mock
}

func (_m *mockResourceDoguFetcher) EXPECT() *mockResourceDoguFetcher_Expecter {
	return &mockResourceDoguFetcher_Expecter{mock: &_m.Mock}
}

// FetchWithResource provides a mock function with given fields: ctx, doguResource
func (_m *mockResourceDoguFetcher) FetchWithResource(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, *v2.DevelopmentDoguMap, error) {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for FetchWithResource")
	}

	var r0 *core.Dogu
	var r1 *v2.DevelopmentDoguMap
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) (*core.Dogu, *v2.DevelopmentDoguMap, error)); ok {
		return rf(ctx, doguResource)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) *core.Dogu); ok {
		r0 = rf(ctx, doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu) *v2.DevelopmentDoguMap); ok {
		r1 = rf(ctx, doguResource)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*v2.DevelopmentDoguMap)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, *v2.Dogu) error); ok {
		r2 = rf(ctx, doguResource)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// mockResourceDoguFetcher_FetchWithResource_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FetchWithResource'
type mockResourceDoguFetcher_FetchWithResource_Call struct {
	*mock.Call
}

// FetchWithResource is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockResourceDoguFetcher_Expecter) FetchWithResource(ctx interface{}, doguResource interface{}) *mockResourceDoguFetcher_FetchWithResource_Call {
	return &mockResourceDoguFetcher_FetchWithResource_Call{Call: _e.mock.On("FetchWithResource", ctx, doguResource)}
}

func (_c *mockResourceDoguFetcher_FetchWithResource_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockResourceDoguFetcher_FetchWithResource_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockResourceDoguFetcher_FetchWithResource_Call) Return(_a0 *core.Dogu, _a1 *v2.DevelopmentDoguMap, _a2 error) *mockResourceDoguFetcher_FetchWithResource_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *mockResourceDoguFetcher_FetchWithResource_Call) RunAndReturn(run func(context.Context, *v2.Dogu) (*core.Dogu, *v2.DevelopmentDoguMap, error)) *mockResourceDoguFetcher_FetchWithResource_Call {
	_c.Call.Return(run)
	return _c
}

// newMockResourceDoguFetcher creates a new instance of mockResourceDoguFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockResourceDoguFetcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockResourceDoguFetcher {
	mock := &mockResourceDoguFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
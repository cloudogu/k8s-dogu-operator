// Code generated by mockery v2.42.1. DO NOT EDIT.

package resource

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// MockDoguGetter is an autogenerated mock type for the DoguGetter type
type MockDoguGetter struct {
	mock.Mock
}

type MockDoguGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDoguGetter) EXPECT() *MockDoguGetter_Expecter {
	return &MockDoguGetter_Expecter{mock: &_m.Mock}
}

// GetCurrent provides a mock function with given fields: ctx, simpleDoguName
func (_m *MockDoguGetter) GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error) {
	ret := _m.Called(ctx, simpleDoguName)

	if len(ret) == 0 {
		panic("no return value specified for GetCurrent")
	}

	var r0 *core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*core.Dogu, error)); ok {
		return rf(ctx, simpleDoguName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *core.Dogu); ok {
		r0 = rf(ctx, simpleDoguName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, simpleDoguName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguGetter_GetCurrent_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCurrent'
type MockDoguGetter_GetCurrent_Call struct {
	*mock.Call
}

// GetCurrent is a helper method to define mock.On call
//   - ctx context.Context
//   - simpleDoguName string
func (_e *MockDoguGetter_Expecter) GetCurrent(ctx interface{}, simpleDoguName interface{}) *MockDoguGetter_GetCurrent_Call {
	return &MockDoguGetter_GetCurrent_Call{Call: _e.mock.On("GetCurrent", ctx, simpleDoguName)}
}

func (_c *MockDoguGetter_GetCurrent_Call) Run(run func(ctx context.Context, simpleDoguName string)) *MockDoguGetter_GetCurrent_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDoguGetter_GetCurrent_Call) Return(_a0 *core.Dogu, _a1 error) *MockDoguGetter_GetCurrent_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguGetter_GetCurrent_Call) RunAndReturn(run func(context.Context, string) (*core.Dogu, error)) *MockDoguGetter_GetCurrent_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDoguGetter creates a new instance of MockDoguGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDoguGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDoguGetter {
	mock := &MockDoguGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

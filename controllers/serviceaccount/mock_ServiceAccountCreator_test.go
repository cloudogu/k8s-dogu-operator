// Code generated by mockery v2.53.3. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// MockServiceAccountCreator is an autogenerated mock type for the ServiceAccountCreator type
type MockServiceAccountCreator struct {
	mock.Mock
}

type MockServiceAccountCreator_Expecter struct {
	mock *mock.Mock
}

func (_m *MockServiceAccountCreator) EXPECT() *MockServiceAccountCreator_Expecter {
	return &MockServiceAccountCreator_Expecter{mock: &_m.Mock}
}

// CreateAll provides a mock function with given fields: ctx, dogu
func (_m *MockServiceAccountCreator) CreateAll(ctx context.Context, dogu *core.Dogu) error {
	ret := _m.Called(ctx, dogu)

	if len(ret) == 0 {
		panic("no return value specified for CreateAll")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) error); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockServiceAccountCreator_CreateAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateAll'
type MockServiceAccountCreator_CreateAll_Call struct {
	*mock.Call
}

// CreateAll is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *MockServiceAccountCreator_Expecter) CreateAll(ctx interface{}, dogu interface{}) *MockServiceAccountCreator_CreateAll_Call {
	return &MockServiceAccountCreator_CreateAll_Call{Call: _e.mock.On("CreateAll", ctx, dogu)}
}

func (_c *MockServiceAccountCreator_CreateAll_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *MockServiceAccountCreator_CreateAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *MockServiceAccountCreator_CreateAll_Call) Return(_a0 error) *MockServiceAccountCreator_CreateAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockServiceAccountCreator_CreateAll_Call) RunAndReturn(run func(context.Context, *core.Dogu) error) *MockServiceAccountCreator_CreateAll_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockServiceAccountCreator creates a new instance of MockServiceAccountCreator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockServiceAccountCreator(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockServiceAccountCreator {
	mock := &MockServiceAccountCreator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

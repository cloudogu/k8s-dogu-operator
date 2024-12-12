// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import mock "github.com/stretchr/testify/mock"

// mockRequeuableError is an autogenerated mock type for the requeuableError type
type mockRequeuableError struct {
	mock.Mock
}

type mockRequeuableError_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRequeuableError) EXPECT() *mockRequeuableError_Expecter {
	return &mockRequeuableError_Expecter{mock: &_m.Mock}
}

// Requeue provides a mock function with given fields:
func (_m *mockRequeuableError) Requeue() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Requeue")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// mockRequeuableError_Requeue_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Requeue'
type mockRequeuableError_Requeue_Call struct {
	*mock.Call
}

// Requeue is a helper method to define mock.On call
func (_e *mockRequeuableError_Expecter) Requeue() *mockRequeuableError_Requeue_Call {
	return &mockRequeuableError_Requeue_Call{Call: _e.mock.On("Requeue")}
}

func (_c *mockRequeuableError_Requeue_Call) Run(run func()) *mockRequeuableError_Requeue_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockRequeuableError_Requeue_Call) Return(_a0 bool) *mockRequeuableError_Requeue_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableError_Requeue_Call) RunAndReturn(run func() bool) *mockRequeuableError_Requeue_Call {
	_c.Call.Return(run)
	return _c
}

// newMockRequeuableError creates a new instance of mockRequeuableError. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRequeuableError(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRequeuableError {
	mock := &mockRequeuableError{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

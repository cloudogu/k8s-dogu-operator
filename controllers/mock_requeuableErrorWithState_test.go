// Code generated by mockery v2.53.3. DO NOT EDIT.

package controllers

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// mockRequeuableErrorWithState is an autogenerated mock type for the requeuableErrorWithState type
type mockRequeuableErrorWithState struct {
	mock.Mock
}

type mockRequeuableErrorWithState_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRequeuableErrorWithState) EXPECT() *mockRequeuableErrorWithState_Expecter {
	return &mockRequeuableErrorWithState_Expecter{mock: &_m.Mock}
}

// GetRequeueTime provides a mock function with no fields
func (_m *mockRequeuableErrorWithState) GetRequeueTime() time.Duration {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRequeueTime")
	}

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// mockRequeuableErrorWithState_GetRequeueTime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRequeueTime'
type mockRequeuableErrorWithState_GetRequeueTime_Call struct {
	*mock.Call
}

// GetRequeueTime is a helper method to define mock.On call
func (_e *mockRequeuableErrorWithState_Expecter) GetRequeueTime() *mockRequeuableErrorWithState_GetRequeueTime_Call {
	return &mockRequeuableErrorWithState_GetRequeueTime_Call{Call: _e.mock.On("GetRequeueTime")}
}

func (_c *mockRequeuableErrorWithState_GetRequeueTime_Call) Run(run func()) *mockRequeuableErrorWithState_GetRequeueTime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockRequeuableErrorWithState_GetRequeueTime_Call) Return(_a0 time.Duration) *mockRequeuableErrorWithState_GetRequeueTime_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableErrorWithState_GetRequeueTime_Call) RunAndReturn(run func() time.Duration) *mockRequeuableErrorWithState_GetRequeueTime_Call {
	_c.Call.Return(run)
	return _c
}

// GetState provides a mock function with no fields
func (_m *mockRequeuableErrorWithState) GetState() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetState")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// mockRequeuableErrorWithState_GetState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetState'
type mockRequeuableErrorWithState_GetState_Call struct {
	*mock.Call
}

// GetState is a helper method to define mock.On call
func (_e *mockRequeuableErrorWithState_Expecter) GetState() *mockRequeuableErrorWithState_GetState_Call {
	return &mockRequeuableErrorWithState_GetState_Call{Call: _e.mock.On("GetState")}
}

func (_c *mockRequeuableErrorWithState_GetState_Call) Run(run func()) *mockRequeuableErrorWithState_GetState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockRequeuableErrorWithState_GetState_Call) Return(_a0 string) *mockRequeuableErrorWithState_GetState_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableErrorWithState_GetState_Call) RunAndReturn(run func() string) *mockRequeuableErrorWithState_GetState_Call {
	_c.Call.Return(run)
	return _c
}

// Requeue provides a mock function with no fields
func (_m *mockRequeuableErrorWithState) Requeue() bool {
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

// mockRequeuableErrorWithState_Requeue_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Requeue'
type mockRequeuableErrorWithState_Requeue_Call struct {
	*mock.Call
}

// Requeue is a helper method to define mock.On call
func (_e *mockRequeuableErrorWithState_Expecter) Requeue() *mockRequeuableErrorWithState_Requeue_Call {
	return &mockRequeuableErrorWithState_Requeue_Call{Call: _e.mock.On("Requeue")}
}

func (_c *mockRequeuableErrorWithState_Requeue_Call) Run(run func()) *mockRequeuableErrorWithState_Requeue_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockRequeuableErrorWithState_Requeue_Call) Return(_a0 bool) *mockRequeuableErrorWithState_Requeue_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableErrorWithState_Requeue_Call) RunAndReturn(run func() bool) *mockRequeuableErrorWithState_Requeue_Call {
	_c.Call.Return(run)
	return _c
}

// newMockRequeuableErrorWithState creates a new instance of mockRequeuableErrorWithState. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRequeuableErrorWithState(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRequeuableErrorWithState {
	mock := &mockRequeuableErrorWithState{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

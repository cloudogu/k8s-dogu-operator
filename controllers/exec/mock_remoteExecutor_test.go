// Code generated by mockery v2.53.2. DO NOT EDIT.

package exec

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	remotecommand "k8s.io/client-go/tools/remotecommand"
)

// mockRemoteExecutor is an autogenerated mock type for the remoteExecutor type
type mockRemoteExecutor struct {
	mock.Mock
}

type mockRemoteExecutor_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRemoteExecutor) EXPECT() *mockRemoteExecutor_Expecter {
	return &mockRemoteExecutor_Expecter{mock: &_m.Mock}
}

// Stream provides a mock function with given fields: options
func (_m *mockRemoteExecutor) Stream(options remotecommand.StreamOptions) error {
	ret := _m.Called(options)

	if len(ret) == 0 {
		panic("no return value specified for Stream")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(remotecommand.StreamOptions) error); ok {
		r0 = rf(options)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRemoteExecutor_Stream_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stream'
type mockRemoteExecutor_Stream_Call struct {
	*mock.Call
}

// Stream is a helper method to define mock.On call
//   - options remotecommand.StreamOptions
func (_e *mockRemoteExecutor_Expecter) Stream(options interface{}) *mockRemoteExecutor_Stream_Call {
	return &mockRemoteExecutor_Stream_Call{Call: _e.mock.On("Stream", options)}
}

func (_c *mockRemoteExecutor_Stream_Call) Run(run func(options remotecommand.StreamOptions)) *mockRemoteExecutor_Stream_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(remotecommand.StreamOptions))
	})
	return _c
}

func (_c *mockRemoteExecutor_Stream_Call) Return(_a0 error) *mockRemoteExecutor_Stream_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRemoteExecutor_Stream_Call) RunAndReturn(run func(remotecommand.StreamOptions) error) *mockRemoteExecutor_Stream_Call {
	_c.Call.Return(run)
	return _c
}

// StreamWithContext provides a mock function with given fields: ctx, options
func (_m *mockRemoteExecutor) StreamWithContext(ctx context.Context, options remotecommand.StreamOptions) error {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for StreamWithContext")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, remotecommand.StreamOptions) error); ok {
		r0 = rf(ctx, options)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRemoteExecutor_StreamWithContext_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'StreamWithContext'
type mockRemoteExecutor_StreamWithContext_Call struct {
	*mock.Call
}

// StreamWithContext is a helper method to define mock.On call
//   - ctx context.Context
//   - options remotecommand.StreamOptions
func (_e *mockRemoteExecutor_Expecter) StreamWithContext(ctx interface{}, options interface{}) *mockRemoteExecutor_StreamWithContext_Call {
	return &mockRemoteExecutor_StreamWithContext_Call{Call: _e.mock.On("StreamWithContext", ctx, options)}
}

func (_c *mockRemoteExecutor_StreamWithContext_Call) Run(run func(ctx context.Context, options remotecommand.StreamOptions)) *mockRemoteExecutor_StreamWithContext_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(remotecommand.StreamOptions))
	})
	return _c
}

func (_c *mockRemoteExecutor_StreamWithContext_Call) Return(_a0 error) *mockRemoteExecutor_StreamWithContext_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRemoteExecutor_StreamWithContext_Call) RunAndReturn(run func(context.Context, remotecommand.StreamOptions) error) *mockRemoteExecutor_StreamWithContext_Call {
	_c.Call.Return(run)
	return _c
}

// newMockRemoteExecutor creates a new instance of mockRemoteExecutor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRemoteExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRemoteExecutor {
	mock := &mockRemoteExecutor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

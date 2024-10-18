// Code generated by mockery v2.46.2. DO NOT EDIT.

package exec

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// MockShellCommand is an autogenerated mock type for the ShellCommand type
type MockShellCommand struct {
	mock.Mock
}

type MockShellCommand_Expecter struct {
	mock *mock.Mock
}

func (_m *MockShellCommand) EXPECT() *MockShellCommand_Expecter {
	return &MockShellCommand_Expecter{mock: &_m.Mock}
}

// CommandWithArgs provides a mock function with given fields:
func (_m *MockShellCommand) CommandWithArgs() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CommandWithArgs")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// MockShellCommand_CommandWithArgs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CommandWithArgs'
type MockShellCommand_CommandWithArgs_Call struct {
	*mock.Call
}

// CommandWithArgs is a helper method to define mock.On call
func (_e *MockShellCommand_Expecter) CommandWithArgs() *MockShellCommand_CommandWithArgs_Call {
	return &MockShellCommand_CommandWithArgs_Call{Call: _e.mock.On("CommandWithArgs")}
}

func (_c *MockShellCommand_CommandWithArgs_Call) Run(run func()) *MockShellCommand_CommandWithArgs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockShellCommand_CommandWithArgs_Call) Return(_a0 []string) *MockShellCommand_CommandWithArgs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockShellCommand_CommandWithArgs_Call) RunAndReturn(run func() []string) *MockShellCommand_CommandWithArgs_Call {
	_c.Call.Return(run)
	return _c
}

// Stdin provides a mock function with given fields:
func (_m *MockShellCommand) Stdin() io.Reader {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Stdin")
	}

	var r0 io.Reader
	if rf, ok := ret.Get(0).(func() io.Reader); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.Reader)
		}
	}

	return r0
}

// MockShellCommand_Stdin_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stdin'
type MockShellCommand_Stdin_Call struct {
	*mock.Call
}

// Stdin is a helper method to define mock.On call
func (_e *MockShellCommand_Expecter) Stdin() *MockShellCommand_Stdin_Call {
	return &MockShellCommand_Stdin_Call{Call: _e.mock.On("Stdin")}
}

func (_c *MockShellCommand_Stdin_Call) Run(run func()) *MockShellCommand_Stdin_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockShellCommand_Stdin_Call) Return(_a0 io.Reader) *MockShellCommand_Stdin_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockShellCommand_Stdin_Call) RunAndReturn(run func() io.Reader) *MockShellCommand_Stdin_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockShellCommand creates a new instance of MockShellCommand. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockShellCommand(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockShellCommand {
	mock := &MockShellCommand{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
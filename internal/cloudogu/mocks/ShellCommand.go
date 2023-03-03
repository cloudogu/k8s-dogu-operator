// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// ShellCommand is an autogenerated mock type for the ShellCommand type
type ShellCommand struct {
	mock.Mock
}

type ShellCommand_Expecter struct {
	mock *mock.Mock
}

func (_m *ShellCommand) EXPECT() *ShellCommand_Expecter {
	return &ShellCommand_Expecter{mock: &_m.Mock}
}

// CommandWithArgs provides a mock function with given fields:
func (_m *ShellCommand) CommandWithArgs() []string {
	ret := _m.Called()

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

// ShellCommand_CommandWithArgs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CommandWithArgs'
type ShellCommand_CommandWithArgs_Call struct {
	*mock.Call
}

// CommandWithArgs is a helper method to define mock.On call
func (_e *ShellCommand_Expecter) CommandWithArgs() *ShellCommand_CommandWithArgs_Call {
	return &ShellCommand_CommandWithArgs_Call{Call: _e.mock.On("CommandWithArgs")}
}

func (_c *ShellCommand_CommandWithArgs_Call) Run(run func()) *ShellCommand_CommandWithArgs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ShellCommand_CommandWithArgs_Call) Return(_a0 []string) *ShellCommand_CommandWithArgs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ShellCommand_CommandWithArgs_Call) RunAndReturn(run func() []string) *ShellCommand_CommandWithArgs_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewShellCommand interface {
	mock.TestingT
	Cleanup(func())
}

// NewShellCommand creates a new instance of ShellCommand. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewShellCommand(t mockConstructorTestingTNewShellCommand) *ShellCommand {
	mock := &ShellCommand{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

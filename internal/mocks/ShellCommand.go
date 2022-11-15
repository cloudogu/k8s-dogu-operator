// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// ShellCommand is an autogenerated mock type for the ShellCommand type
type ShellCommand struct {
	mock.Mock
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

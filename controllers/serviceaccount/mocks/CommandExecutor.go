// Code generated by mockery v2.10.2. DO NOT EDIT.

package mocks

import (
	bytes "bytes"
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"
)

// CommandExecutor is an autogenerated mock type for the CommandExecutor type
type CommandExecutor struct {
	mock.Mock
}

// ExecCommand provides a mock function with given fields: ctx, targetDogu, namespace, command, params
func (_m *CommandExecutor) ExecCommand(ctx context.Context, targetDogu string, namespace string, command *core.ExposedCommand, params []string) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, targetDogu, namespace, command, params)

	var r0 *bytes.Buffer
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *core.ExposedCommand, []string) *bytes.Buffer); ok {
		r0 = rf(ctx, targetDogu, namespace, command, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, *core.ExposedCommand, []string) error); ok {
		r1 = rf(ctx, targetDogu, namespace, command, params)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

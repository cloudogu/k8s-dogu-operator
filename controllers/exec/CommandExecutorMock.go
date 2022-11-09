// Code generated by mockery v2.14.0. DO NOT EDIT.

package exec

import (
	bytes "bytes"
	context "context"

	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// CommandExecutor is an autogenerated mock type for the CommandExecutor type
type CommandExecutor struct {
	mock.Mock
}

// ExecCommandForDogu provides a mock function with given fields: ctx, resource, command, expectedStatus
func (_m *CommandExecutor) ExecCommandForDogu(ctx context.Context, resource *v1.Dogu, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, resource, command, expectedStatus)

	var r0 *bytes.Buffer
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *ShellCommand, PodStatus) *bytes.Buffer); ok {
		r0 = rf(ctx, resource, command, expectedStatus)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *ShellCommand, PodStatus) error); ok {
		r1 = rf(ctx, resource, command, expectedStatus)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExecCommandForPod provides a mock function with given fields: ctx, pod, command, expectedStatus
func (_m *CommandExecutor) ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, pod, command, expectedStatus)

	var r0 *bytes.Buffer
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, *ShellCommand, PodStatus) *bytes.Buffer); ok {
		r0 = rf(ctx, pod, command, expectedStatus)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *corev1.Pod, *ShellCommand, PodStatus) error); ok {
		r1 = rf(ctx, pod, command, expectedStatus)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewCommandExecutor interface {
	mock.TestingT
	Cleanup(func())
}

// NewCommandExecutor creates a new instance of CommandExecutor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewCommandExecutorMock(t mockConstructorTestingTNewCommandExecutor) *CommandExecutor {
	mock := &CommandExecutor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
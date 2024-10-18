// Code generated by mockery v2.46.2. DO NOT EDIT.

package upgrade

import (
	bytes "bytes"
	context "context"

	exec "github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// mockCommandExecutor is an autogenerated mock type for the commandExecutor type
type mockCommandExecutor struct {
	mock.Mock
}

type mockCommandExecutor_Expecter struct {
	mock *mock.Mock
}

func (_m *mockCommandExecutor) EXPECT() *mockCommandExecutor_Expecter {
	return &mockCommandExecutor_Expecter{mock: &_m.Mock}
}

// ExecCommandForDogu provides a mock function with given fields: ctx, resource, command, expected
func (_m *mockCommandExecutor) ExecCommandForDogu(ctx context.Context, resource *v2.Dogu, command exec.ShellCommand, expected exec.PodStatusForExec) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, resource, command, expected)

	if len(ret) == 0 {
		panic("no return value specified for ExecCommandForDogu")
	}

	var r0 *bytes.Buffer
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, exec.ShellCommand, exec.PodStatusForExec) (*bytes.Buffer, error)); ok {
		return rf(ctx, resource, command, expected)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, exec.ShellCommand, exec.PodStatusForExec) *bytes.Buffer); ok {
		r0 = rf(ctx, resource, command, expected)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, exec.ShellCommand, exec.PodStatusForExec) error); ok {
		r1 = rf(ctx, resource, command, expected)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCommandExecutor_ExecCommandForDogu_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExecCommandForDogu'
type mockCommandExecutor_ExecCommandForDogu_Call struct {
	*mock.Call
}

// ExecCommandForDogu is a helper method to define mock.On call
//   - ctx context.Context
//   - resource *v2.Dogu
//   - command exec.ShellCommand
//   - expected exec.PodStatusForExec
func (_e *mockCommandExecutor_Expecter) ExecCommandForDogu(ctx interface{}, resource interface{}, command interface{}, expected interface{}) *mockCommandExecutor_ExecCommandForDogu_Call {
	return &mockCommandExecutor_ExecCommandForDogu_Call{Call: _e.mock.On("ExecCommandForDogu", ctx, resource, command, expected)}
}

func (_c *mockCommandExecutor_ExecCommandForDogu_Call) Run(run func(ctx context.Context, resource *v2.Dogu, command exec.ShellCommand, expected exec.PodStatusForExec)) *mockCommandExecutor_ExecCommandForDogu_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(exec.ShellCommand), args[3].(exec.PodStatusForExec))
	})
	return _c
}

func (_c *mockCommandExecutor_ExecCommandForDogu_Call) Return(_a0 *bytes.Buffer, _a1 error) *mockCommandExecutor_ExecCommandForDogu_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCommandExecutor_ExecCommandForDogu_Call) RunAndReturn(run func(context.Context, *v2.Dogu, exec.ShellCommand, exec.PodStatusForExec) (*bytes.Buffer, error)) *mockCommandExecutor_ExecCommandForDogu_Call {
	_c.Call.Return(run)
	return _c
}

// ExecCommandForPod provides a mock function with given fields: ctx, pod, command, expected
func (_m *mockCommandExecutor) ExecCommandForPod(ctx context.Context, pod *v1.Pod, command exec.ShellCommand, expected exec.PodStatusForExec) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, pod, command, expected)

	if len(ret) == 0 {
		panic("no return value specified for ExecCommandForPod")
	}

	var r0 *bytes.Buffer
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Pod, exec.ShellCommand, exec.PodStatusForExec) (*bytes.Buffer, error)); ok {
		return rf(ctx, pod, command, expected)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Pod, exec.ShellCommand, exec.PodStatusForExec) *bytes.Buffer); ok {
		r0 = rf(ctx, pod, command, expected)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Pod, exec.ShellCommand, exec.PodStatusForExec) error); ok {
		r1 = rf(ctx, pod, command, expected)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCommandExecutor_ExecCommandForPod_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExecCommandForPod'
type mockCommandExecutor_ExecCommandForPod_Call struct {
	*mock.Call
}

// ExecCommandForPod is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *v1.Pod
//   - command exec.ShellCommand
//   - expected exec.PodStatusForExec
func (_e *mockCommandExecutor_Expecter) ExecCommandForPod(ctx interface{}, pod interface{}, command interface{}, expected interface{}) *mockCommandExecutor_ExecCommandForPod_Call {
	return &mockCommandExecutor_ExecCommandForPod_Call{Call: _e.mock.On("ExecCommandForPod", ctx, pod, command, expected)}
}

func (_c *mockCommandExecutor_ExecCommandForPod_Call) Run(run func(ctx context.Context, pod *v1.Pod, command exec.ShellCommand, expected exec.PodStatusForExec)) *mockCommandExecutor_ExecCommandForPod_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Pod), args[2].(exec.ShellCommand), args[3].(exec.PodStatusForExec))
	})
	return _c
}

func (_c *mockCommandExecutor_ExecCommandForPod_Call) Return(_a0 *bytes.Buffer, _a1 error) *mockCommandExecutor_ExecCommandForPod_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCommandExecutor_ExecCommandForPod_Call) RunAndReturn(run func(context.Context, *v1.Pod, exec.ShellCommand, exec.PodStatusForExec) (*bytes.Buffer, error)) *mockCommandExecutor_ExecCommandForPod_Call {
	_c.Call.Return(run)
	return _c
}

// newMockCommandExecutor creates a new instance of mockCommandExecutor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockCommandExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockCommandExecutor {
	mock := &mockCommandExecutor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

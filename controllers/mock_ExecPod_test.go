// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	bytes "bytes"
	context "context"

	exec "github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"
)

// MockExecPod is an autogenerated mock type for the ExecPod type
type MockExecPod struct {
	mock.Mock
}

type MockExecPod_Expecter struct {
	mock *mock.Mock
}

func (_m *MockExecPod) EXPECT() *MockExecPod_Expecter {
	return &MockExecPod_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx
func (_m *MockExecPod) Create(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockExecPod_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type MockExecPod_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockExecPod_Expecter) Create(ctx interface{}) *MockExecPod_Create_Call {
	return &MockExecPod_Create_Call{Call: _e.mock.On("Create", ctx)}
}

func (_c *MockExecPod_Create_Call) Run(run func(ctx context.Context)) *MockExecPod_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockExecPod_Create_Call) Return(_a0 error) *MockExecPod_Create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecPod_Create_Call) RunAndReturn(run func(context.Context) error) *MockExecPod_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx
func (_m *MockExecPod) Delete(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockExecPod_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type MockExecPod_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockExecPod_Expecter) Delete(ctx interface{}) *MockExecPod_Delete_Call {
	return &MockExecPod_Delete_Call{Call: _e.mock.On("Delete", ctx)}
}

func (_c *MockExecPod_Delete_Call) Run(run func(ctx context.Context)) *MockExecPod_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockExecPod_Delete_Call) Return(_a0 error) *MockExecPod_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecPod_Delete_Call) RunAndReturn(run func(context.Context) error) *MockExecPod_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// Exec provides a mock function with given fields: ctx, cmd
func (_m *MockExecPod) Exec(ctx context.Context, cmd exec.ShellCommand) (*bytes.Buffer, error) {
	ret := _m.Called(ctx, cmd)

	if len(ret) == 0 {
		panic("no return value specified for Exec")
	}

	var r0 *bytes.Buffer
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, exec.ShellCommand) (*bytes.Buffer, error)); ok {
		return rf(ctx, cmd)
	}
	if rf, ok := ret.Get(0).(func(context.Context, exec.ShellCommand) *bytes.Buffer); ok {
		r0 = rf(ctx, cmd)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bytes.Buffer)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, exec.ShellCommand) error); ok {
		r1 = rf(ctx, cmd)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockExecPod_Exec_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exec'
type MockExecPod_Exec_Call struct {
	*mock.Call
}

// Exec is a helper method to define mock.On call
//   - ctx context.Context
//   - cmd exec.ShellCommand
func (_e *MockExecPod_Expecter) Exec(ctx interface{}, cmd interface{}) *MockExecPod_Exec_Call {
	return &MockExecPod_Exec_Call{Call: _e.mock.On("Exec", ctx, cmd)}
}

func (_c *MockExecPod_Exec_Call) Run(run func(ctx context.Context, cmd exec.ShellCommand)) *MockExecPod_Exec_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(exec.ShellCommand))
	})
	return _c
}

func (_c *MockExecPod_Exec_Call) Return(out *bytes.Buffer, err error) *MockExecPod_Exec_Call {
	_c.Call.Return(out, err)
	return _c
}

func (_c *MockExecPod_Exec_Call) RunAndReturn(run func(context.Context, exec.ShellCommand) (*bytes.Buffer, error)) *MockExecPod_Exec_Call {
	_c.Call.Return(run)
	return _c
}

// ObjectKey provides a mock function with given fields:
func (_m *MockExecPod) ObjectKey() *types.NamespacedName {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ObjectKey")
	}

	var r0 *types.NamespacedName
	if rf, ok := ret.Get(0).(func() *types.NamespacedName); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NamespacedName)
		}
	}

	return r0
}

// MockExecPod_ObjectKey_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ObjectKey'
type MockExecPod_ObjectKey_Call struct {
	*mock.Call
}

// ObjectKey is a helper method to define mock.On call
func (_e *MockExecPod_Expecter) ObjectKey() *MockExecPod_ObjectKey_Call {
	return &MockExecPod_ObjectKey_Call{Call: _e.mock.On("ObjectKey")}
}

func (_c *MockExecPod_ObjectKey_Call) Run(run func()) *MockExecPod_ObjectKey_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockExecPod_ObjectKey_Call) Return(_a0 *types.NamespacedName) *MockExecPod_ObjectKey_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecPod_ObjectKey_Call) RunAndReturn(run func() *types.NamespacedName) *MockExecPod_ObjectKey_Call {
	_c.Call.Return(run)
	return _c
}

// PodName provides a mock function with given fields:
func (_m *MockExecPod) PodName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for PodName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockExecPod_PodName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PodName'
type MockExecPod_PodName_Call struct {
	*mock.Call
}

// PodName is a helper method to define mock.On call
func (_e *MockExecPod_Expecter) PodName() *MockExecPod_PodName_Call {
	return &MockExecPod_PodName_Call{Call: _e.mock.On("PodName")}
}

func (_c *MockExecPod_PodName_Call) Run(run func()) *MockExecPod_PodName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockExecPod_PodName_Call) Return(_a0 string) *MockExecPod_PodName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockExecPod_PodName_Call) RunAndReturn(run func() string) *MockExecPod_PodName_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockExecPod creates a new instance of MockExecPod. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockExecPod(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockExecPod {
	mock := &MockExecPod{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

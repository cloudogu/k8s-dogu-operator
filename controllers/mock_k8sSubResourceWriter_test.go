// Code generated by mockery v2.20.0. DO NOT EDIT.

package controllers

import (
	context "context"

	client "sigs.k8s.io/controller-runtime/pkg/client"

	mock "github.com/stretchr/testify/mock"
)

// mockK8sSubResourceWriter is an autogenerated mock type for the k8sSubResourceWriter type
type mockK8sSubResourceWriter struct {
	mock.Mock
}

type mockK8sSubResourceWriter_Expecter struct {
	mock *mock.Mock
}

func (_m *mockK8sSubResourceWriter) EXPECT() *mockK8sSubResourceWriter_Expecter {
	return &mockK8sSubResourceWriter_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx, obj, subResource, opts
func (_m *mockK8sSubResourceWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj, subResource)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error); ok {
		r0 = rf(ctx, obj, subResource, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockK8sSubResourceWriter_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockK8sSubResourceWriter_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - subResource client.Object
//   - opts ...client.SubResourceCreateOption
func (_e *mockK8sSubResourceWriter_Expecter) Create(ctx interface{}, obj interface{}, subResource interface{}, opts ...interface{}) *mockK8sSubResourceWriter_Create_Call {
	return &mockK8sSubResourceWriter_Create_Call{Call: _e.mock.On("Create",
		append([]interface{}{ctx, obj, subResource}, opts...)...)}
}

func (_c *mockK8sSubResourceWriter_Create_Call) Run(run func(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption)) *mockK8sSubResourceWriter_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.SubResourceCreateOption, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(client.SubResourceCreateOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), args[2].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *mockK8sSubResourceWriter_Create_Call) Return(_a0 error) *mockK8sSubResourceWriter_Create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockK8sSubResourceWriter_Create_Call) RunAndReturn(run func(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error) *mockK8sSubResourceWriter_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, obj, patch, opts
func (_m *mockK8sSubResourceWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj, patch)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, client.Patch, ...client.SubResourcePatchOption) error); ok {
		r0 = rf(ctx, obj, patch, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockK8sSubResourceWriter_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockK8sSubResourceWriter_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - patch client.Patch
//   - opts ...client.SubResourcePatchOption
func (_e *mockK8sSubResourceWriter_Expecter) Patch(ctx interface{}, obj interface{}, patch interface{}, opts ...interface{}) *mockK8sSubResourceWriter_Patch_Call {
	return &mockK8sSubResourceWriter_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, obj, patch}, opts...)...)}
}

func (_c *mockK8sSubResourceWriter_Patch_Call) Run(run func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption)) *mockK8sSubResourceWriter_Patch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.SubResourcePatchOption, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(client.SubResourcePatchOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), args[2].(client.Patch), variadicArgs...)
	})
	return _c
}

func (_c *mockK8sSubResourceWriter_Patch_Call) Return(_a0 error) *mockK8sSubResourceWriter_Patch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockK8sSubResourceWriter_Patch_Call) RunAndReturn(run func(context.Context, client.Object, client.Patch, ...client.SubResourcePatchOption) error) *mockK8sSubResourceWriter_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, obj, opts
func (_m *mockK8sSubResourceWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, ...client.SubResourceUpdateOption) error); ok {
		r0 = rf(ctx, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockK8sSubResourceWriter_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockK8sSubResourceWriter_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - opts ...client.SubResourceUpdateOption
func (_e *mockK8sSubResourceWriter_Expecter) Update(ctx interface{}, obj interface{}, opts ...interface{}) *mockK8sSubResourceWriter_Update_Call {
	return &mockK8sSubResourceWriter_Update_Call{Call: _e.mock.On("Update",
		append([]interface{}{ctx, obj}, opts...)...)}
}

func (_c *mockK8sSubResourceWriter_Update_Call) Run(run func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption)) *mockK8sSubResourceWriter_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.SubResourceUpdateOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.SubResourceUpdateOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *mockK8sSubResourceWriter_Update_Call) Return(_a0 error) *mockK8sSubResourceWriter_Update_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockK8sSubResourceWriter_Update_Call) RunAndReturn(run func(context.Context, client.Object, ...client.SubResourceUpdateOption) error) *mockK8sSubResourceWriter_Update_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockK8sSubResourceWriter interface {
	mock.TestingT
	Cleanup(func())
}

// newMockK8sSubResourceWriter creates a new instance of mockK8sSubResourceWriter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockK8sSubResourceWriter(t mockConstructorTestingTnewMockK8sSubResourceWriter) *mockK8sSubResourceWriter {
	mock := &mockK8sSubResourceWriter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

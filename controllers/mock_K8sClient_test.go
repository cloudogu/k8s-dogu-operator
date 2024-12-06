// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	client "sigs.k8s.io/controller-runtime/pkg/client"

	meta "k8s.io/apimachinery/pkg/api/meta"

	mock "github.com/stretchr/testify/mock"

	runtime "k8s.io/apimachinery/pkg/runtime"

	schema "k8s.io/apimachinery/pkg/runtime/schema"

	types "k8s.io/apimachinery/pkg/types"
)

// MockK8sClient is an autogenerated mock type for the K8sClient type
type MockK8sClient struct {
	mock.Mock
}

type MockK8sClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockK8sClient) EXPECT() *MockK8sClient_Expecter {
	return &MockK8sClient_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx, obj, opts
func (_m *MockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, ...client.CreateOption) error); ok {
		r0 = rf(ctx, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type MockK8sClient_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - opts ...client.CreateOption
func (_e *MockK8sClient_Expecter) Create(ctx interface{}, obj interface{}, opts ...interface{}) *MockK8sClient_Create_Call {
	return &MockK8sClient_Create_Call{Call: _e.mock.On("Create",
		append([]interface{}{ctx, obj}, opts...)...)}
}

func (_c *MockK8sClient_Create_Call) Run(run func(ctx context.Context, obj client.Object, opts ...client.CreateOption)) *MockK8sClient_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.CreateOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.CreateOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_Create_Call) Return(_a0 error) *MockK8sClient_Create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Create_Call) RunAndReturn(run func(context.Context, client.Object, ...client.CreateOption) error) *MockK8sClient_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, obj, opts
func (_m *MockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, ...client.DeleteOption) error); ok {
		r0 = rf(ctx, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type MockK8sClient_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - opts ...client.DeleteOption
func (_e *MockK8sClient_Expecter) Delete(ctx interface{}, obj interface{}, opts ...interface{}) *MockK8sClient_Delete_Call {
	return &MockK8sClient_Delete_Call{Call: _e.mock.On("Delete",
		append([]interface{}{ctx, obj}, opts...)...)}
}

func (_c *MockK8sClient_Delete_Call) Run(run func(ctx context.Context, obj client.Object, opts ...client.DeleteOption)) *MockK8sClient_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.DeleteOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.DeleteOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_Delete_Call) Return(_a0 error) *MockK8sClient_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Delete_Call) RunAndReturn(run func(context.Context, client.Object, ...client.DeleteOption) error) *MockK8sClient_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteAllOf provides a mock function with given fields: ctx, obj, opts
func (_m *MockK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for DeleteAllOf")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, ...client.DeleteAllOfOption) error); ok {
		r0 = rf(ctx, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_DeleteAllOf_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteAllOf'
type MockK8sClient_DeleteAllOf_Call struct {
	*mock.Call
}

// DeleteAllOf is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - opts ...client.DeleteAllOfOption
func (_e *MockK8sClient_Expecter) DeleteAllOf(ctx interface{}, obj interface{}, opts ...interface{}) *MockK8sClient_DeleteAllOf_Call {
	return &MockK8sClient_DeleteAllOf_Call{Call: _e.mock.On("DeleteAllOf",
		append([]interface{}{ctx, obj}, opts...)...)}
}

func (_c *MockK8sClient_DeleteAllOf_Call) Run(run func(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption)) *MockK8sClient_DeleteAllOf_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.DeleteAllOfOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.DeleteAllOfOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_DeleteAllOf_Call) Return(_a0 error) *MockK8sClient_DeleteAllOf_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_DeleteAllOf_Call) RunAndReturn(run func(context.Context, client.Object, ...client.DeleteAllOfOption) error) *MockK8sClient_DeleteAllOf_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, key, obj, opts
func (_m *MockK8sClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, key, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error); ok {
		r0 = rf(ctx, key, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockK8sClient_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - key types.NamespacedName
//   - obj client.Object
//   - opts ...client.GetOption
func (_e *MockK8sClient_Expecter) Get(ctx interface{}, key interface{}, obj interface{}, opts ...interface{}) *MockK8sClient_Get_Call {
	return &MockK8sClient_Get_Call{Call: _e.mock.On("Get",
		append([]interface{}{ctx, key, obj}, opts...)...)}
}

func (_c *MockK8sClient_Get_Call) Run(run func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption)) *MockK8sClient_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.GetOption, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(client.GetOption)
			}
		}
		run(args[0].(context.Context), args[1].(types.NamespacedName), args[2].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_Get_Call) Return(_a0 error) *MockK8sClient_Get_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Get_Call) RunAndReturn(run func(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error) *MockK8sClient_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GroupVersionKindFor provides a mock function with given fields: obj
func (_m *MockK8sClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	ret := _m.Called(obj)

	if len(ret) == 0 {
		panic("no return value specified for GroupVersionKindFor")
	}

	var r0 schema.GroupVersionKind
	var r1 error
	if rf, ok := ret.Get(0).(func(runtime.Object) (schema.GroupVersionKind, error)); ok {
		return rf(obj)
	}
	if rf, ok := ret.Get(0).(func(runtime.Object) schema.GroupVersionKind); ok {
		r0 = rf(obj)
	} else {
		r0 = ret.Get(0).(schema.GroupVersionKind)
	}

	if rf, ok := ret.Get(1).(func(runtime.Object) error); ok {
		r1 = rf(obj)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockK8sClient_GroupVersionKindFor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GroupVersionKindFor'
type MockK8sClient_GroupVersionKindFor_Call struct {
	*mock.Call
}

// GroupVersionKindFor is a helper method to define mock.On call
//   - obj runtime.Object
func (_e *MockK8sClient_Expecter) GroupVersionKindFor(obj interface{}) *MockK8sClient_GroupVersionKindFor_Call {
	return &MockK8sClient_GroupVersionKindFor_Call{Call: _e.mock.On("GroupVersionKindFor", obj)}
}

func (_c *MockK8sClient_GroupVersionKindFor_Call) Run(run func(obj runtime.Object)) *MockK8sClient_GroupVersionKindFor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(runtime.Object))
	})
	return _c
}

func (_c *MockK8sClient_GroupVersionKindFor_Call) Return(_a0 schema.GroupVersionKind, _a1 error) *MockK8sClient_GroupVersionKindFor_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockK8sClient_GroupVersionKindFor_Call) RunAndReturn(run func(runtime.Object) (schema.GroupVersionKind, error)) *MockK8sClient_GroupVersionKindFor_Call {
	_c.Call.Return(run)
	return _c
}

// IsObjectNamespaced provides a mock function with given fields: obj
func (_m *MockK8sClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	ret := _m.Called(obj)

	if len(ret) == 0 {
		panic("no return value specified for IsObjectNamespaced")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(runtime.Object) (bool, error)); ok {
		return rf(obj)
	}
	if rf, ok := ret.Get(0).(func(runtime.Object) bool); ok {
		r0 = rf(obj)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(runtime.Object) error); ok {
		r1 = rf(obj)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockK8sClient_IsObjectNamespaced_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsObjectNamespaced'
type MockK8sClient_IsObjectNamespaced_Call struct {
	*mock.Call
}

// IsObjectNamespaced is a helper method to define mock.On call
//   - obj runtime.Object
func (_e *MockK8sClient_Expecter) IsObjectNamespaced(obj interface{}) *MockK8sClient_IsObjectNamespaced_Call {
	return &MockK8sClient_IsObjectNamespaced_Call{Call: _e.mock.On("IsObjectNamespaced", obj)}
}

func (_c *MockK8sClient_IsObjectNamespaced_Call) Run(run func(obj runtime.Object)) *MockK8sClient_IsObjectNamespaced_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(runtime.Object))
	})
	return _c
}

func (_c *MockK8sClient_IsObjectNamespaced_Call) Return(_a0 bool, _a1 error) *MockK8sClient_IsObjectNamespaced_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockK8sClient_IsObjectNamespaced_Call) RunAndReturn(run func(runtime.Object) (bool, error)) *MockK8sClient_IsObjectNamespaced_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, list, opts
func (_m *MockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, list)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.ObjectList, ...client.ListOption) error); ok {
		r0 = rf(ctx, list, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type MockK8sClient_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - list client.ObjectList
//   - opts ...client.ListOption
func (_e *MockK8sClient_Expecter) List(ctx interface{}, list interface{}, opts ...interface{}) *MockK8sClient_List_Call {
	return &MockK8sClient_List_Call{Call: _e.mock.On("List",
		append([]interface{}{ctx, list}, opts...)...)}
}

func (_c *MockK8sClient_List_Call) Run(run func(ctx context.Context, list client.ObjectList, opts ...client.ListOption)) *MockK8sClient_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.ListOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.ListOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.ObjectList), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_List_Call) Return(_a0 error) *MockK8sClient_List_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_List_Call) RunAndReturn(run func(context.Context, client.ObjectList, ...client.ListOption) error) *MockK8sClient_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, obj, patch, opts
func (_m *MockK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj, patch)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Patch")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, client.Patch, ...client.PatchOption) error); ok {
		r0 = rf(ctx, obj, patch, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type MockK8sClient_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - patch client.Patch
//   - opts ...client.PatchOption
func (_e *MockK8sClient_Expecter) Patch(ctx interface{}, obj interface{}, patch interface{}, opts ...interface{}) *MockK8sClient_Patch_Call {
	return &MockK8sClient_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, obj, patch}, opts...)...)}
}

func (_c *MockK8sClient_Patch_Call) Run(run func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption)) *MockK8sClient_Patch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.PatchOption, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(client.PatchOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), args[2].(client.Patch), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_Patch_Call) Return(_a0 error) *MockK8sClient_Patch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Patch_Call) RunAndReturn(run func(context.Context, client.Object, client.Patch, ...client.PatchOption) error) *MockK8sClient_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// RESTMapper provides a mock function with given fields:
func (_m *MockK8sClient) RESTMapper() meta.RESTMapper {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for RESTMapper")
	}

	var r0 meta.RESTMapper
	if rf, ok := ret.Get(0).(func() meta.RESTMapper); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(meta.RESTMapper)
		}
	}

	return r0
}

// MockK8sClient_RESTMapper_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RESTMapper'
type MockK8sClient_RESTMapper_Call struct {
	*mock.Call
}

// RESTMapper is a helper method to define mock.On call
func (_e *MockK8sClient_Expecter) RESTMapper() *MockK8sClient_RESTMapper_Call {
	return &MockK8sClient_RESTMapper_Call{Call: _e.mock.On("RESTMapper")}
}

func (_c *MockK8sClient_RESTMapper_Call) Run(run func()) *MockK8sClient_RESTMapper_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockK8sClient_RESTMapper_Call) Return(_a0 meta.RESTMapper) *MockK8sClient_RESTMapper_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_RESTMapper_Call) RunAndReturn(run func() meta.RESTMapper) *MockK8sClient_RESTMapper_Call {
	_c.Call.Return(run)
	return _c
}

// Scheme provides a mock function with given fields:
func (_m *MockK8sClient) Scheme() *runtime.Scheme {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Scheme")
	}

	var r0 *runtime.Scheme
	if rf, ok := ret.Get(0).(func() *runtime.Scheme); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*runtime.Scheme)
		}
	}

	return r0
}

// MockK8sClient_Scheme_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Scheme'
type MockK8sClient_Scheme_Call struct {
	*mock.Call
}

// Scheme is a helper method to define mock.On call
func (_e *MockK8sClient_Expecter) Scheme() *MockK8sClient_Scheme_Call {
	return &MockK8sClient_Scheme_Call{Call: _e.mock.On("Scheme")}
}

func (_c *MockK8sClient_Scheme_Call) Run(run func()) *MockK8sClient_Scheme_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockK8sClient_Scheme_Call) Return(_a0 *runtime.Scheme) *MockK8sClient_Scheme_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Scheme_Call) RunAndReturn(run func() *runtime.Scheme) *MockK8sClient_Scheme_Call {
	_c.Call.Return(run)
	return _c
}

// Status provides a mock function with given fields:
func (_m *MockK8sClient) Status() client.SubResourceWriter {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Status")
	}

	var r0 client.SubResourceWriter
	if rf, ok := ret.Get(0).(func() client.SubResourceWriter); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(client.SubResourceWriter)
		}
	}

	return r0
}

// MockK8sClient_Status_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Status'
type MockK8sClient_Status_Call struct {
	*mock.Call
}

// Status is a helper method to define mock.On call
func (_e *MockK8sClient_Expecter) Status() *MockK8sClient_Status_Call {
	return &MockK8sClient_Status_Call{Call: _e.mock.On("Status")}
}

func (_c *MockK8sClient_Status_Call) Run(run func()) *MockK8sClient_Status_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockK8sClient_Status_Call) Return(_a0 client.SubResourceWriter) *MockK8sClient_Status_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Status_Call) RunAndReturn(run func() client.SubResourceWriter) *MockK8sClient_Status_Call {
	_c.Call.Return(run)
	return _c
}

// SubResource provides a mock function with given fields: subResource
func (_m *MockK8sClient) SubResource(subResource string) client.SubResourceClient {
	ret := _m.Called(subResource)

	if len(ret) == 0 {
		panic("no return value specified for SubResource")
	}

	var r0 client.SubResourceClient
	if rf, ok := ret.Get(0).(func(string) client.SubResourceClient); ok {
		r0 = rf(subResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(client.SubResourceClient)
		}
	}

	return r0
}

// MockK8sClient_SubResource_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubResource'
type MockK8sClient_SubResource_Call struct {
	*mock.Call
}

// SubResource is a helper method to define mock.On call
//   - subResource string
func (_e *MockK8sClient_Expecter) SubResource(subResource interface{}) *MockK8sClient_SubResource_Call {
	return &MockK8sClient_SubResource_Call{Call: _e.mock.On("SubResource", subResource)}
}

func (_c *MockK8sClient_SubResource_Call) Run(run func(subResource string)) *MockK8sClient_SubResource_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockK8sClient_SubResource_Call) Return(_a0 client.SubResourceClient) *MockK8sClient_SubResource_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_SubResource_Call) RunAndReturn(run func(string) client.SubResourceClient) *MockK8sClient_SubResource_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, obj, opts
func (_m *MockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, obj)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object, ...client.UpdateOption) error); ok {
		r0 = rf(ctx, obj, opts...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockK8sClient_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type MockK8sClient_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - obj client.Object
//   - opts ...client.UpdateOption
func (_e *MockK8sClient_Expecter) Update(ctx interface{}, obj interface{}, opts ...interface{}) *MockK8sClient_Update_Call {
	return &MockK8sClient_Update_Call{Call: _e.mock.On("Update",
		append([]interface{}{ctx, obj}, opts...)...)}
}

func (_c *MockK8sClient_Update_Call) Run(run func(ctx context.Context, obj client.Object, opts ...client.UpdateOption)) *MockK8sClient_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]client.UpdateOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(client.UpdateOption)
			}
		}
		run(args[0].(context.Context), args[1].(client.Object), variadicArgs...)
	})
	return _c
}

func (_c *MockK8sClient_Update_Call) Return(_a0 error) *MockK8sClient_Update_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockK8sClient_Update_Call) RunAndReturn(run func(context.Context, client.Object, ...client.UpdateOption) error) *MockK8sClient_Update_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockK8sClient creates a new instance of MockK8sClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockK8sClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockK8sClient {
	mock := &MockK8sClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

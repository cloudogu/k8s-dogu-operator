// Code generated by mockery v2.46.2. DO NOT EDIT.

package ecoSystem

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"

	watch "k8s.io/apimachinery/pkg/watch"
)

// MockDoguInterface is an autogenerated mock type for the DoguInterface type
type MockDoguInterface struct {
	mock.Mock
}

type MockDoguInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDoguInterface) EXPECT() *MockDoguInterface_Expecter {
	return &MockDoguInterface_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx, dogu, opts
func (_m *MockDoguInterface) Create(ctx context.Context, dogu *v2.Dogu, opts v1.CreateOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.CreateOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.CreateOptions) *v2.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, v1.CreateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type MockDoguInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - opts v1.CreateOptions
func (_e *MockDoguInterface_Expecter) Create(ctx interface{}, dogu interface{}, opts interface{}) *MockDoguInterface_Create_Call {
	return &MockDoguInterface_Create_Call{Call: _e.mock.On("Create", ctx, dogu, opts)}
}

func (_c *MockDoguInterface_Create_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, opts v1.CreateOptions)) *MockDoguInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(v1.CreateOptions))
	})
	return _c
}

func (_c *MockDoguInterface_Create_Call) Return(_a0 *v2.Dogu, _a1 error) *MockDoguInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_Create_Call) RunAndReturn(run func(context.Context, *v2.Dogu, v1.CreateOptions) (*v2.Dogu, error)) *MockDoguInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *MockDoguInterface) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type MockDoguInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts v1.DeleteOptions
func (_e *MockDoguInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *MockDoguInterface_Delete_Call {
	return &MockDoguInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *MockDoguInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts v1.DeleteOptions)) *MockDoguInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(v1.DeleteOptions))
	})
	return _c
}

func (_c *MockDoguInterface_Delete_Call) Return(_a0 error) *MockDoguInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguInterface_Delete_Call) RunAndReturn(run func(context.Context, string, v1.DeleteOptions) error) *MockDoguInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *MockDoguInterface) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	if len(ret) == 0 {
		panic("no return value specified for DeleteCollection")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, v1.DeleteOptions, v1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDoguInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type MockDoguInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts v1.DeleteOptions
//   - listOpts v1.ListOptions
func (_e *MockDoguInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *MockDoguInterface_DeleteCollection_Call {
	return &MockDoguInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *MockDoguInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions)) *MockDoguInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(v1.DeleteOptions), args[2].(v1.ListOptions))
	})
	return _c
}

func (_c *MockDoguInterface_DeleteCollection_Call) Return(_a0 error) *MockDoguInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDoguInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, v1.DeleteOptions, v1.ListOptions) error) *MockDoguInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *MockDoguInterface) Get(ctx context.Context, name string, opts v1.GetOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.GetOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, v1.GetOptions) *v2.Dogu); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, v1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockDoguInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts v1.GetOptions
func (_e *MockDoguInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *MockDoguInterface_Get_Call {
	return &MockDoguInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *MockDoguInterface_Get_Call) Run(run func(ctx context.Context, name string, opts v1.GetOptions)) *MockDoguInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(v1.GetOptions))
	})
	return _c
}

func (_c *MockDoguInterface_Get_Call) Return(_a0 *v2.Dogu, _a1 error) *MockDoguInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_Get_Call) RunAndReturn(run func(context.Context, string, v1.GetOptions) (*v2.Dogu, error)) *MockDoguInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *MockDoguInterface) List(ctx context.Context, opts v1.ListOptions) (*v2.DoguList, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *v2.DoguList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) (*v2.DoguList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) *v2.DoguList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.DoguList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, v1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type MockDoguInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts v1.ListOptions
func (_e *MockDoguInterface_Expecter) List(ctx interface{}, opts interface{}) *MockDoguInterface_List_Call {
	return &MockDoguInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *MockDoguInterface_List_Call) Run(run func(ctx context.Context, opts v1.ListOptions)) *MockDoguInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(v1.ListOptions))
	})
	return _c
}

func (_c *MockDoguInterface_List_Call) Return(_a0 *v2.DoguList, _a1 error) *MockDoguInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_List_Call) RunAndReturn(run func(context.Context, v1.ListOptions) (*v2.DoguList, error)) *MockDoguInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *MockDoguInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (*v2.Dogu, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Patch")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) (*v2.Dogu, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) *v2.Dogu); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type MockDoguInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts v1.PatchOptions
//   - subresources ...string
func (_e *MockDoguInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *MockDoguInterface_Patch_Call {
	return &MockDoguInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *MockDoguInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string)) *MockDoguInterface_Patch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-5)
		for i, a := range args[5:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(args[0].(context.Context), args[1].(string), args[2].(types.PatchType), args[3].([]byte), args[4].(v1.PatchOptions), variadicArgs...)
	})
	return _c
}

func (_c *MockDoguInterface_Patch_Call) Return(result *v2.Dogu, err error) *MockDoguInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *MockDoguInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, v1.PatchOptions, ...string) (*v2.Dogu, error)) *MockDoguInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, dogu, opts
func (_m *MockDoguInterface) Update(ctx context.Context, dogu *v2.Dogu, opts v1.UpdateOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.UpdateOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.UpdateOptions) *v2.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, v1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type MockDoguInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - opts v1.UpdateOptions
func (_e *MockDoguInterface_Expecter) Update(ctx interface{}, dogu interface{}, opts interface{}) *MockDoguInterface_Update_Call {
	return &MockDoguInterface_Update_Call{Call: _e.mock.On("Update", ctx, dogu, opts)}
}

func (_c *MockDoguInterface_Update_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, opts v1.UpdateOptions)) *MockDoguInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(v1.UpdateOptions))
	})
	return _c
}

func (_c *MockDoguInterface_Update_Call) Return(_a0 *v2.Dogu, _a1 error) *MockDoguInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_Update_Call) RunAndReturn(run func(context.Context, *v2.Dogu, v1.UpdateOptions) (*v2.Dogu, error)) *MockDoguInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateSpecWithRetry provides a mock function with given fields: ctx, dogu, modifySpecFn, opts
func (_m *MockDoguInterface) UpdateSpecWithRetry(ctx context.Context, dogu *v2.Dogu, modifySpecFn func(v2.DoguSpec) v2.DoguSpec, opts v1.UpdateOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, dogu, modifySpecFn, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateSpecWithRetry")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, func(v2.DoguSpec) v2.DoguSpec, v1.UpdateOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, dogu, modifySpecFn, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, func(v2.DoguSpec) v2.DoguSpec, v1.UpdateOptions) *v2.Dogu); ok {
		r0 = rf(ctx, dogu, modifySpecFn, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, func(v2.DoguSpec) v2.DoguSpec, v1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, modifySpecFn, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_UpdateSpecWithRetry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateSpecWithRetry'
type MockDoguInterface_UpdateSpecWithRetry_Call struct {
	*mock.Call
}

// UpdateSpecWithRetry is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - modifySpecFn func(v2.DoguSpec) v2.DoguSpec
//   - opts v1.UpdateOptions
func (_e *MockDoguInterface_Expecter) UpdateSpecWithRetry(ctx interface{}, dogu interface{}, modifySpecFn interface{}, opts interface{}) *MockDoguInterface_UpdateSpecWithRetry_Call {
	return &MockDoguInterface_UpdateSpecWithRetry_Call{Call: _e.mock.On("UpdateSpecWithRetry", ctx, dogu, modifySpecFn, opts)}
}

func (_c *MockDoguInterface_UpdateSpecWithRetry_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, modifySpecFn func(v2.DoguSpec) v2.DoguSpec, opts v1.UpdateOptions)) *MockDoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(func(v2.DoguSpec) v2.DoguSpec), args[3].(v1.UpdateOptions))
	})
	return _c
}

func (_c *MockDoguInterface_UpdateSpecWithRetry_Call) Return(result *v2.Dogu, err error) *MockDoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *MockDoguInterface_UpdateSpecWithRetry_Call) RunAndReturn(run func(context.Context, *v2.Dogu, func(v2.DoguSpec) v2.DoguSpec, v1.UpdateOptions) (*v2.Dogu, error)) *MockDoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, dogu, opts
func (_m *MockDoguInterface) UpdateStatus(ctx context.Context, dogu *v2.Dogu, opts v1.UpdateOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatus")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.UpdateOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, v1.UpdateOptions) *v2.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, v1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type MockDoguInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - opts v1.UpdateOptions
func (_e *MockDoguInterface_Expecter) UpdateStatus(ctx interface{}, dogu interface{}, opts interface{}) *MockDoguInterface_UpdateStatus_Call {
	return &MockDoguInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, dogu, opts)}
}

func (_c *MockDoguInterface_UpdateStatus_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, opts v1.UpdateOptions)) *MockDoguInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(v1.UpdateOptions))
	})
	return _c
}

func (_c *MockDoguInterface_UpdateStatus_Call) Return(_a0 *v2.Dogu, _a1 error) *MockDoguInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *v2.Dogu, v1.UpdateOptions) (*v2.Dogu, error)) *MockDoguInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusWithRetry provides a mock function with given fields: ctx, dogu, modifyStatusFn, opts
func (_m *MockDoguInterface) UpdateStatusWithRetry(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) (*v2.Dogu, error) {
	ret := _m.Called(ctx, dogu, modifyStatusFn, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatusWithRetry")
	}

	var r0 *v2.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, func(v2.DoguStatus) v2.DoguStatus, v1.UpdateOptions) (*v2.Dogu, error)); ok {
		return rf(ctx, dogu, modifyStatusFn, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu, func(v2.DoguStatus) v2.DoguStatus, v1.UpdateOptions) *v2.Dogu); ok {
		r0 = rf(ctx, dogu, modifyStatusFn, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v2.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v2.Dogu, func(v2.DoguStatus) v2.DoguStatus, v1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, modifyStatusFn, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_UpdateStatusWithRetry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusWithRetry'
type MockDoguInterface_UpdateStatusWithRetry_Call struct {
	*mock.Call
}

// UpdateStatusWithRetry is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v2.Dogu
//   - modifyStatusFn func(v2.DoguStatus) v2.DoguStatus
//   - opts v1.UpdateOptions
func (_e *MockDoguInterface_Expecter) UpdateStatusWithRetry(ctx interface{}, dogu interface{}, modifyStatusFn interface{}, opts interface{}) *MockDoguInterface_UpdateStatusWithRetry_Call {
	return &MockDoguInterface_UpdateStatusWithRetry_Call{Call: _e.mock.On("UpdateStatusWithRetry", ctx, dogu, modifyStatusFn, opts)}
}

func (_c *MockDoguInterface_UpdateStatusWithRetry_Call) Run(run func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions)) *MockDoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu), args[2].(func(v2.DoguStatus) v2.DoguStatus), args[3].(v1.UpdateOptions))
	})
	return _c
}

func (_c *MockDoguInterface_UpdateStatusWithRetry_Call) Return(result *v2.Dogu, err error) *MockDoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *MockDoguInterface_UpdateStatusWithRetry_Call) RunAndReturn(run func(context.Context, *v2.Dogu, func(v2.DoguStatus) v2.DoguStatus, v1.UpdateOptions) (*v2.Dogu, error)) *MockDoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *MockDoguInterface) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for Watch")
	}

	var r0 watch.Interface
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) (watch.Interface, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, v1.ListOptions) watch.Interface); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(watch.Interface)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, v1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDoguInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type MockDoguInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts v1.ListOptions
func (_e *MockDoguInterface_Expecter) Watch(ctx interface{}, opts interface{}) *MockDoguInterface_Watch_Call {
	return &MockDoguInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *MockDoguInterface_Watch_Call) Run(run func(ctx context.Context, opts v1.ListOptions)) *MockDoguInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(v1.ListOptions))
	})
	return _c
}

func (_c *MockDoguInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *MockDoguInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDoguInterface_Watch_Call) RunAndReturn(run func(context.Context, v1.ListOptions) (watch.Interface, error)) *MockDoguInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDoguInterface creates a new instance of MockDoguInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDoguInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDoguInterface {
	mock := &MockDoguInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

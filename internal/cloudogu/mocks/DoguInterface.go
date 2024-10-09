// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	types "k8s.io/apimachinery/pkg/types"

	v1 "github.com/cloudogu/k8s-dogu-operator/v2/api/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// DoguInterface is an autogenerated mock type for the DoguInterface type
type DoguInterface struct {
	mock.Mock
}

type DoguInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguInterface) EXPECT() *DoguInterface_Expecter {
	return &DoguInterface_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx, dogu, opts
func (_m *DoguInterface) Create(ctx context.Context, dogu *v1.Dogu, opts metav1.CreateOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.CreateOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.CreateOptions) *v1.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type DoguInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
//   - opts metav1.CreateOptions
func (_e *DoguInterface_Expecter) Create(ctx interface{}, dogu interface{}, opts interface{}) *DoguInterface_Create_Call {
	return &DoguInterface_Create_Call{Call: _e.mock.On("Create", ctx, dogu, opts)}
}

func (_c *DoguInterface_Create_Call) Run(run func(ctx context.Context, dogu *v1.Dogu, opts metav1.CreateOptions)) *DoguInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *DoguInterface_Create_Call) Return(_a0 *v1.Dogu, _a1 error) *DoguInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_Create_Call) RunAndReturn(run func(context.Context, *v1.Dogu, metav1.CreateOptions) (*v1.Dogu, error)) *DoguInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *DoguInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type DoguInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *DoguInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *DoguInterface_Delete_Call {
	return &DoguInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *DoguInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *DoguInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *DoguInterface_Delete_Call) Return(_a0 error) *DoguInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *DoguInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *DoguInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type DoguInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *DoguInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *DoguInterface_DeleteCollection_Call {
	return &DoguInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *DoguInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *DoguInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *DoguInterface_DeleteCollection_Call) Return(_a0 error) *DoguInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *DoguInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *DoguInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, name, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *v1.Dogu); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type DoguInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *DoguInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *DoguInterface_Get_Call {
	return &DoguInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *DoguInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *DoguInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *DoguInterface_Get_Call) Return(_a0 *v1.Dogu, _a1 error) *DoguInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*v1.Dogu, error)) *DoguInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *DoguInterface) List(ctx context.Context, opts metav1.ListOptions) (*v1.DoguList, error) {
	ret := _m.Called(ctx, opts)

	var r0 *v1.DoguList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*v1.DoguList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *v1.DoguList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DoguList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type DoguInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *DoguInterface_Expecter) List(ctx interface{}, opts interface{}) *DoguInterface_List_Call {
	return &DoguInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *DoguInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *DoguInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *DoguInterface_List_Call) Return(_a0 *v1.DoguList, _a1 error) *DoguInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*v1.DoguList, error)) *DoguInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *DoguInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.Dogu, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Dogu, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *v1.Dogu); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type DoguInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *DoguInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *DoguInterface_Patch_Call {
	return &DoguInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *DoguInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *DoguInterface_Patch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-5)
		for i, a := range args[5:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(args[0].(context.Context), args[1].(string), args[2].(types.PatchType), args[3].([]byte), args[4].(metav1.PatchOptions), variadicArgs...)
	})
	return _c
}

func (_c *DoguInterface_Patch_Call) Return(result *v1.Dogu, err error) *DoguInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *DoguInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Dogu, error)) *DoguInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, dogu, opts
func (_m *DoguInterface) Update(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) *v1.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type DoguInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
//   - opts metav1.UpdateOptions
func (_e *DoguInterface_Expecter) Update(ctx interface{}, dogu interface{}, opts interface{}) *DoguInterface_Update_Call {
	return &DoguInterface_Update_Call{Call: _e.mock.On("Update", ctx, dogu, opts)}
}

func (_c *DoguInterface_Update_Call) Run(run func(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions)) *DoguInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *DoguInterface_Update_Call) Return(_a0 *v1.Dogu, _a1 error) *DoguInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_Update_Call) RunAndReturn(run func(context.Context, *v1.Dogu, metav1.UpdateOptions) (*v1.Dogu, error)) *DoguInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateSpecWithRetry provides a mock function with given fields: ctx, dogu, modifySpecFn, opts
func (_m *DoguInterface) UpdateSpecWithRetry(ctx context.Context, dogu *v1.Dogu, modifySpecFn func(v1.DoguSpec) v1.DoguSpec, opts metav1.UpdateOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, dogu, modifySpecFn, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, func(v1.DoguSpec) v1.DoguSpec, metav1.UpdateOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, dogu, modifySpecFn, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, func(v1.DoguSpec) v1.DoguSpec, metav1.UpdateOptions) *v1.Dogu); ok {
		r0 = rf(ctx, dogu, modifySpecFn, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, func(v1.DoguSpec) v1.DoguSpec, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, modifySpecFn, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_UpdateSpecWithRetry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateSpecWithRetry'
type DoguInterface_UpdateSpecWithRetry_Call struct {
	*mock.Call
}

// UpdateSpecWithRetry is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
//   - modifySpecFn func(v1.DoguSpec) v1.DoguSpec
//   - opts metav1.UpdateOptions
func (_e *DoguInterface_Expecter) UpdateSpecWithRetry(ctx interface{}, dogu interface{}, modifySpecFn interface{}, opts interface{}) *DoguInterface_UpdateSpecWithRetry_Call {
	return &DoguInterface_UpdateSpecWithRetry_Call{Call: _e.mock.On("UpdateSpecWithRetry", ctx, dogu, modifySpecFn, opts)}
}

func (_c *DoguInterface_UpdateSpecWithRetry_Call) Run(run func(ctx context.Context, dogu *v1.Dogu, modifySpecFn func(v1.DoguSpec) v1.DoguSpec, opts metav1.UpdateOptions)) *DoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(func(v1.DoguSpec) v1.DoguSpec), args[3].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *DoguInterface_UpdateSpecWithRetry_Call) Return(result *v1.Dogu, err error) *DoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *DoguInterface_UpdateSpecWithRetry_Call) RunAndReturn(run func(context.Context, *v1.Dogu, func(v1.DoguSpec) v1.DoguSpec, metav1.UpdateOptions) (*v1.Dogu, error)) *DoguInterface_UpdateSpecWithRetry_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, dogu, opts
func (_m *DoguInterface) UpdateStatus(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, dogu, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, dogu, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) *v1.Dogu); ok {
		r0 = rf(ctx, dogu, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type DoguInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
//   - opts metav1.UpdateOptions
func (_e *DoguInterface_Expecter) UpdateStatus(ctx interface{}, dogu interface{}, opts interface{}) *DoguInterface_UpdateStatus_Call {
	return &DoguInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, dogu, opts)}
}

func (_c *DoguInterface_UpdateStatus_Call) Run(run func(ctx context.Context, dogu *v1.Dogu, opts metav1.UpdateOptions)) *DoguInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *DoguInterface_UpdateStatus_Call) Return(_a0 *v1.Dogu, _a1 error) *DoguInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *v1.Dogu, metav1.UpdateOptions) (*v1.Dogu, error)) *DoguInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusWithRetry provides a mock function with given fields: ctx, dogu, modifyStatusFn, opts
func (_m *DoguInterface) UpdateStatusWithRetry(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1.UpdateOptions) (*v1.Dogu, error) {
	ret := _m.Called(ctx, dogu, modifyStatusFn, opts)

	var r0 *v1.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, func(v1.DoguStatus) v1.DoguStatus, metav1.UpdateOptions) (*v1.Dogu, error)); ok {
		return rf(ctx, dogu, modifyStatusFn, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, func(v1.DoguStatus) v1.DoguStatus, metav1.UpdateOptions) *v1.Dogu); ok {
		r0 = rf(ctx, dogu, modifyStatusFn, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, func(v1.DoguStatus) v1.DoguStatus, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, dogu, modifyStatusFn, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_UpdateStatusWithRetry_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusWithRetry'
type DoguInterface_UpdateStatusWithRetry_Call struct {
	*mock.Call
}

// UpdateStatusWithRetry is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *v1.Dogu
//   - modifyStatusFn func(v1.DoguStatus) v1.DoguStatus
//   - opts metav1.UpdateOptions
func (_e *DoguInterface_Expecter) UpdateStatusWithRetry(ctx interface{}, dogu interface{}, modifyStatusFn interface{}, opts interface{}) *DoguInterface_UpdateStatusWithRetry_Call {
	return &DoguInterface_UpdateStatusWithRetry_Call{Call: _e.mock.On("UpdateStatusWithRetry", ctx, dogu, modifyStatusFn, opts)}
}

func (_c *DoguInterface_UpdateStatusWithRetry_Call) Run(run func(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1.UpdateOptions)) *DoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu), args[2].(func(v1.DoguStatus) v1.DoguStatus), args[3].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *DoguInterface_UpdateStatusWithRetry_Call) Return(result *v1.Dogu, err error) *DoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *DoguInterface_UpdateStatusWithRetry_Call) RunAndReturn(run func(context.Context, *v1.Dogu, func(v1.DoguStatus) v1.DoguStatus, metav1.UpdateOptions) (*v1.Dogu, error)) *DoguInterface_UpdateStatusWithRetry_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *DoguInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	var r0 watch.Interface
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (watch.Interface, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) watch.Interface); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(watch.Interface)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type DoguInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *DoguInterface_Expecter) Watch(ctx interface{}, opts interface{}) *DoguInterface_Watch_Call {
	return &DoguInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *DoguInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *DoguInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *DoguInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *DoguInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *DoguInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguInterface interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguInterface creates a new instance of DoguInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguInterface(t mockConstructorTestingTNewDoguInterface) *DoguInterface {
	mock := &DoguInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

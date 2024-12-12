// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	policyv1 "k8s.io/api/policy/v1"

	rest "k8s.io/client-go/rest"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/client-go/applyconfigurations/core/v1"

	v1beta1 "k8s.io/api/policy/v1beta1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockPodInterface is an autogenerated mock type for the podInterface type
type mockPodInterface struct {
	mock.Mock
}

type mockPodInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockPodInterface) EXPECT() *mockPodInterface_Expecter {
	return &mockPodInterface_Expecter{mock: &_m.Mock}
}

// Apply provides a mock function with given fields: ctx, pod, opts
func (_m *mockPodInterface) Apply(ctx context.Context, pod *v1.PodApplyConfiguration, opts metav1.ApplyOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for Apply")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) *corev1.Pod); ok {
		r0 = rf(ctx, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_Apply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Apply'
type mockPodInterface_Apply_Call struct {
	*mock.Call
}

// Apply is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *v1.PodApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockPodInterface_Expecter) Apply(ctx interface{}, pod interface{}, opts interface{}) *mockPodInterface_Apply_Call {
	return &mockPodInterface_Apply_Call{Call: _e.mock.On("Apply", ctx, pod, opts)}
}

func (_c *mockPodInterface_Apply_Call) Run(run func(ctx context.Context, pod *v1.PodApplyConfiguration, opts metav1.ApplyOptions)) *mockPodInterface_Apply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.PodApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockPodInterface_Apply_Call) Return(result *corev1.Pod, err error) *mockPodInterface_Apply_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockPodInterface_Apply_Call) RunAndReturn(run func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) (*corev1.Pod, error)) *mockPodInterface_Apply_Call {
	_c.Call.Return(run)
	return _c
}

// ApplyStatus provides a mock function with given fields: ctx, pod, opts
func (_m *mockPodInterface) ApplyStatus(ctx context.Context, pod *v1.PodApplyConfiguration, opts metav1.ApplyOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for ApplyStatus")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) *corev1.Pod); ok {
		r0 = rf(ctx, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_ApplyStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyStatus'
type mockPodInterface_ApplyStatus_Call struct {
	*mock.Call
}

// ApplyStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *v1.PodApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockPodInterface_Expecter) ApplyStatus(ctx interface{}, pod interface{}, opts interface{}) *mockPodInterface_ApplyStatus_Call {
	return &mockPodInterface_ApplyStatus_Call{Call: _e.mock.On("ApplyStatus", ctx, pod, opts)}
}

func (_c *mockPodInterface_ApplyStatus_Call) Run(run func(ctx context.Context, pod *v1.PodApplyConfiguration, opts metav1.ApplyOptions)) *mockPodInterface_ApplyStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.PodApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockPodInterface_ApplyStatus_Call) Return(result *corev1.Pod, err error) *mockPodInterface_ApplyStatus_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockPodInterface_ApplyStatus_Call) RunAndReturn(run func(context.Context, *v1.PodApplyConfiguration, metav1.ApplyOptions) (*corev1.Pod, error)) *mockPodInterface_ApplyStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Bind provides a mock function with given fields: ctx, binding, opts
func (_m *mockPodInterface) Bind(ctx context.Context, binding *corev1.Binding, opts metav1.CreateOptions) error {
	ret := _m.Called(ctx, binding, opts)

	if len(ret) == 0 {
		panic("no return value specified for Bind")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Binding, metav1.CreateOptions) error); ok {
		r0 = rf(ctx, binding, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_Bind_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Bind'
type mockPodInterface_Bind_Call struct {
	*mock.Call
}

// Bind is a helper method to define mock.On call
//   - ctx context.Context
//   - binding *corev1.Binding
//   - opts metav1.CreateOptions
func (_e *mockPodInterface_Expecter) Bind(ctx interface{}, binding interface{}, opts interface{}) *mockPodInterface_Bind_Call {
	return &mockPodInterface_Bind_Call{Call: _e.mock.On("Bind", ctx, binding, opts)}
}

func (_c *mockPodInterface_Bind_Call) Run(run func(ctx context.Context, binding *corev1.Binding, opts metav1.CreateOptions)) *mockPodInterface_Bind_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.Binding), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockPodInterface_Bind_Call) Return(_a0 error) *mockPodInterface_Bind_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_Bind_Call) RunAndReturn(run func(context.Context, *corev1.Binding, metav1.CreateOptions) error) *mockPodInterface_Bind_Call {
	_c.Call.Return(run)
	return _c
}

// Create provides a mock function with given fields: ctx, pod, opts
func (_m *mockPodInterface) Create(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.CreateOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.CreateOptions) *corev1.Pod); ok {
		r0 = rf(ctx, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *corev1.Pod, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockPodInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *corev1.Pod
//   - opts metav1.CreateOptions
func (_e *mockPodInterface_Expecter) Create(ctx interface{}, pod interface{}, opts interface{}) *mockPodInterface_Create_Call {
	return &mockPodInterface_Create_Call{Call: _e.mock.On("Create", ctx, pod, opts)}
}

func (_c *mockPodInterface_Create_Call) Run(run func(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions)) *mockPodInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.Pod), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockPodInterface_Create_Call) Return(_a0 *corev1.Pod, _a1 error) *mockPodInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_Create_Call) RunAndReturn(run func(context.Context, *corev1.Pod, metav1.CreateOptions) (*corev1.Pod, error)) *mockPodInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockPodInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockPodInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockPodInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockPodInterface_Delete_Call {
	return &mockPodInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockPodInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockPodInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockPodInterface_Delete_Call) Return(_a0 error) *mockPodInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockPodInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockPodInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	if len(ret) == 0 {
		panic("no return value specified for DeleteCollection")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockPodInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockPodInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockPodInterface_DeleteCollection_Call {
	return &mockPodInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockPodInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockPodInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockPodInterface_DeleteCollection_Call) Return(_a0 error) *mockPodInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockPodInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Evict provides a mock function with given fields: ctx, eviction
func (_m *mockPodInterface) Evict(ctx context.Context, eviction *v1beta1.Eviction) error {
	ret := _m.Called(ctx, eviction)

	if len(ret) == 0 {
		panic("no return value specified for Evict")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1beta1.Eviction) error); ok {
		r0 = rf(ctx, eviction)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_Evict_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Evict'
type mockPodInterface_Evict_Call struct {
	*mock.Call
}

// Evict is a helper method to define mock.On call
//   - ctx context.Context
//   - eviction *v1beta1.Eviction
func (_e *mockPodInterface_Expecter) Evict(ctx interface{}, eviction interface{}) *mockPodInterface_Evict_Call {
	return &mockPodInterface_Evict_Call{Call: _e.mock.On("Evict", ctx, eviction)}
}

func (_c *mockPodInterface_Evict_Call) Run(run func(ctx context.Context, eviction *v1beta1.Eviction)) *mockPodInterface_Evict_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1beta1.Eviction))
	})
	return _c
}

func (_c *mockPodInterface_Evict_Call) Return(_a0 error) *mockPodInterface_Evict_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_Evict_Call) RunAndReturn(run func(context.Context, *v1beta1.Eviction) error) *mockPodInterface_Evict_Call {
	_c.Call.Return(run)
	return _c
}

// EvictV1 provides a mock function with given fields: ctx, eviction
func (_m *mockPodInterface) EvictV1(ctx context.Context, eviction *policyv1.Eviction) error {
	ret := _m.Called(ctx, eviction)

	if len(ret) == 0 {
		panic("no return value specified for EvictV1")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *policyv1.Eviction) error); ok {
		r0 = rf(ctx, eviction)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_EvictV1_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EvictV1'
type mockPodInterface_EvictV1_Call struct {
	*mock.Call
}

// EvictV1 is a helper method to define mock.On call
//   - ctx context.Context
//   - eviction *policyv1.Eviction
func (_e *mockPodInterface_Expecter) EvictV1(ctx interface{}, eviction interface{}) *mockPodInterface_EvictV1_Call {
	return &mockPodInterface_EvictV1_Call{Call: _e.mock.On("EvictV1", ctx, eviction)}
}

func (_c *mockPodInterface_EvictV1_Call) Run(run func(ctx context.Context, eviction *policyv1.Eviction)) *mockPodInterface_EvictV1_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*policyv1.Eviction))
	})
	return _c
}

func (_c *mockPodInterface_EvictV1_Call) Return(_a0 error) *mockPodInterface_EvictV1_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_EvictV1_Call) RunAndReturn(run func(context.Context, *policyv1.Eviction) error) *mockPodInterface_EvictV1_Call {
	_c.Call.Return(run)
	return _c
}

// EvictV1beta1 provides a mock function with given fields: ctx, eviction
func (_m *mockPodInterface) EvictV1beta1(ctx context.Context, eviction *v1beta1.Eviction) error {
	ret := _m.Called(ctx, eviction)

	if len(ret) == 0 {
		panic("no return value specified for EvictV1beta1")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1beta1.Eviction) error); ok {
		r0 = rf(ctx, eviction)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPodInterface_EvictV1beta1_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EvictV1beta1'
type mockPodInterface_EvictV1beta1_Call struct {
	*mock.Call
}

// EvictV1beta1 is a helper method to define mock.On call
//   - ctx context.Context
//   - eviction *v1beta1.Eviction
func (_e *mockPodInterface_Expecter) EvictV1beta1(ctx interface{}, eviction interface{}) *mockPodInterface_EvictV1beta1_Call {
	return &mockPodInterface_EvictV1beta1_Call{Call: _e.mock.On("EvictV1beta1", ctx, eviction)}
}

func (_c *mockPodInterface_EvictV1beta1_Call) Run(run func(ctx context.Context, eviction *v1beta1.Eviction)) *mockPodInterface_EvictV1beta1_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1beta1.Eviction))
	})
	return _c
}

func (_c *mockPodInterface_EvictV1beta1_Call) Return(_a0 error) *mockPodInterface_EvictV1beta1_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_EvictV1beta1_Call) RunAndReturn(run func(context.Context, *v1beta1.Eviction) error) *mockPodInterface_EvictV1beta1_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockPodInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *corev1.Pod); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockPodInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockPodInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockPodInterface_Get_Call {
	return &mockPodInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockPodInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockPodInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockPodInterface_Get_Call) Return(_a0 *corev1.Pod, _a1 error) *mockPodInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*corev1.Pod, error)) *mockPodInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetLogs provides a mock function with given fields: name, opts
func (_m *mockPodInterface) GetLogs(name string, opts *corev1.PodLogOptions) *rest.Request {
	ret := _m.Called(name, opts)

	if len(ret) == 0 {
		panic("no return value specified for GetLogs")
	}

	var r0 *rest.Request
	if rf, ok := ret.Get(0).(func(string, *corev1.PodLogOptions) *rest.Request); ok {
		r0 = rf(name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rest.Request)
		}
	}

	return r0
}

// mockPodInterface_GetLogs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetLogs'
type mockPodInterface_GetLogs_Call struct {
	*mock.Call
}

// GetLogs is a helper method to define mock.On call
//   - name string
//   - opts *corev1.PodLogOptions
func (_e *mockPodInterface_Expecter) GetLogs(name interface{}, opts interface{}) *mockPodInterface_GetLogs_Call {
	return &mockPodInterface_GetLogs_Call{Call: _e.mock.On("GetLogs", name, opts)}
}

func (_c *mockPodInterface_GetLogs_Call) Run(run func(name string, opts *corev1.PodLogOptions)) *mockPodInterface_GetLogs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(*corev1.PodLogOptions))
	})
	return _c
}

func (_c *mockPodInterface_GetLogs_Call) Return(_a0 *rest.Request) *mockPodInterface_GetLogs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_GetLogs_Call) RunAndReturn(run func(string, *corev1.PodLogOptions) *rest.Request) *mockPodInterface_GetLogs_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockPodInterface) List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *corev1.PodList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*corev1.PodList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *corev1.PodList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PodList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockPodInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockPodInterface_Expecter) List(ctx interface{}, opts interface{}) *mockPodInterface_List_Call {
	return &mockPodInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockPodInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockPodInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockPodInterface_List_Call) Return(_a0 *corev1.PodList, _a1 error) *mockPodInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*corev1.PodList, error)) *mockPodInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockPodInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*corev1.Pod, error) {
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

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.Pod, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *corev1.Pod); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockPodInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockPodInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockPodInterface_Patch_Call {
	return &mockPodInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockPodInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockPodInterface_Patch_Call {
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

func (_c *mockPodInterface_Patch_Call) Return(result *corev1.Pod, err error) *mockPodInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockPodInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.Pod, error)) *mockPodInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// ProxyGet provides a mock function with given fields: scheme, name, port, path, params
func (_m *mockPodInterface) ProxyGet(scheme string, name string, port string, path string, params map[string]string) rest.ResponseWrapper {
	ret := _m.Called(scheme, name, port, path, params)

	if len(ret) == 0 {
		panic("no return value specified for ProxyGet")
	}

	var r0 rest.ResponseWrapper
	if rf, ok := ret.Get(0).(func(string, string, string, string, map[string]string) rest.ResponseWrapper); ok {
		r0 = rf(scheme, name, port, path, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rest.ResponseWrapper)
		}
	}

	return r0
}

// mockPodInterface_ProxyGet_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProxyGet'
type mockPodInterface_ProxyGet_Call struct {
	*mock.Call
}

// ProxyGet is a helper method to define mock.On call
//   - scheme string
//   - name string
//   - port string
//   - path string
//   - params map[string]string
func (_e *mockPodInterface_Expecter) ProxyGet(scheme interface{}, name interface{}, port interface{}, path interface{}, params interface{}) *mockPodInterface_ProxyGet_Call {
	return &mockPodInterface_ProxyGet_Call{Call: _e.mock.On("ProxyGet", scheme, name, port, path, params)}
}

func (_c *mockPodInterface_ProxyGet_Call) Run(run func(scheme string, name string, port string, path string, params map[string]string)) *mockPodInterface_ProxyGet_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(string), args[3].(string), args[4].(map[string]string))
	})
	return _c
}

func (_c *mockPodInterface_ProxyGet_Call) Return(_a0 rest.ResponseWrapper) *mockPodInterface_ProxyGet_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPodInterface_ProxyGet_Call) RunAndReturn(run func(string, string, string, string, map[string]string) rest.ResponseWrapper) *mockPodInterface_ProxyGet_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, pod, opts
func (_m *mockPodInterface) Update(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) *corev1.Pod); ok {
		r0 = rf(ctx, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockPodInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *corev1.Pod
//   - opts metav1.UpdateOptions
func (_e *mockPodInterface_Expecter) Update(ctx interface{}, pod interface{}, opts interface{}) *mockPodInterface_Update_Call {
	return &mockPodInterface_Update_Call{Call: _e.mock.On("Update", ctx, pod, opts)}
}

func (_c *mockPodInterface_Update_Call) Run(run func(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions)) *mockPodInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.Pod), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockPodInterface_Update_Call) Return(_a0 *corev1.Pod, _a1 error) *mockPodInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_Update_Call) RunAndReturn(run func(context.Context, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)) *mockPodInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateEphemeralContainers provides a mock function with given fields: ctx, podName, pod, opts
func (_m *mockPodInterface) UpdateEphemeralContainers(ctx context.Context, podName string, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, podName, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateEphemeralContainers")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, podName, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *corev1.Pod, metav1.UpdateOptions) *corev1.Pod); ok {
		r0 = rf(ctx, podName, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *corev1.Pod, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, podName, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_UpdateEphemeralContainers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateEphemeralContainers'
type mockPodInterface_UpdateEphemeralContainers_Call struct {
	*mock.Call
}

// UpdateEphemeralContainers is a helper method to define mock.On call
//   - ctx context.Context
//   - podName string
//   - pod *corev1.Pod
//   - opts metav1.UpdateOptions
func (_e *mockPodInterface_Expecter) UpdateEphemeralContainers(ctx interface{}, podName interface{}, pod interface{}, opts interface{}) *mockPodInterface_UpdateEphemeralContainers_Call {
	return &mockPodInterface_UpdateEphemeralContainers_Call{Call: _e.mock.On("UpdateEphemeralContainers", ctx, podName, pod, opts)}
}

func (_c *mockPodInterface_UpdateEphemeralContainers_Call) Run(run func(ctx context.Context, podName string, pod *corev1.Pod, opts metav1.UpdateOptions)) *mockPodInterface_UpdateEphemeralContainers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*corev1.Pod), args[3].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockPodInterface_UpdateEphemeralContainers_Call) Return(_a0 *corev1.Pod, _a1 error) *mockPodInterface_UpdateEphemeralContainers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_UpdateEphemeralContainers_Call) RunAndReturn(run func(context.Context, string, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)) *mockPodInterface_UpdateEphemeralContainers_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, pod, opts
func (_m *mockPodInterface) UpdateStatus(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions) (*corev1.Pod, error) {
	ret := _m.Called(ctx, pod, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatus")
	}

	var r0 *corev1.Pod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)); ok {
		return rf(ctx, pod, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) *corev1.Pod); ok {
		r0 = rf(ctx, pod, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Pod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *corev1.Pod, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, pod, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type mockPodInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - pod *corev1.Pod
//   - opts metav1.UpdateOptions
func (_e *mockPodInterface_Expecter) UpdateStatus(ctx interface{}, pod interface{}, opts interface{}) *mockPodInterface_UpdateStatus_Call {
	return &mockPodInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, pod, opts)}
}

func (_c *mockPodInterface_UpdateStatus_Call) Run(run func(ctx context.Context, pod *corev1.Pod, opts metav1.UpdateOptions)) *mockPodInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.Pod), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockPodInterface_UpdateStatus_Call) Return(_a0 *corev1.Pod, _a1 error) *mockPodInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *corev1.Pod, metav1.UpdateOptions) (*corev1.Pod, error)) *mockPodInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockPodInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for Watch")
	}

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

// mockPodInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockPodInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockPodInterface_Expecter) Watch(ctx interface{}, opts interface{}) *mockPodInterface_Watch_Call {
	return &mockPodInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockPodInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockPodInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockPodInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockPodInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockPodInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// newMockPodInterface creates a new instance of mockPodInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockPodInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockPodInterface {
	mock := &mockPodInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

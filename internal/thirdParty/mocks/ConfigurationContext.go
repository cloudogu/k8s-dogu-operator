// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// ConfigurationContext is an autogenerated mock type for the ConfigurationContext type
type ConfigurationContext struct {
	mock.Mock
}

type ConfigurationContext_Expecter struct {
	mock *mock.Mock
}

func (_m *ConfigurationContext) EXPECT() *ConfigurationContext_Expecter {
	return &ConfigurationContext_Expecter{mock: &_m.Mock}
}

// Delete provides a mock function with given fields: key
func (_m *ConfigurationContext) Delete(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type ConfigurationContext_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - key string
func (_e *ConfigurationContext_Expecter) Delete(key interface{}) *ConfigurationContext_Delete_Call {
	return &ConfigurationContext_Delete_Call{Call: _e.mock.On("Delete", key)}
}

func (_c *ConfigurationContext_Delete_Call) Run(run func(key string)) *ConfigurationContext_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ConfigurationContext_Delete_Call) Return(_a0 error) *ConfigurationContext_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_Delete_Call) RunAndReturn(run func(string) error) *ConfigurationContext_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteRecursive provides a mock function with given fields: key
func (_m *ConfigurationContext) DeleteRecursive(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_DeleteRecursive_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteRecursive'
type ConfigurationContext_DeleteRecursive_Call struct {
	*mock.Call
}

// DeleteRecursive is a helper method to define mock.On call
//   - key string
func (_e *ConfigurationContext_Expecter) DeleteRecursive(key interface{}) *ConfigurationContext_DeleteRecursive_Call {
	return &ConfigurationContext_DeleteRecursive_Call{Call: _e.mock.On("DeleteRecursive", key)}
}

func (_c *ConfigurationContext_DeleteRecursive_Call) Run(run func(key string)) *ConfigurationContext_DeleteRecursive_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ConfigurationContext_DeleteRecursive_Call) Return(_a0 error) *ConfigurationContext_DeleteRecursive_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_DeleteRecursive_Call) RunAndReturn(run func(string) error) *ConfigurationContext_DeleteRecursive_Call {
	_c.Call.Return(run)
	return _c
}

// Exists provides a mock function with given fields: key
func (_m *ConfigurationContext) Exists(key string) (bool, error) {
	ret := _m.Called(key)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ConfigurationContext_Exists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exists'
type ConfigurationContext_Exists_Call struct {
	*mock.Call
}

// Exists is a helper method to define mock.On call
//   - key string
func (_e *ConfigurationContext_Expecter) Exists(key interface{}) *ConfigurationContext_Exists_Call {
	return &ConfigurationContext_Exists_Call{Call: _e.mock.On("Exists", key)}
}

func (_c *ConfigurationContext_Exists_Call) Run(run func(key string)) *ConfigurationContext_Exists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ConfigurationContext_Exists_Call) Return(_a0 bool, _a1 error) *ConfigurationContext_Exists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ConfigurationContext_Exists_Call) RunAndReturn(run func(string) (bool, error)) *ConfigurationContext_Exists_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: key
func (_m *ConfigurationContext) Get(key string) (string, error) {
	ret := _m.Called(key)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ConfigurationContext_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type ConfigurationContext_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - key string
func (_e *ConfigurationContext_Expecter) Get(key interface{}) *ConfigurationContext_Get_Call {
	return &ConfigurationContext_Get_Call{Call: _e.mock.On("Get", key)}
}

func (_c *ConfigurationContext_Get_Call) Run(run func(key string)) *ConfigurationContext_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ConfigurationContext_Get_Call) Return(_a0 string, _a1 error) *ConfigurationContext_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ConfigurationContext_Get_Call) RunAndReturn(run func(string) (string, error)) *ConfigurationContext_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetAll provides a mock function with given fields:
func (_m *ConfigurationContext) GetAll() (map[string]string, error) {
	ret := _m.Called()

	var r0 map[string]string
	var r1 error
	if rf, ok := ret.Get(0).(func() (map[string]string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() map[string]string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ConfigurationContext_GetAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAll'
type ConfigurationContext_GetAll_Call struct {
	*mock.Call
}

// GetAll is a helper method to define mock.On call
func (_e *ConfigurationContext_Expecter) GetAll() *ConfigurationContext_GetAll_Call {
	return &ConfigurationContext_GetAll_Call{Call: _e.mock.On("GetAll")}
}

func (_c *ConfigurationContext_GetAll_Call) Run(run func()) *ConfigurationContext_GetAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ConfigurationContext_GetAll_Call) Return(_a0 map[string]string, _a1 error) *ConfigurationContext_GetAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ConfigurationContext_GetAll_Call) RunAndReturn(run func() (map[string]string, error)) *ConfigurationContext_GetAll_Call {
	_c.Call.Return(run)
	return _c
}

// GetOrFalse provides a mock function with given fields: key
func (_m *ConfigurationContext) GetOrFalse(key string) (bool, string, error) {
	ret := _m.Called(key)

	var r0 bool
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(string) (bool, string, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) string); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(key)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ConfigurationContext_GetOrFalse_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetOrFalse'
type ConfigurationContext_GetOrFalse_Call struct {
	*mock.Call
}

// GetOrFalse is a helper method to define mock.On call
//   - key string
func (_e *ConfigurationContext_Expecter) GetOrFalse(key interface{}) *ConfigurationContext_GetOrFalse_Call {
	return &ConfigurationContext_GetOrFalse_Call{Call: _e.mock.On("GetOrFalse", key)}
}

func (_c *ConfigurationContext_GetOrFalse_Call) Run(run func(key string)) *ConfigurationContext_GetOrFalse_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *ConfigurationContext_GetOrFalse_Call) Return(_a0 bool, _a1 string, _a2 error) *ConfigurationContext_GetOrFalse_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *ConfigurationContext_GetOrFalse_Call) RunAndReturn(run func(string) (bool, string, error)) *ConfigurationContext_GetOrFalse_Call {
	_c.Call.Return(run)
	return _c
}

// Refresh provides a mock function with given fields: key, timeToLiveInSeconds
func (_m *ConfigurationContext) Refresh(key string, timeToLiveInSeconds int) error {
	ret := _m.Called(key, timeToLiveInSeconds)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(key, timeToLiveInSeconds)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_Refresh_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Refresh'
type ConfigurationContext_Refresh_Call struct {
	*mock.Call
}

// Refresh is a helper method to define mock.On call
//   - key string
//   - timeToLiveInSeconds int
func (_e *ConfigurationContext_Expecter) Refresh(key interface{}, timeToLiveInSeconds interface{}) *ConfigurationContext_Refresh_Call {
	return &ConfigurationContext_Refresh_Call{Call: _e.mock.On("Refresh", key, timeToLiveInSeconds)}
}

func (_c *ConfigurationContext_Refresh_Call) Run(run func(key string, timeToLiveInSeconds int)) *ConfigurationContext_Refresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(int))
	})
	return _c
}

func (_c *ConfigurationContext_Refresh_Call) Return(_a0 error) *ConfigurationContext_Refresh_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_Refresh_Call) RunAndReturn(run func(string, int) error) *ConfigurationContext_Refresh_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveAll provides a mock function with given fields:
func (_m *ConfigurationContext) RemoveAll() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_RemoveAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAll'
type ConfigurationContext_RemoveAll_Call struct {
	*mock.Call
}

// RemoveAll is a helper method to define mock.On call
func (_e *ConfigurationContext_Expecter) RemoveAll() *ConfigurationContext_RemoveAll_Call {
	return &ConfigurationContext_RemoveAll_Call{Call: _e.mock.On("RemoveAll")}
}

func (_c *ConfigurationContext_RemoveAll_Call) Run(run func()) *ConfigurationContext_RemoveAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ConfigurationContext_RemoveAll_Call) Return(_a0 error) *ConfigurationContext_RemoveAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_RemoveAll_Call) RunAndReturn(run func() error) *ConfigurationContext_RemoveAll_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: key, value
func (_m *ConfigurationContext) Set(key string, value string) error {
	ret := _m.Called(key, value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type ConfigurationContext_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - key string
//   - value string
func (_e *ConfigurationContext_Expecter) Set(key interface{}, value interface{}) *ConfigurationContext_Set_Call {
	return &ConfigurationContext_Set_Call{Call: _e.mock.On("Set", key, value)}
}

func (_c *ConfigurationContext_Set_Call) Run(run func(key string, value string)) *ConfigurationContext_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *ConfigurationContext_Set_Call) Return(_a0 error) *ConfigurationContext_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_Set_Call) RunAndReturn(run func(string, string) error) *ConfigurationContext_Set_Call {
	_c.Call.Return(run)
	return _c
}

// SetWithLifetime provides a mock function with given fields: key, value, timeToLiveInSeconds
func (_m *ConfigurationContext) SetWithLifetime(key string, value string, timeToLiveInSeconds int) error {
	ret := _m.Called(key, value, timeToLiveInSeconds)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, int) error); ok {
		r0 = rf(key, value, timeToLiveInSeconds)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigurationContext_SetWithLifetime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetWithLifetime'
type ConfigurationContext_SetWithLifetime_Call struct {
	*mock.Call
}

// SetWithLifetime is a helper method to define mock.On call
//   - key string
//   - value string
//   - timeToLiveInSeconds int
func (_e *ConfigurationContext_Expecter) SetWithLifetime(key interface{}, value interface{}, timeToLiveInSeconds interface{}) *ConfigurationContext_SetWithLifetime_Call {
	return &ConfigurationContext_SetWithLifetime_Call{Call: _e.mock.On("SetWithLifetime", key, value, timeToLiveInSeconds)}
}

func (_c *ConfigurationContext_SetWithLifetime_Call) Run(run func(key string, value string, timeToLiveInSeconds int)) *ConfigurationContext_SetWithLifetime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(int))
	})
	return _c
}

func (_c *ConfigurationContext_SetWithLifetime_Call) Return(_a0 error) *ConfigurationContext_SetWithLifetime_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ConfigurationContext_SetWithLifetime_Call) RunAndReturn(run func(string, string, int) error) *ConfigurationContext_SetWithLifetime_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewConfigurationContext interface {
	mock.TestingT
	Cleanup(func())
}

// NewConfigurationContext creates a new instance of ConfigurationContext. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewConfigurationContext(t mockConstructorTestingTNewConfigurationContext) *ConfigurationContext {
	mock := &ConfigurationContext{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

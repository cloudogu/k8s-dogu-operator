// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// DoguRegistry is an autogenerated mock type for the DoguRegistry type
type DoguRegistry struct {
	mock.Mock
}

type DoguRegistry_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguRegistry) EXPECT() *DoguRegistry_Expecter {
	return &DoguRegistry_Expecter{mock: &_m.Mock}
}

// Enable provides a mock function with given fields: dogu
func (_m *DoguRegistry) Enable(dogu *core.Dogu) error {
	ret := _m.Called(dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(*core.Dogu) error); ok {
		r0 = rf(dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguRegistry_Enable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enable'
type DoguRegistry_Enable_Call struct {
	*mock.Call
}

// Enable is a helper method to define mock.On call
//  - dogu *core.Dogu
func (_e *DoguRegistry_Expecter) Enable(dogu interface{}) *DoguRegistry_Enable_Call {
	return &DoguRegistry_Enable_Call{Call: _e.mock.On("Enable", dogu)}
}

func (_c *DoguRegistry_Enable_Call) Run(run func(dogu *core.Dogu)) *DoguRegistry_Enable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*core.Dogu))
	})
	return _c
}

func (_c *DoguRegistry_Enable_Call) Return(_a0 error) *DoguRegistry_Enable_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguRegistry_Enable_Call) RunAndReturn(run func(*core.Dogu) error) *DoguRegistry_Enable_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: name
func (_m *DoguRegistry) Get(name string) (*core.Dogu, error) {
	ret := _m.Called(name)

	var r0 *core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*core.Dogu, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *core.Dogu); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguRegistry_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type DoguRegistry_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//  - name string
func (_e *DoguRegistry_Expecter) Get(name interface{}) *DoguRegistry_Get_Call {
	return &DoguRegistry_Get_Call{Call: _e.mock.On("Get", name)}
}

func (_c *DoguRegistry_Get_Call) Run(run func(name string)) *DoguRegistry_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *DoguRegistry_Get_Call) Return(_a0 *core.Dogu, _a1 error) *DoguRegistry_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguRegistry_Get_Call) RunAndReturn(run func(string) (*core.Dogu, error)) *DoguRegistry_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetAll provides a mock function with given fields:
func (_m *DoguRegistry) GetAll() ([]*core.Dogu, error) {
	ret := _m.Called()

	var r0 []*core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]*core.Dogu, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []*core.Dogu); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguRegistry_GetAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAll'
type DoguRegistry_GetAll_Call struct {
	*mock.Call
}

// GetAll is a helper method to define mock.On call
func (_e *DoguRegistry_Expecter) GetAll() *DoguRegistry_GetAll_Call {
	return &DoguRegistry_GetAll_Call{Call: _e.mock.On("GetAll")}
}

func (_c *DoguRegistry_GetAll_Call) Run(run func()) *DoguRegistry_GetAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DoguRegistry_GetAll_Call) Return(_a0 []*core.Dogu, _a1 error) *DoguRegistry_GetAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguRegistry_GetAll_Call) RunAndReturn(run func() ([]*core.Dogu, error)) *DoguRegistry_GetAll_Call {
	_c.Call.Return(run)
	return _c
}

// IsEnabled provides a mock function with given fields: name
func (_m *DoguRegistry) IsEnabled(name string) (bool, error) {
	ret := _m.Called(name)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoguRegistry_IsEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsEnabled'
type DoguRegistry_IsEnabled_Call struct {
	*mock.Call
}

// IsEnabled is a helper method to define mock.On call
//  - name string
func (_e *DoguRegistry_Expecter) IsEnabled(name interface{}) *DoguRegistry_IsEnabled_Call {
	return &DoguRegistry_IsEnabled_Call{Call: _e.mock.On("IsEnabled", name)}
}

func (_c *DoguRegistry_IsEnabled_Call) Run(run func(name string)) *DoguRegistry_IsEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *DoguRegistry_IsEnabled_Call) Return(_a0 bool, _a1 error) *DoguRegistry_IsEnabled_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *DoguRegistry_IsEnabled_Call) RunAndReturn(run func(string) (bool, error)) *DoguRegistry_IsEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// Register provides a mock function with given fields: dogu
func (_m *DoguRegistry) Register(dogu *core.Dogu) error {
	ret := _m.Called(dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(*core.Dogu) error); ok {
		r0 = rf(dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguRegistry_Register_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Register'
type DoguRegistry_Register_Call struct {
	*mock.Call
}

// Register is a helper method to define mock.On call
//  - dogu *core.Dogu
func (_e *DoguRegistry_Expecter) Register(dogu interface{}) *DoguRegistry_Register_Call {
	return &DoguRegistry_Register_Call{Call: _e.mock.On("Register", dogu)}
}

func (_c *DoguRegistry_Register_Call) Run(run func(dogu *core.Dogu)) *DoguRegistry_Register_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*core.Dogu))
	})
	return _c
}

func (_c *DoguRegistry_Register_Call) Return(_a0 error) *DoguRegistry_Register_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguRegistry_Register_Call) RunAndReturn(run func(*core.Dogu) error) *DoguRegistry_Register_Call {
	_c.Call.Return(run)
	return _c
}

// Unregister provides a mock function with given fields: name
func (_m *DoguRegistry) Unregister(name string) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DoguRegistry_Unregister_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Unregister'
type DoguRegistry_Unregister_Call struct {
	*mock.Call
}

// Unregister is a helper method to define mock.On call
//  - name string
func (_e *DoguRegistry_Expecter) Unregister(name interface{}) *DoguRegistry_Unregister_Call {
	return &DoguRegistry_Unregister_Call{Call: _e.mock.On("Unregister", name)}
}

func (_c *DoguRegistry_Unregister_Call) Run(run func(name string)) *DoguRegistry_Unregister_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *DoguRegistry_Unregister_Call) Return(_a0 error) *DoguRegistry_Unregister_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguRegistry_Unregister_Call) RunAndReturn(run func(string) error) *DoguRegistry_Unregister_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguRegistry interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguRegistry creates a new instance of DoguRegistry. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguRegistry(t mockConstructorTestingTNewDoguRegistry) *DoguRegistry {
	mock := &DoguRegistry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

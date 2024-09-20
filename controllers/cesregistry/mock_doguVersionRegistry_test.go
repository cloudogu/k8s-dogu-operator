// Code generated by mockery v2.46.0. DO NOT EDIT.

package cesregistry

import (
	context "context"

	dogu "github.com/cloudogu/k8s-registry-lib/dogu"
	mock "github.com/stretchr/testify/mock"
)

// MockdoguVersionRegistry is an autogenerated mock type for the doguVersionRegistry type
type MockdoguVersionRegistry struct {
	mock.Mock
}

type MockdoguVersionRegistry_Expecter struct {
	mock *mock.Mock
}

func (_m *MockdoguVersionRegistry) EXPECT() *MockdoguVersionRegistry_Expecter {
	return &MockdoguVersionRegistry_Expecter{mock: &_m.Mock}
}

// Enable provides a mock function with given fields: _a0, _a1
func (_m *MockdoguVersionRegistry) Enable(_a0 context.Context, _a1 dogu.DoguVersion) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Enable")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.DoguVersion) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockdoguVersionRegistry_Enable_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enable'
type MockdoguVersionRegistry_Enable_Call struct {
	*mock.Call
}

// Enable is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.DoguVersion
func (_e *MockdoguVersionRegistry_Expecter) Enable(_a0 interface{}, _a1 interface{}) *MockdoguVersionRegistry_Enable_Call {
	return &MockdoguVersionRegistry_Enable_Call{Call: _e.mock.On("Enable", _a0, _a1)}
}

func (_c *MockdoguVersionRegistry_Enable_Call) Run(run func(_a0 context.Context, _a1 dogu.DoguVersion)) *MockdoguVersionRegistry_Enable_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.DoguVersion))
	})
	return _c
}

func (_c *MockdoguVersionRegistry_Enable_Call) Return(_a0 error) *MockdoguVersionRegistry_Enable_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockdoguVersionRegistry_Enable_Call) RunAndReturn(run func(context.Context, dogu.DoguVersion) error) *MockdoguVersionRegistry_Enable_Call {
	_c.Call.Return(run)
	return _c
}

// GetCurrent provides a mock function with given fields: _a0, _a1
func (_m *MockdoguVersionRegistry) GetCurrent(_a0 context.Context, _a1 dogu.SimpleDoguName) (dogu.DoguVersion, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetCurrent")
	}

	var r0 dogu.DoguVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleDoguName) (dogu.DoguVersion, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleDoguName) dogu.DoguVersion); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(dogu.DoguVersion)
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.SimpleDoguName) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockdoguVersionRegistry_GetCurrent_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCurrent'
type MockdoguVersionRegistry_GetCurrent_Call struct {
	*mock.Call
}

// GetCurrent is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.SimpleDoguName
func (_e *MockdoguVersionRegistry_Expecter) GetCurrent(_a0 interface{}, _a1 interface{}) *MockdoguVersionRegistry_GetCurrent_Call {
	return &MockdoguVersionRegistry_GetCurrent_Call{Call: _e.mock.On("GetCurrent", _a0, _a1)}
}

func (_c *MockdoguVersionRegistry_GetCurrent_Call) Run(run func(_a0 context.Context, _a1 dogu.SimpleDoguName)) *MockdoguVersionRegistry_GetCurrent_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleDoguName))
	})
	return _c
}

func (_c *MockdoguVersionRegistry_GetCurrent_Call) Return(_a0 dogu.DoguVersion, _a1 error) *MockdoguVersionRegistry_GetCurrent_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockdoguVersionRegistry_GetCurrent_Call) RunAndReturn(run func(context.Context, dogu.SimpleDoguName) (dogu.DoguVersion, error)) *MockdoguVersionRegistry_GetCurrent_Call {
	_c.Call.Return(run)
	return _c
}

// GetCurrentOfAll provides a mock function with given fields: _a0
func (_m *MockdoguVersionRegistry) GetCurrentOfAll(_a0 context.Context) ([]dogu.DoguVersion, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetCurrentOfAll")
	}

	var r0 []dogu.DoguVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]dogu.DoguVersion, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []dogu.DoguVersion); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dogu.DoguVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockdoguVersionRegistry_GetCurrentOfAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCurrentOfAll'
type MockdoguVersionRegistry_GetCurrentOfAll_Call struct {
	*mock.Call
}

// GetCurrentOfAll is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockdoguVersionRegistry_Expecter) GetCurrentOfAll(_a0 interface{}) *MockdoguVersionRegistry_GetCurrentOfAll_Call {
	return &MockdoguVersionRegistry_GetCurrentOfAll_Call{Call: _e.mock.On("GetCurrentOfAll", _a0)}
}

func (_c *MockdoguVersionRegistry_GetCurrentOfAll_Call) Run(run func(_a0 context.Context)) *MockdoguVersionRegistry_GetCurrentOfAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockdoguVersionRegistry_GetCurrentOfAll_Call) Return(_a0 []dogu.DoguVersion, _a1 error) *MockdoguVersionRegistry_GetCurrentOfAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockdoguVersionRegistry_GetCurrentOfAll_Call) RunAndReturn(run func(context.Context) ([]dogu.DoguVersion, error)) *MockdoguVersionRegistry_GetCurrentOfAll_Call {
	_c.Call.Return(run)
	return _c
}

// IsEnabled provides a mock function with given fields: _a0, _a1
func (_m *MockdoguVersionRegistry) IsEnabled(_a0 context.Context, _a1 dogu.DoguVersion) (bool, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for IsEnabled")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.DoguVersion) (bool, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.DoguVersion) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.DoguVersion) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockdoguVersionRegistry_IsEnabled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsEnabled'
type MockdoguVersionRegistry_IsEnabled_Call struct {
	*mock.Call
}

// IsEnabled is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.DoguVersion
func (_e *MockdoguVersionRegistry_Expecter) IsEnabled(_a0 interface{}, _a1 interface{}) *MockdoguVersionRegistry_IsEnabled_Call {
	return &MockdoguVersionRegistry_IsEnabled_Call{Call: _e.mock.On("IsEnabled", _a0, _a1)}
}

func (_c *MockdoguVersionRegistry_IsEnabled_Call) Run(run func(_a0 context.Context, _a1 dogu.DoguVersion)) *MockdoguVersionRegistry_IsEnabled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.DoguVersion))
	})
	return _c
}

func (_c *MockdoguVersionRegistry_IsEnabled_Call) Return(_a0 bool, _a1 error) *MockdoguVersionRegistry_IsEnabled_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockdoguVersionRegistry_IsEnabled_Call) RunAndReturn(run func(context.Context, dogu.DoguVersion) (bool, error)) *MockdoguVersionRegistry_IsEnabled_Call {
	_c.Call.Return(run)
	return _c
}

// WatchAllCurrent provides a mock function with given fields: _a0
func (_m *MockdoguVersionRegistry) WatchAllCurrent(_a0 context.Context) (<-chan dogu.CurrentVersionsWatchResult, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for WatchAllCurrent")
	}

	var r0 <-chan dogu.CurrentVersionsWatchResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (<-chan dogu.CurrentVersionsWatchResult, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) <-chan dogu.CurrentVersionsWatchResult); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan dogu.CurrentVersionsWatchResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockdoguVersionRegistry_WatchAllCurrent_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WatchAllCurrent'
type MockdoguVersionRegistry_WatchAllCurrent_Call struct {
	*mock.Call
}

// WatchAllCurrent is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockdoguVersionRegistry_Expecter) WatchAllCurrent(_a0 interface{}) *MockdoguVersionRegistry_WatchAllCurrent_Call {
	return &MockdoguVersionRegistry_WatchAllCurrent_Call{Call: _e.mock.On("WatchAllCurrent", _a0)}
}

func (_c *MockdoguVersionRegistry_WatchAllCurrent_Call) Run(run func(_a0 context.Context)) *MockdoguVersionRegistry_WatchAllCurrent_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockdoguVersionRegistry_WatchAllCurrent_Call) Return(_a0 <-chan dogu.CurrentVersionsWatchResult, _a1 error) *MockdoguVersionRegistry_WatchAllCurrent_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockdoguVersionRegistry_WatchAllCurrent_Call) RunAndReturn(run func(context.Context) (<-chan dogu.CurrentVersionsWatchResult, error)) *MockdoguVersionRegistry_WatchAllCurrent_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockdoguVersionRegistry creates a new instance of MockdoguVersionRegistry. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockdoguVersionRegistry(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockdoguVersionRegistry {
	mock := &MockdoguVersionRegistry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

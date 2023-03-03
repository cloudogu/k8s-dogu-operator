// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

// DoguLimits is an autogenerated mock type for the DoguLimits type
type DoguLimits struct {
	mock.Mock
}

type DoguLimits_Expecter struct {
	mock *mock.Mock
}

func (_m *DoguLimits) EXPECT() *DoguLimits_Expecter {
	return &DoguLimits_Expecter{mock: &_m.Mock}
}

// CpuLimit provides a mock function with given fields:
func (_m *DoguLimits) CpuLimit() *resource.Quantity {
	ret := _m.Called()

	var r0 *resource.Quantity
	if rf, ok := ret.Get(0).(func() *resource.Quantity); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*resource.Quantity)
		}
	}

	return r0
}

// DoguLimits_CpuLimit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CpuLimit'
type DoguLimits_CpuLimit_Call struct {
	*mock.Call
}

// CpuLimit is a helper method to define mock.On call
func (_e *DoguLimits_Expecter) CpuLimit() *DoguLimits_CpuLimit_Call {
	return &DoguLimits_CpuLimit_Call{Call: _e.mock.On("CpuLimit")}
}

func (_c *DoguLimits_CpuLimit_Call) Run(run func()) *DoguLimits_CpuLimit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DoguLimits_CpuLimit_Call) Return(_a0 *resource.Quantity) *DoguLimits_CpuLimit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguLimits_CpuLimit_Call) RunAndReturn(run func() *resource.Quantity) *DoguLimits_CpuLimit_Call {
	_c.Call.Return(run)
	return _c
}

// EphemeralStorageLimit provides a mock function with given fields:
func (_m *DoguLimits) EphemeralStorageLimit() *resource.Quantity {
	ret := _m.Called()

	var r0 *resource.Quantity
	if rf, ok := ret.Get(0).(func() *resource.Quantity); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*resource.Quantity)
		}
	}

	return r0
}

// DoguLimits_EphemeralStorageLimit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EphemeralStorageLimit'
type DoguLimits_EphemeralStorageLimit_Call struct {
	*mock.Call
}

// EphemeralStorageLimit is a helper method to define mock.On call
func (_e *DoguLimits_Expecter) EphemeralStorageLimit() *DoguLimits_EphemeralStorageLimit_Call {
	return &DoguLimits_EphemeralStorageLimit_Call{Call: _e.mock.On("EphemeralStorageLimit")}
}

func (_c *DoguLimits_EphemeralStorageLimit_Call) Run(run func()) *DoguLimits_EphemeralStorageLimit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DoguLimits_EphemeralStorageLimit_Call) Return(_a0 *resource.Quantity) *DoguLimits_EphemeralStorageLimit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguLimits_EphemeralStorageLimit_Call) RunAndReturn(run func() *resource.Quantity) *DoguLimits_EphemeralStorageLimit_Call {
	_c.Call.Return(run)
	return _c
}

// MemoryLimit provides a mock function with given fields:
func (_m *DoguLimits) MemoryLimit() *resource.Quantity {
	ret := _m.Called()

	var r0 *resource.Quantity
	if rf, ok := ret.Get(0).(func() *resource.Quantity); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*resource.Quantity)
		}
	}

	return r0
}

// DoguLimits_MemoryLimit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MemoryLimit'
type DoguLimits_MemoryLimit_Call struct {
	*mock.Call
}

// MemoryLimit is a helper method to define mock.On call
func (_e *DoguLimits_Expecter) MemoryLimit() *DoguLimits_MemoryLimit_Call {
	return &DoguLimits_MemoryLimit_Call{Call: _e.mock.On("MemoryLimit")}
}

func (_c *DoguLimits_MemoryLimit_Call) Run(run func()) *DoguLimits_MemoryLimit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DoguLimits_MemoryLimit_Call) Return(_a0 *resource.Quantity) *DoguLimits_MemoryLimit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DoguLimits_MemoryLimit_Call) RunAndReturn(run func() *resource.Quantity) *DoguLimits_MemoryLimit_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewDoguLimits interface {
	mock.TestingT
	Cleanup(func())
}

// NewDoguLimits creates a new instance of DoguLimits. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDoguLimits(t mockConstructorTestingTNewDoguLimits) *DoguLimits {
	mock := &DoguLimits{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"
)

// HostAliasGenerator is an autogenerated mock type for the HostAliasGenerator type
type HostAliasGenerator struct {
	mock.Mock
}

type HostAliasGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *HostAliasGenerator) EXPECT() *HostAliasGenerator_Expecter {
	return &HostAliasGenerator_Expecter{mock: &_m.Mock}
}

// Generate provides a mock function with given fields:
func (_m *HostAliasGenerator) Generate() ([]v1.HostAlias, error) {
	ret := _m.Called()

	var r0 []v1.HostAlias
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]v1.HostAlias, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []v1.HostAlias); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]v1.HostAlias)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HostAliasGenerator_Generate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Generate'
type HostAliasGenerator_Generate_Call struct {
	*mock.Call
}

// Generate is a helper method to define mock.On call
func (_e *HostAliasGenerator_Expecter) Generate() *HostAliasGenerator_Generate_Call {
	return &HostAliasGenerator_Generate_Call{Call: _e.mock.On("Generate")}
}

func (_c *HostAliasGenerator_Generate_Call) Run(run func()) *HostAliasGenerator_Generate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *HostAliasGenerator_Generate_Call) Return(hostAliases []v1.HostAlias, err error) *HostAliasGenerator_Generate_Call {
	_c.Call.Return(hostAliases, err)
	return _c
}

func (_c *HostAliasGenerator_Generate_Call) RunAndReturn(run func() ([]v1.HostAlias, error)) *HostAliasGenerator_Generate_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewHostAliasGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewHostAliasGenerator creates a new instance of HostAliasGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewHostAliasGenerator(t mockConstructorTestingTNewHostAliasGenerator) *HostAliasGenerator {
	mock := &HostAliasGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

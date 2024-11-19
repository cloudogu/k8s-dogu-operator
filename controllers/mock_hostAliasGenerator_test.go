// Code generated by mockery v2.20.0. DO NOT EDIT.

package controllers

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
)

// mockHostAliasGenerator is an autogenerated mock type for the hostAliasGenerator type
type mockHostAliasGenerator struct {
	mock.Mock
}

type mockHostAliasGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockHostAliasGenerator) EXPECT() *mockHostAliasGenerator_Expecter {
	return &mockHostAliasGenerator_Expecter{mock: &_m.Mock}
}

// Generate provides a mock function with given fields: _a0
func (_m *mockHostAliasGenerator) Generate(_a0 context.Context) ([]v1.HostAlias, error) {
	ret := _m.Called(_a0)

	var r0 []v1.HostAlias
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]v1.HostAlias, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []v1.HostAlias); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]v1.HostAlias)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockHostAliasGenerator_Generate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Generate'
type mockHostAliasGenerator_Generate_Call struct {
	*mock.Call
}

// Generate is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *mockHostAliasGenerator_Expecter) Generate(_a0 interface{}) *mockHostAliasGenerator_Generate_Call {
	return &mockHostAliasGenerator_Generate_Call{Call: _e.mock.On("Generate", _a0)}
}

func (_c *mockHostAliasGenerator_Generate_Call) Run(run func(_a0 context.Context)) *mockHostAliasGenerator_Generate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *mockHostAliasGenerator_Generate_Call) Return(hostAliases []v1.HostAlias, err error) *mockHostAliasGenerator_Generate_Call {
	_c.Call.Return(hostAliases, err)
	return _c
}

func (_c *mockHostAliasGenerator_Generate_Call) RunAndReturn(run func(context.Context) ([]v1.HostAlias, error)) *mockHostAliasGenerator_Generate_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockHostAliasGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// newMockHostAliasGenerator creates a new instance of mockHostAliasGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockHostAliasGenerator(t mockConstructorTestingTnewMockHostAliasGenerator) *mockHostAliasGenerator {
	mock := &mockHostAliasGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

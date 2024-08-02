// Code generated by mockery v2.42.1. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"
)

// RequirementsGenerator is an autogenerated mock type for the RequirementsGenerator type
type RequirementsGenerator struct {
	mock.Mock
}

type RequirementsGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *RequirementsGenerator) EXPECT() *RequirementsGenerator_Expecter {
	return &RequirementsGenerator_Expecter{mock: &_m.Mock}
}

// Generate provides a mock function with given fields: ctx, dogu
func (_m *RequirementsGenerator) Generate(ctx context.Context, dogu *core.Dogu) (v1.ResourceRequirements, error) {
	ret := _m.Called(ctx, dogu)

	if len(ret) == 0 {
		panic("no return value specified for Generate")
	}

	var r0 v1.ResourceRequirements
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) (v1.ResourceRequirements, error)); ok {
		return rf(ctx, dogu)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *core.Dogu) v1.ResourceRequirements); ok {
		r0 = rf(ctx, dogu)
	} else {
		r0 = ret.Get(0).(v1.ResourceRequirements)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *core.Dogu) error); ok {
		r1 = rf(ctx, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RequirementsGenerator_Generate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Generate'
type RequirementsGenerator_Generate_Call struct {
	*mock.Call
}

// Generate is a helper method to define mock.On call
//   - ctx context.Context
//   - dogu *core.Dogu
func (_e *RequirementsGenerator_Expecter) Generate(ctx interface{}, dogu interface{}) *RequirementsGenerator_Generate_Call {
	return &RequirementsGenerator_Generate_Call{Call: _e.mock.On("Generate", ctx, dogu)}
}

func (_c *RequirementsGenerator_Generate_Call) Run(run func(ctx context.Context, dogu *core.Dogu)) *RequirementsGenerator_Generate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *RequirementsGenerator_Generate_Call) Return(_a0 v1.ResourceRequirements, _a1 error) *RequirementsGenerator_Generate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *RequirementsGenerator_Generate_Call) RunAndReturn(run func(context.Context, *core.Dogu) (v1.ResourceRequirements, error)) *RequirementsGenerator_Generate_Call {
	_c.Call.Return(run)
	return _c
}

// NewRequirementsGenerator creates a new instance of RequirementsGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRequirementsGenerator(t interface {
	mock.TestingT
	Cleanup(func())
}) *RequirementsGenerator {
	mock := &RequirementsGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

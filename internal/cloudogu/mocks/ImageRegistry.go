// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	mock "github.com/stretchr/testify/mock"
)

// ImageRegistry is an autogenerated mock type for the ImageRegistry type
type ImageRegistry struct {
	mock.Mock
}

type ImageRegistry_Expecter struct {
	mock *mock.Mock
}

func (_m *ImageRegistry) EXPECT() *ImageRegistry_Expecter {
	return &ImageRegistry_Expecter{mock: &_m.Mock}
}

// PullImageConfig provides a mock function with given fields: ctx, image
func (_m *ImageRegistry) PullImageConfig(ctx context.Context, image string) (*v1.ConfigFile, error) {
	ret := _m.Called(ctx, image)

	var r0 *v1.ConfigFile
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1.ConfigFile, error)); ok {
		return rf(ctx, image)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1.ConfigFile); ok {
		r0 = rf(ctx, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.ConfigFile)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ImageRegistry_PullImageConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PullImageConfig'
type ImageRegistry_PullImageConfig_Call struct {
	*mock.Call
}

// PullImageConfig is a helper method to define mock.On call
//  - ctx context.Context
//  - image string
func (_e *ImageRegistry_Expecter) PullImageConfig(ctx interface{}, image interface{}) *ImageRegistry_PullImageConfig_Call {
	return &ImageRegistry_PullImageConfig_Call{Call: _e.mock.On("PullImageConfig", ctx, image)}
}

func (_c *ImageRegistry_PullImageConfig_Call) Run(run func(ctx context.Context, image string)) *ImageRegistry_PullImageConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *ImageRegistry_PullImageConfig_Call) Return(_a0 *v1.ConfigFile, _a1 error) *ImageRegistry_PullImageConfig_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ImageRegistry_PullImageConfig_Call) RunAndReturn(run func(context.Context, string) (*v1.ConfigFile, error)) *ImageRegistry_PullImageConfig_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewImageRegistry interface {
	mock.TestingT
	Cleanup(func())
}

// NewImageRegistry creates a new instance of ImageRegistry. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewImageRegistry(t mockConstructorTestingTNewImageRegistry) *ImageRegistry {
	mock := &ImageRegistry{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

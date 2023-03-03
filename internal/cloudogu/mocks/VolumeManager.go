// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	context "context"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// VolumeManager is an autogenerated mock type for the VolumeManager type
type VolumeManager struct {
	mock.Mock
}

type VolumeManager_Expecter struct {
	mock *mock.Mock
}

func (_m *VolumeManager) EXPECT() *VolumeManager_Expecter {
	return &VolumeManager_Expecter{mock: &_m.Mock}
}

// SetDoguDataVolumeSize provides a mock function with given fields: ctx, doguResource
func (_m *VolumeManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// VolumeManager_SetDoguDataVolumeSize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguDataVolumeSize'
type VolumeManager_SetDoguDataVolumeSize_Call struct {
	*mock.Call
}

// SetDoguDataVolumeSize is a helper method to define mock.On call
//  - ctx context.Context
//  - doguResource *v1.Dogu
func (_e *VolumeManager_Expecter) SetDoguDataVolumeSize(ctx interface{}, doguResource interface{}) *VolumeManager_SetDoguDataVolumeSize_Call {
	return &VolumeManager_SetDoguDataVolumeSize_Call{Call: _e.mock.On("SetDoguDataVolumeSize", ctx, doguResource)}
}

func (_c *VolumeManager_SetDoguDataVolumeSize_Call) Run(run func(ctx context.Context, doguResource *v1.Dogu)) *VolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Dogu))
	})
	return _c
}

func (_c *VolumeManager_SetDoguDataVolumeSize_Call) Return(_a0 error) *VolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *VolumeManager_SetDoguDataVolumeSize_Call) RunAndReturn(run func(context.Context, *v1.Dogu) error) *VolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewVolumeManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewVolumeManager creates a new instance of VolumeManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewVolumeManager(t mockConstructorTestingTNewVolumeManager) *VolumeManager {
	mock := &VolumeManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

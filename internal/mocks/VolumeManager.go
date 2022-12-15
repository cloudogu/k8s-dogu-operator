// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// VolumeManager is an autogenerated mock type for the VolumeManager type
type VolumeManager struct {
	mock.Mock
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

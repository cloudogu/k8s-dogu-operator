// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// FileExtractor is an autogenerated mock type for the FileExtractor type
type FileExtractor struct {
	mock.Mock
}

// ExtractK8sResourcesFromContainer provides a mock function with given fields: ctx, doguResource, dogu
func (_m *FileExtractor) ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	ret := _m.Called(ctx, doguResource, dogu)

	var r0 map[string]string
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) map[string]string); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu, *core.Dogu) error); ok {
		r1 = rf(ctx, doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewFileExtractor interface {
	mock.TestingT
	Cleanup(func())
}

// NewFileExtractor creates a new instance of FileExtractor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFileExtractor(t mockConstructorTestingTNewFileExtractor) *FileExtractor {
	mock := &FileExtractor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

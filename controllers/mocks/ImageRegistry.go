// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ImageRegistry is an autogenerated mock type for the ImageRegistry type
type ImageRegistry struct {
	mock.Mock
}

// PullImageConfig provides a mock function with given fields: ctx, image
func (_m *ImageRegistry) PullImageConfig(ctx context.Context, image string) (*v1.ConfigFile, error) {
	ret := _m.Called(ctx, image)

	var r0 *v1.ConfigFile
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1.ConfigFile); ok {
		r0 = rf(ctx, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.ConfigFile)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

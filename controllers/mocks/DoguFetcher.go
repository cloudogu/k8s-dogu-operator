// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguFetcher is an autogenerated mock type for the DoguFetcher type
type DoguFetcher struct {
	mock.Mock
}

// FetchFromResource provides a mock function with given fields: ctx, doguResource
func (_m *DoguFetcher) FetchFromResource(ctx context.Context, doguResource *v1.Dogu) (*core.Dogu, *v1.DevelopmentDoguMap, error) {
	ret := _m.Called(ctx, doguResource)

	var r0 *core.Dogu
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) *core.Dogu); ok {
		r0 = rf(ctx, doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	var r1 *v1.DevelopmentDoguMap
	if rf, ok := ret.Get(1).(func(context.Context, *v1.Dogu) *v1.DevelopmentDoguMap); ok {
		r1 = rf(ctx, doguResource)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*v1.DevelopmentDoguMap)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *v1.Dogu) error); ok {
		r2 = rf(ctx, doguResource)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// FetchInstalled provides a mock function with given fields: doguName
func (_m *DoguFetcher) FetchInstalled(doguName string) (*core.Dogu, error) {
	ret := _m.Called(doguName)

	var r0 *core.Dogu
	if rf, ok := ret.Get(0).(func(string) *core.Dogu); ok {
		r0 = rf(doguName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(doguName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTnewDoguFetcher interface {
	mock.TestingT
	Cleanup(func())
}

// newDoguFetcher creates a new instance of DoguFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newDoguFetcher(t mockConstructorTestingTnewDoguFetcher) *DoguFetcher {
	mock := &DoguFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

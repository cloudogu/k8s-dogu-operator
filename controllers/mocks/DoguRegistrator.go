// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguRegistrator is an autogenerated mock type for the DoguRegistrator type
type DoguRegistrator struct {
	mock.Mock
}

// RegisterDogu provides a mock function with given fields: ctx, doguResource, dogu
func (_m *DoguRegistrator) RegisterDogu(ctx context.Context, doguResource *v1.Dogu, dogu *core.Dogu) error {
	ret := _m.Called(ctx, doguResource, dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu, *core.Dogu) error); ok {
		r0 = rf(ctx, doguResource, dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UnRegisterDogu provides a mock function with given fields: dogu
func (_m *DoguRegistrator) UnregisterDogu(dogu string) error {
	ret := _m.Called(dogu)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(dogu)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

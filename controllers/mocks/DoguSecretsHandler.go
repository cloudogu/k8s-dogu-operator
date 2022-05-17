// Code generated by mockery v2.10.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguSecretsHandler is an autogenerated mock type for the DoguSecretsHandler type
type DoguSecretsHandler struct {
	mock.Mock
}

// WriteDoguSecretsToRegistry provides a mock function with given fields: ctx, doguResource
func (_m *DoguSecretsHandler) WriteDoguSecretsToRegistry(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// UpgradeManager is an autogenerated mock type for the UpgradeManager type
type UpgradeManager struct {
	mock.Mock
}

// Upgrade provides a mock function with given fields: ctx, doguResource
func (_m *UpgradeManager) Upgrade(ctx context.Context, doguResource *v1.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTnewUpgradeManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewUpgradeManager creates a new instance of UpgradeManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewUpgradeManager(t mockConstructorTestingTnewUpgradeManager) *UpgradeManager {
	mock := &UpgradeManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

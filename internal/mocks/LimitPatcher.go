// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	apiv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	internal "github.com/cloudogu/k8s-dogu-operator/internal"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/apps/v1"
)

// LimitPatcher is an autogenerated mock type for the LimitPatcher type
type LimitPatcher struct {
	mock.Mock
}

// PatchDeployment provides a mock function with given fields: deployment, limits
func (_m *LimitPatcher) PatchDeployment(deployment *v1.Deployment, limits internal.DoguLimits) error {
	ret := _m.Called(deployment, limits)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1.Deployment, internal.DoguLimits) error); ok {
		r0 = rf(deployment, limits)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RetrievePodLimits provides a mock function with given fields: doguResource
func (_m *LimitPatcher) RetrievePodLimits(doguResource *apiv1.Dogu) (internal.DoguLimits, error) {
	ret := _m.Called(doguResource)

	var r0 internal.DoguLimits
	if rf, ok := ret.Get(0).(func(*apiv1.Dogu) internal.DoguLimits); ok {
		r0 = rf(doguResource)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.DoguLimits)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*apiv1.Dogu) error); ok {
		r1 = rf(doguResource)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewLimitPatcher interface {
	mock.TestingT
	Cleanup(func())
}

// NewLimitPatcher creates a new instance of LimitPatcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewLimitPatcher(t mockConstructorTestingTNewLimitPatcher) *LimitPatcher {
	mock := &LimitPatcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

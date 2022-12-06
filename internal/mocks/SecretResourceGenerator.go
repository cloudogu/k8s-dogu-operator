// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// SecretResourceGenerator is an autogenerated mock type for the SecretResourceGenerator type
type SecretResourceGenerator struct {
	mock.Mock
}

// CreateDoguSecret provides a mock function with given fields: doguResource, stringData
func (_m *SecretResourceGenerator) CreateDoguSecret(doguResource *v1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	ret := _m.Called(doguResource, stringData)

	var r0 *corev1.Secret
	if rf, ok := ret.Get(0).(func(*v1.Dogu, map[string]string) *corev1.Secret); ok {
		r0 = rf(doguResource, stringData)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu, map[string]string) error); ok {
		r1 = rf(doguResource, stringData)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewSecretResourceGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewSecretResourceGenerator creates a new instance of SecretResourceGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSecretResourceGenerator(t mockConstructorTestingTNewSecretResourceGenerator) *SecretResourceGenerator {
	mock := &SecretResourceGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

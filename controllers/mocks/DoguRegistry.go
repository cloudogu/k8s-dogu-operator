// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	core "github.com/cloudogu/cesapp/v4/core"
	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguRegistry is an autogenerated mock type for the DoguRegistry type
type DoguRegistry struct {
	mock.Mock
}

// GetDogu provides a mock function with given fields: a0
func (_m *DoguRegistry) GetDogu(a0 *v1.Dogu) (*core.Dogu, error) {
	ret := _m.Called(a0)

	var r0 *core.Dogu
	if rf, ok := ret.Get(0).(func(*v1.Dogu) *core.Dogu); ok {
		r0 = rf(a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.Dogu) error); ok {
		r1 = rf(a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

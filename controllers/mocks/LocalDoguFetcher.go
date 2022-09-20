// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"
)

// LocalDoguFetcher is an autogenerated mock type for the LocalDoguFetcher type
type LocalDoguFetcher struct {
	mock.Mock
}

// FetchInstalled provides a mock function with given fields: doguName
func (_m *LocalDoguFetcher) FetchInstalled(doguName string) (*core.Dogu, error) {
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

type mockConstructorTestingTnewLocalDoguFetcher interface {
	mock.TestingT
	Cleanup(func())
}

// newLocalDoguFetcher creates a new instance of LocalDoguFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newLocalDoguFetcher(t mockConstructorTestingTnewLocalDoguFetcher) *LocalDoguFetcher {
	mock := &LocalDoguFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

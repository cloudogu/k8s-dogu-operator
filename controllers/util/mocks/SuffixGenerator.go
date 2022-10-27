// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// SuffixGenerator is an autogenerated mock type for the SuffixGenerator type
type SuffixGenerator struct {
	mock.Mock
}

// String provides a mock function with given fields: length
func (_m *SuffixGenerator) String(length int) string {
	ret := _m.Called(length)

	var r0 string
	if rf, ok := ret.Get(0).(func(int) string); ok {
		r0 = rf(length)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

type mockConstructorTestingTNewSuffixGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewSuffixGenerator creates a new instance of SuffixGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSuffixGenerator(t mockConstructorTestingTNewSuffixGenerator) *SuffixGenerator {
	mock := &SuffixGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// SuffixGenerator is an autogenerated mock type for the SuffixGenerator type
type SuffixGenerator struct {
	mock.Mock
}

type SuffixGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *SuffixGenerator) EXPECT() *SuffixGenerator_Expecter {
	return &SuffixGenerator_Expecter{mock: &_m.Mock}
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

// SuffixGenerator_String_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'String'
type SuffixGenerator_String_Call struct {
	*mock.Call
}

// String is a helper method to define mock.On call
//   - length int
func (_e *SuffixGenerator_Expecter) String(length interface{}) *SuffixGenerator_String_Call {
	return &SuffixGenerator_String_Call{Call: _e.mock.On("String", length)}
}

func (_c *SuffixGenerator_String_Call) Run(run func(length int)) *SuffixGenerator_String_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *SuffixGenerator_String_Call) Return(_a0 string) *SuffixGenerator_String_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SuffixGenerator_String_Call) RunAndReturn(run func(int) string) *SuffixGenerator_String_Call {
	_c.Call.Return(run)
	return _c
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

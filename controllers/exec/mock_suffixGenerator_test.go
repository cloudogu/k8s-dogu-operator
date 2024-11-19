// Code generated by mockery v2.20.0. DO NOT EDIT.

package exec

import mock "github.com/stretchr/testify/mock"

// mockSuffixGenerator is an autogenerated mock type for the suffixGenerator type
type mockSuffixGenerator struct {
	mock.Mock
}

type mockSuffixGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSuffixGenerator) EXPECT() *mockSuffixGenerator_Expecter {
	return &mockSuffixGenerator_Expecter{mock: &_m.Mock}
}

// String provides a mock function with given fields: length
func (_m *mockSuffixGenerator) String(length int) string {
	ret := _m.Called(length)

	var r0 string
	if rf, ok := ret.Get(0).(func(int) string); ok {
		r0 = rf(length)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// mockSuffixGenerator_String_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'String'
type mockSuffixGenerator_String_Call struct {
	*mock.Call
}

// String is a helper method to define mock.On call
//   - length int
func (_e *mockSuffixGenerator_Expecter) String(length interface{}) *mockSuffixGenerator_String_Call {
	return &mockSuffixGenerator_String_Call{Call: _e.mock.On("String", length)}
}

func (_c *mockSuffixGenerator_String_Call) Run(run func(length int)) *mockSuffixGenerator_String_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *mockSuffixGenerator_String_Call) Return(_a0 string) *mockSuffixGenerator_String_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockSuffixGenerator_String_Call) RunAndReturn(run func(int) string) *mockSuffixGenerator_String_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockSuffixGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// newMockSuffixGenerator creates a new instance of mockSuffixGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockSuffixGenerator(t mockConstructorTestingTnewMockSuffixGenerator) *mockSuffixGenerator {
	mock := &mockSuffixGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

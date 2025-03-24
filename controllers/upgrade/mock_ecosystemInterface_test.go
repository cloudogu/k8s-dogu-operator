// Code generated by mockery v2.53.2. DO NOT EDIT.

package upgrade

import (
	ecoSystem "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	mock "github.com/stretchr/testify/mock"
)

// mockEcosystemInterface is an autogenerated mock type for the ecosystemInterface type
type mockEcosystemInterface struct {
	mock.Mock
}

type mockEcosystemInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockEcosystemInterface) EXPECT() *mockEcosystemInterface_Expecter {
	return &mockEcosystemInterface_Expecter{mock: &_m.Mock}
}

// DoguRestarts provides a mock function with given fields: namespace
func (_m *mockEcosystemInterface) DoguRestarts(namespace string) ecoSystem.DoguRestartInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for DoguRestarts")
	}

	var r0 ecoSystem.DoguRestartInterface
	if rf, ok := ret.Get(0).(func(string) ecoSystem.DoguRestartInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ecoSystem.DoguRestartInterface)
		}
	}

	return r0
}

// mockEcosystemInterface_DoguRestarts_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DoguRestarts'
type mockEcosystemInterface_DoguRestarts_Call struct {
	*mock.Call
}

// DoguRestarts is a helper method to define mock.On call
//   - namespace string
func (_e *mockEcosystemInterface_Expecter) DoguRestarts(namespace interface{}) *mockEcosystemInterface_DoguRestarts_Call {
	return &mockEcosystemInterface_DoguRestarts_Call{Call: _e.mock.On("DoguRestarts", namespace)}
}

func (_c *mockEcosystemInterface_DoguRestarts_Call) Run(run func(namespace string)) *mockEcosystemInterface_DoguRestarts_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEcosystemInterface_DoguRestarts_Call) Return(_a0 ecoSystem.DoguRestartInterface) *mockEcosystemInterface_DoguRestarts_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEcosystemInterface_DoguRestarts_Call) RunAndReturn(run func(string) ecoSystem.DoguRestartInterface) *mockEcosystemInterface_DoguRestarts_Call {
	_c.Call.Return(run)
	return _c
}

// Dogus provides a mock function with given fields: namespace
func (_m *mockEcosystemInterface) Dogus(namespace string) ecoSystem.DoguInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Dogus")
	}

	var r0 ecoSystem.DoguInterface
	if rf, ok := ret.Get(0).(func(string) ecoSystem.DoguInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ecoSystem.DoguInterface)
		}
	}

	return r0
}

// mockEcosystemInterface_Dogus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Dogus'
type mockEcosystemInterface_Dogus_Call struct {
	*mock.Call
}

// Dogus is a helper method to define mock.On call
//   - namespace string
func (_e *mockEcosystemInterface_Expecter) Dogus(namespace interface{}) *mockEcosystemInterface_Dogus_Call {
	return &mockEcosystemInterface_Dogus_Call{Call: _e.mock.On("Dogus", namespace)}
}

func (_c *mockEcosystemInterface_Dogus_Call) Run(run func(namespace string)) *mockEcosystemInterface_Dogus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEcosystemInterface_Dogus_Call) Return(_a0 ecoSystem.DoguInterface) *mockEcosystemInterface_Dogus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEcosystemInterface_Dogus_Call) RunAndReturn(run func(string) ecoSystem.DoguInterface) *mockEcosystemInterface_Dogus_Call {
	_c.Call.Return(run)
	return _c
}

// newMockEcosystemInterface creates a new instance of mockEcosystemInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockEcosystemInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockEcosystemInterface {
	mock := &mockEcosystemInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

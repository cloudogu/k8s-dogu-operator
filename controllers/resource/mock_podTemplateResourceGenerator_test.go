// Code generated by mockery v2.46.2. DO NOT EDIT.

package resource

import (
	core "github.com/cloudogu/cesapp-lib/core"
	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// mockPodTemplateResourceGenerator is an autogenerated mock type for the podTemplateResourceGenerator type
type mockPodTemplateResourceGenerator struct {
	mock.Mock
}

type mockPodTemplateResourceGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockPodTemplateResourceGenerator) EXPECT() *mockPodTemplateResourceGenerator_Expecter {
	return &mockPodTemplateResourceGenerator_Expecter{mock: &_m.Mock}
}

// GetPodTemplate provides a mock function with given fields: doguResource, dogu
func (_m *mockPodTemplateResourceGenerator) GetPodTemplate(doguResource *v2.Dogu, dogu *core.Dogu) (*v1.PodTemplateSpec, error) {
	ret := _m.Called(doguResource, dogu)

	if len(ret) == 0 {
		panic("no return value specified for GetPodTemplate")
	}

	var r0 *v1.PodTemplateSpec
	var r1 error
	if rf, ok := ret.Get(0).(func(*v2.Dogu, *core.Dogu) (*v1.PodTemplateSpec, error)); ok {
		return rf(doguResource, dogu)
	}
	if rf, ok := ret.Get(0).(func(*v2.Dogu, *core.Dogu) *v1.PodTemplateSpec); ok {
		r0 = rf(doguResource, dogu)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.PodTemplateSpec)
		}
	}

	if rf, ok := ret.Get(1).(func(*v2.Dogu, *core.Dogu) error); ok {
		r1 = rf(doguResource, dogu)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPodTemplateResourceGenerator_GetPodTemplate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPodTemplate'
type mockPodTemplateResourceGenerator_GetPodTemplate_Call struct {
	*mock.Call
}

// GetPodTemplate is a helper method to define mock.On call
//   - doguResource *v2.Dogu
//   - dogu *core.Dogu
func (_e *mockPodTemplateResourceGenerator_Expecter) GetPodTemplate(doguResource interface{}, dogu interface{}) *mockPodTemplateResourceGenerator_GetPodTemplate_Call {
	return &mockPodTemplateResourceGenerator_GetPodTemplate_Call{Call: _e.mock.On("GetPodTemplate", doguResource, dogu)}
}

func (_c *mockPodTemplateResourceGenerator_GetPodTemplate_Call) Run(run func(doguResource *v2.Dogu, dogu *core.Dogu)) *mockPodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v2.Dogu), args[1].(*core.Dogu))
	})
	return _c
}

func (_c *mockPodTemplateResourceGenerator_GetPodTemplate_Call) Return(_a0 *v1.PodTemplateSpec, _a1 error) *mockPodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPodTemplateResourceGenerator_GetPodTemplate_Call) RunAndReturn(run func(*v2.Dogu, *core.Dogu) (*v1.PodTemplateSpec, error)) *mockPodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(run)
	return _c
}

// newMockPodTemplateResourceGenerator creates a new instance of mockPodTemplateResourceGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockPodTemplateResourceGenerator(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockPodTemplateResourceGenerator {
	mock := &mockPodTemplateResourceGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
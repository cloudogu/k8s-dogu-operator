// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	core "github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// PodTemplateResourceGenerator is an autogenerated mock type for the PodTemplateResourceGenerator type
type PodTemplateResourceGenerator struct {
	mock.Mock
}

type PodTemplateResourceGenerator_Expecter struct {
	mock *mock.Mock
}

func (_m *PodTemplateResourceGenerator) EXPECT() *PodTemplateResourceGenerator_Expecter {
	return &PodTemplateResourceGenerator_Expecter{mock: &_m.Mock}
}

// GetPodTemplate provides a mock function with given fields: doguResource, dogu, chownInitImage
func (_m *PodTemplateResourceGenerator) GetPodTemplate(doguResource *v1.Dogu, dogu *core.Dogu, chownInitImage string) (*corev1.PodTemplateSpec, error) {
	ret := _m.Called(doguResource, dogu, chownInitImage)

	var r0 *corev1.PodTemplateSpec
	var r1 error
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu, string) (*corev1.PodTemplateSpec, error)); ok {
		return rf(doguResource, dogu, chownInitImage)
	}
	if rf, ok := ret.Get(0).(func(*v1.Dogu, *core.Dogu, string) *corev1.PodTemplateSpec); ok {
		r0 = rf(doguResource, dogu, chownInitImage)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.PodTemplateSpec)
		}
	}

	if rf, ok := ret.Get(1).(func(*v1.Dogu, *core.Dogu, string) error); ok {
		r1 = rf(doguResource, dogu, chownInitImage)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PodTemplateResourceGenerator_GetPodTemplate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPodTemplate'
type PodTemplateResourceGenerator_GetPodTemplate_Call struct {
	*mock.Call
}

// GetPodTemplate is a helper method to define mock.On call
//   - doguResource *v1.Dogu
//   - dogu *core.Dogu
//   - chownInitImage string
func (_e *PodTemplateResourceGenerator_Expecter) GetPodTemplate(doguResource interface{}, dogu interface{}, chownInitImage interface{}) *PodTemplateResourceGenerator_GetPodTemplate_Call {
	return &PodTemplateResourceGenerator_GetPodTemplate_Call{Call: _e.mock.On("GetPodTemplate", doguResource, dogu, chownInitImage)}
}

func (_c *PodTemplateResourceGenerator_GetPodTemplate_Call) Run(run func(doguResource *v1.Dogu, dogu *core.Dogu, chownInitImage string)) *PodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1.Dogu), args[1].(*core.Dogu), args[2].(string))
	})
	return _c
}

func (_c *PodTemplateResourceGenerator_GetPodTemplate_Call) Return(_a0 *corev1.PodTemplateSpec, _a1 error) *PodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PodTemplateResourceGenerator_GetPodTemplate_Call) RunAndReturn(run func(*v1.Dogu, *core.Dogu, string) (*corev1.PodTemplateSpec, error)) *PodTemplateResourceGenerator_GetPodTemplate_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewPodTemplateResourceGenerator interface {
	mock.TestingT
	Cleanup(func())
}

// NewPodTemplateResourceGenerator creates a new instance of PodTemplateResourceGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPodTemplateResourceGenerator(t mockConstructorTestingTNewPodTemplateResourceGenerator) *PodTemplateResourceGenerator {
	mock := &PodTemplateResourceGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
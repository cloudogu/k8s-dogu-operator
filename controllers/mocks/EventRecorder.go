// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EventRecorder is an autogenerated mock type for the EventRecorder type
type EventRecorder struct {
	mock.Mock
}

// AnnotatedEventf provides a mock function with given fields: object, annotations, eventtype, reason, messageFmt, args
func (_m *EventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, annotations, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

// Event provides a mock function with given fields: object, eventtype, reason, message
func (_m *EventRecorder) Event(object runtime.Object, eventtype string, reason string, message string) {
	_m.Called(object, eventtype, reason, message)
}

// Eventf provides a mock function with given fields: object, eventtype, reason, messageFmt, args
func (_m *EventRecorder) Eventf(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

type mockConstructorTestingTNewEventRecorder interface {
	mock.TestingT
	Cleanup(func())
}

// NewEventRecorder creates a new instance of EventRecorder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewEventRecorder(t mockConstructorTestingTNewEventRecorder) *EventRecorder {
	mock := &EventRecorder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

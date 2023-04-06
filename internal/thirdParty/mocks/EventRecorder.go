// Code generated by mockery v2.20.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EventRecorder is an autogenerated mock type for the EventRecorder type
type EventRecorder struct {
	mock.Mock
}

type EventRecorder_Expecter struct {
	mock *mock.Mock
}

func (_m *EventRecorder) EXPECT() *EventRecorder_Expecter {
	return &EventRecorder_Expecter{mock: &_m.Mock}
}

// AnnotatedEventf provides a mock function with given fields: object, annotations, eventtype, reason, messageFmt, args
func (_m *EventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, annotations, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

// EventRecorder_AnnotatedEventf_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AnnotatedEventf'
type EventRecorder_AnnotatedEventf_Call struct {
	*mock.Call
}

// AnnotatedEventf is a helper method to define mock.On call
//   - object runtime.Object
//   - annotations map[string]string
//   - eventtype string
//   - reason string
//   - messageFmt string
//   - args ...interface{}
func (_e *EventRecorder_Expecter) AnnotatedEventf(object interface{}, annotations interface{}, eventtype interface{}, reason interface{}, messageFmt interface{}, args ...interface{}) *EventRecorder_AnnotatedEventf_Call {
	return &EventRecorder_AnnotatedEventf_Call{Call: _e.mock.On("AnnotatedEventf",
		append([]interface{}{object, annotations, eventtype, reason, messageFmt}, args...)...)}
}

func (_c *EventRecorder_AnnotatedEventf_Call) Run(run func(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{})) *EventRecorder_AnnotatedEventf_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-5)
		for i, a := range args[5:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(args[0].(runtime.Object), args[1].(map[string]string), args[2].(string), args[3].(string), args[4].(string), variadicArgs...)
	})
	return _c
}

func (_c *EventRecorder_AnnotatedEventf_Call) Return() *EventRecorder_AnnotatedEventf_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorder_AnnotatedEventf_Call) RunAndReturn(run func(runtime.Object, map[string]string, string, string, string, ...interface{})) *EventRecorder_AnnotatedEventf_Call {
	_c.Call.Return(run)
	return _c
}

// Event provides a mock function with given fields: object, eventtype, reason, message
func (_m *EventRecorder) Event(object runtime.Object, eventtype string, reason string, message string) {
	_m.Called(object, eventtype, reason, message)
}

// EventRecorder_Event_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Event'
type EventRecorder_Event_Call struct {
	*mock.Call
}

// Event is a helper method to define mock.On call
//   - object runtime.Object
//   - eventtype string
//   - reason string
//   - message string
func (_e *EventRecorder_Expecter) Event(object interface{}, eventtype interface{}, reason interface{}, message interface{}) *EventRecorder_Event_Call {
	return &EventRecorder_Event_Call{Call: _e.mock.On("Event", object, eventtype, reason, message)}
}

func (_c *EventRecorder_Event_Call) Run(run func(object runtime.Object, eventtype string, reason string, message string)) *EventRecorder_Event_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(runtime.Object), args[1].(string), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *EventRecorder_Event_Call) Return() *EventRecorder_Event_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorder_Event_Call) RunAndReturn(run func(runtime.Object, string, string, string)) *EventRecorder_Event_Call {
	_c.Call.Return(run)
	return _c
}

// Eventf provides a mock function with given fields: object, eventtype, reason, messageFmt, args
func (_m *EventRecorder) Eventf(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

// EventRecorder_Eventf_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Eventf'
type EventRecorder_Eventf_Call struct {
	*mock.Call
}

// Eventf is a helper method to define mock.On call
//   - object runtime.Object
//   - eventtype string
//   - reason string
//   - messageFmt string
//   - args ...interface{}
func (_e *EventRecorder_Expecter) Eventf(object interface{}, eventtype interface{}, reason interface{}, messageFmt interface{}, args ...interface{}) *EventRecorder_Eventf_Call {
	return &EventRecorder_Eventf_Call{Call: _e.mock.On("Eventf",
		append([]interface{}{object, eventtype, reason, messageFmt}, args...)...)}
}

func (_c *EventRecorder_Eventf_Call) Run(run func(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{})) *EventRecorder_Eventf_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-4)
		for i, a := range args[4:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(args[0].(runtime.Object), args[1].(string), args[2].(string), args[3].(string), variadicArgs...)
	})
	return _c
}

func (_c *EventRecorder_Eventf_Call) Return() *EventRecorder_Eventf_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorder_Eventf_Call) RunAndReturn(run func(runtime.Object, string, string, string, ...interface{})) *EventRecorder_Eventf_Call {
	_c.Call.Return(run)
	return _c
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
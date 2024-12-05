// Code generated by mockery v2.46.2. DO NOT EDIT.

package resource

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// MockCollectApplier is an autogenerated mock type for the CollectApplier type
type MockCollectApplier struct {
	mock.Mock
}

type MockCollectApplier_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCollectApplier) EXPECT() *MockCollectApplier_Expecter {
	return &MockCollectApplier_Expecter{mock: &_m.Mock}
}

// CollectApply provides a mock function with given fields: ctx, customK8sResources, doguResource
func (_m *MockCollectApplier) CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, customK8sResources, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for CollectApply")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, map[string]string, *v2.Dogu) error); ok {
		r0 = rf(ctx, customK8sResources, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockCollectApplier_CollectApply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CollectApply'
type MockCollectApplier_CollectApply_Call struct {
	*mock.Call
}

// CollectApply is a helper method to define mock.On call
//   - ctx context.Context
//   - customK8sResources map[string]string
//   - doguResource *v2.Dogu
func (_e *MockCollectApplier_Expecter) CollectApply(ctx interface{}, customK8sResources interface{}, doguResource interface{}) *MockCollectApplier_CollectApply_Call {
	return &MockCollectApplier_CollectApply_Call{Call: _e.mock.On("CollectApply", ctx, customK8sResources, doguResource)}
}

func (_c *MockCollectApplier_CollectApply_Call) Run(run func(ctx context.Context, customK8sResources map[string]string, doguResource *v2.Dogu)) *MockCollectApplier_CollectApply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(map[string]string), args[2].(*v2.Dogu))
	})
	return _c
}

func (_c *MockCollectApplier_CollectApply_Call) Return(_a0 error) *MockCollectApplier_CollectApply_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockCollectApplier_CollectApply_Call) RunAndReturn(run func(context.Context, map[string]string, *v2.Dogu) error) *MockCollectApplier_CollectApply_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockCollectApplier creates a new instance of MockCollectApplier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCollectApplier(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCollectApplier {
	mock := &MockCollectApplier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

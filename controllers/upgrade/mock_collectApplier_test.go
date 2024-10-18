// Code generated by mockery v2.46.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// mockCollectApplier is an autogenerated mock type for the collectApplier type
type mockCollectApplier struct {
	mock.Mock
}

type mockCollectApplier_Expecter struct {
	mock *mock.Mock
}

func (_m *mockCollectApplier) EXPECT() *mockCollectApplier_Expecter {
	return &mockCollectApplier_Expecter{mock: &_m.Mock}
}

// CollectApply provides a mock function with given fields: ctx, customK8sResources, doguResource
func (_m *mockCollectApplier) CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *v2.Dogu) error {
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

// mockCollectApplier_CollectApply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CollectApply'
type mockCollectApplier_CollectApply_Call struct {
	*mock.Call
}

// CollectApply is a helper method to define mock.On call
//   - ctx context.Context
//   - customK8sResources map[string]string
//   - doguResource *v2.Dogu
func (_e *mockCollectApplier_Expecter) CollectApply(ctx interface{}, customK8sResources interface{}, doguResource interface{}) *mockCollectApplier_CollectApply_Call {
	return &mockCollectApplier_CollectApply_Call{Call: _e.mock.On("CollectApply", ctx, customK8sResources, doguResource)}
}

func (_c *mockCollectApplier_CollectApply_Call) Run(run func(ctx context.Context, customK8sResources map[string]string, doguResource *v2.Dogu)) *mockCollectApplier_CollectApply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(map[string]string), args[2].(*v2.Dogu))
	})
	return _c
}

func (_c *mockCollectApplier_CollectApply_Call) Return(_a0 error) *mockCollectApplier_CollectApply_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockCollectApplier_CollectApply_Call) RunAndReturn(run func(context.Context, map[string]string, *v2.Dogu) error) *mockCollectApplier_CollectApply_Call {
	_c.Call.Return(run)
	return _c
}

// newMockCollectApplier creates a new instance of mockCollectApplier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockCollectApplier(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockCollectApplier {
	mock := &mockCollectApplier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
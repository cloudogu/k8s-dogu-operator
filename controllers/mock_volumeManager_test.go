// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import (
	context "context"

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	mock "github.com/stretchr/testify/mock"
)

// mockVolumeManager is an autogenerated mock type for the volumeManager type
type mockVolumeManager struct {
	mock.Mock
}

type mockVolumeManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockVolumeManager) EXPECT() *mockVolumeManager_Expecter {
	return &mockVolumeManager_Expecter{mock: &_m.Mock}
}

// SetDoguDataVolumeSize provides a mock function with given fields: ctx, doguResource
func (_m *mockVolumeManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *v2.Dogu) error {
	ret := _m.Called(ctx, doguResource)

	if len(ret) == 0 {
		panic("no return value specified for SetDoguDataVolumeSize")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v2.Dogu) error); ok {
		r0 = rf(ctx, doguResource)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockVolumeManager_SetDoguDataVolumeSize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetDoguDataVolumeSize'
type mockVolumeManager_SetDoguDataVolumeSize_Call struct {
	*mock.Call
}

// SetDoguDataVolumeSize is a helper method to define mock.On call
//   - ctx context.Context
//   - doguResource *v2.Dogu
func (_e *mockVolumeManager_Expecter) SetDoguDataVolumeSize(ctx interface{}, doguResource interface{}) *mockVolumeManager_SetDoguDataVolumeSize_Call {
	return &mockVolumeManager_SetDoguDataVolumeSize_Call{Call: _e.mock.On("SetDoguDataVolumeSize", ctx, doguResource)}
}

func (_c *mockVolumeManager_SetDoguDataVolumeSize_Call) Run(run func(ctx context.Context, doguResource *v2.Dogu)) *mockVolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v2.Dogu))
	})
	return _c
}

func (_c *mockVolumeManager_SetDoguDataVolumeSize_Call) Return(_a0 error) *mockVolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVolumeManager_SetDoguDataVolumeSize_Call) RunAndReturn(run func(context.Context, *v2.Dogu) error) *mockVolumeManager_SetDoguDataVolumeSize_Call {
	_c.Call.Return(run)
	return _c
}

// newMockVolumeManager creates a new instance of mockVolumeManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockVolumeManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockVolumeManager {
	mock := &mockVolumeManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.42.1. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	config "github.com/cloudogu/k8s-registry-lib/config"

	mock "github.com/stretchr/testify/mock"
)

// MockSensitiveDoguConfigRepository is an autogenerated mock type for the SensitiveDoguConfigRepository type
type MockSensitiveDoguConfigRepository struct {
	mock.Mock
}

type MockSensitiveDoguConfigRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSensitiveDoguConfigRepository) EXPECT() *MockSensitiveDoguConfigRepository_Expecter {
	return &MockSensitiveDoguConfigRepository_Expecter{mock: &_m.Mock}
}

// Get provides a mock function with given fields: ctx, name
func (_m *MockSensitiveDoguConfigRepository) Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 config.DoguConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, config.SimpleDoguName) (config.DoguConfig, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.SimpleDoguName) config.DoguConfig); ok {
		r0 = rf(ctx, name)
	} else {
		r0 = ret.Get(0).(config.DoguConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.SimpleDoguName) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSensitiveDoguConfigRepository_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockSensitiveDoguConfigRepository_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name config.SimpleDoguName
func (_e *MockSensitiveDoguConfigRepository_Expecter) Get(ctx interface{}, name interface{}) *MockSensitiveDoguConfigRepository_Get_Call {
	return &MockSensitiveDoguConfigRepository_Get_Call{Call: _e.mock.On("Get", ctx, name)}
}

func (_c *MockSensitiveDoguConfigRepository_Get_Call) Run(run func(ctx context.Context, name config.SimpleDoguName)) *MockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(config.SimpleDoguName))
	})
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_Get_Call) Return(_a0 config.DoguConfig, _a1 error) *MockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_Get_Call) RunAndReturn(run func(context.Context, config.SimpleDoguName) (config.DoguConfig, error)) *MockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Return(run)
	return _c
}

// SaveOrMerge provides a mock function with given fields: ctx, doguConfig
func (_m *MockSensitiveDoguConfigRepository) SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error) {
	ret := _m.Called(ctx, doguConfig)

	if len(ret) == 0 {
		panic("no return value specified for SaveOrMerge")
	}

	var r0 config.DoguConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, config.DoguConfig) (config.DoguConfig, error)); ok {
		return rf(ctx, doguConfig)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.DoguConfig) config.DoguConfig); ok {
		r0 = rf(ctx, doguConfig)
	} else {
		r0 = ret.Get(0).(config.DoguConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.DoguConfig) error); ok {
		r1 = rf(ctx, doguConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSensitiveDoguConfigRepository_SaveOrMerge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SaveOrMerge'
type MockSensitiveDoguConfigRepository_SaveOrMerge_Call struct {
	*mock.Call
}

// SaveOrMerge is a helper method to define mock.On call
//   - ctx context.Context
//   - doguConfig config.DoguConfig
func (_e *MockSensitiveDoguConfigRepository_Expecter) SaveOrMerge(ctx interface{}, doguConfig interface{}) *MockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	return &MockSensitiveDoguConfigRepository_SaveOrMerge_Call{Call: _e.mock.On("SaveOrMerge", ctx, doguConfig)}
}

func (_c *MockSensitiveDoguConfigRepository_SaveOrMerge_Call) Run(run func(ctx context.Context, doguConfig config.DoguConfig)) *MockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(config.DoguConfig))
	})
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_SaveOrMerge_Call) Return(_a0 config.DoguConfig, _a1 error) *MockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_SaveOrMerge_Call) RunAndReturn(run func(context.Context, config.DoguConfig) (config.DoguConfig, error)) *MockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, doguConfig
func (_m *MockSensitiveDoguConfigRepository) Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error) {
	ret := _m.Called(ctx, doguConfig)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 config.DoguConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, config.DoguConfig) (config.DoguConfig, error)); ok {
		return rf(ctx, doguConfig)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.DoguConfig) config.DoguConfig); ok {
		r0 = rf(ctx, doguConfig)
	} else {
		r0 = ret.Get(0).(config.DoguConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.DoguConfig) error); ok {
		r1 = rf(ctx, doguConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSensitiveDoguConfigRepository_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type MockSensitiveDoguConfigRepository_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - doguConfig config.DoguConfig
func (_e *MockSensitiveDoguConfigRepository_Expecter) Update(ctx interface{}, doguConfig interface{}) *MockSensitiveDoguConfigRepository_Update_Call {
	return &MockSensitiveDoguConfigRepository_Update_Call{Call: _e.mock.On("Update", ctx, doguConfig)}
}

func (_c *MockSensitiveDoguConfigRepository_Update_Call) Run(run func(ctx context.Context, doguConfig config.DoguConfig)) *MockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(config.DoguConfig))
	})
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_Update_Call) Return(_a0 config.DoguConfig, _a1 error) *MockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSensitiveDoguConfigRepository_Update_Call) RunAndReturn(run func(context.Context, config.DoguConfig) (config.DoguConfig, error)) *MockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSensitiveDoguConfigRepository creates a new instance of MockSensitiveDoguConfigRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSensitiveDoguConfigRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockSensitiveDoguConfigRepository {
	mock := &MockSensitiveDoguConfigRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

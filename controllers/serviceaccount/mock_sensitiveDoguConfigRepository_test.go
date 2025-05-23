// Code generated by mockery v2.53.3. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	config "github.com/cloudogu/k8s-registry-lib/config"

	dogu "github.com/cloudogu/ces-commons-lib/dogu"

	mock "github.com/stretchr/testify/mock"
)

// mockSensitiveDoguConfigRepository is an autogenerated mock type for the sensitiveDoguConfigRepository type
type mockSensitiveDoguConfigRepository struct {
	mock.Mock
}

type mockSensitiveDoguConfigRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *mockSensitiveDoguConfigRepository) EXPECT() *mockSensitiveDoguConfigRepository_Expecter {
	return &mockSensitiveDoguConfigRepository_Expecter{mock: &_m.Mock}
}

// Get provides a mock function with given fields: ctx, name
func (_m *mockSensitiveDoguConfigRepository) Get(ctx context.Context, name dogu.SimpleName) (config.DoguConfig, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 config.DoguConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) (config.DoguConfig, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleName) config.DoguConfig); ok {
		r0 = rf(ctx, name)
	} else {
		r0 = ret.Get(0).(config.DoguConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.SimpleName) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockSensitiveDoguConfigRepository_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockSensitiveDoguConfigRepository_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name dogu.SimpleName
func (_e *mockSensitiveDoguConfigRepository_Expecter) Get(ctx interface{}, name interface{}) *mockSensitiveDoguConfigRepository_Get_Call {
	return &mockSensitiveDoguConfigRepository_Get_Call{Call: _e.mock.On("Get", ctx, name)}
}

func (_c *mockSensitiveDoguConfigRepository_Get_Call) Run(run func(ctx context.Context, name dogu.SimpleName)) *mockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleName))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_Get_Call) Return(_a0 config.DoguConfig, _a1 error) *mockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_Get_Call) RunAndReturn(run func(context.Context, dogu.SimpleName) (config.DoguConfig, error)) *mockSensitiveDoguConfigRepository_Get_Call {
	_c.Call.Return(run)
	return _c
}

// SaveOrMerge provides a mock function with given fields: ctx, doguConfig
func (_m *mockSensitiveDoguConfigRepository) SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error) {
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

// mockSensitiveDoguConfigRepository_SaveOrMerge_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SaveOrMerge'
type mockSensitiveDoguConfigRepository_SaveOrMerge_Call struct {
	*mock.Call
}

// SaveOrMerge is a helper method to define mock.On call
//   - ctx context.Context
//   - doguConfig config.DoguConfig
func (_e *mockSensitiveDoguConfigRepository_Expecter) SaveOrMerge(ctx interface{}, doguConfig interface{}) *mockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	return &mockSensitiveDoguConfigRepository_SaveOrMerge_Call{Call: _e.mock.On("SaveOrMerge", ctx, doguConfig)}
}

func (_c *mockSensitiveDoguConfigRepository_SaveOrMerge_Call) Run(run func(ctx context.Context, doguConfig config.DoguConfig)) *mockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(config.DoguConfig))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_SaveOrMerge_Call) Return(_a0 config.DoguConfig, _a1 error) *mockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_SaveOrMerge_Call) RunAndReturn(run func(context.Context, config.DoguConfig) (config.DoguConfig, error)) *mockSensitiveDoguConfigRepository_SaveOrMerge_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, doguConfig
func (_m *mockSensitiveDoguConfigRepository) Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error) {
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

// mockSensitiveDoguConfigRepository_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockSensitiveDoguConfigRepository_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - doguConfig config.DoguConfig
func (_e *mockSensitiveDoguConfigRepository_Expecter) Update(ctx interface{}, doguConfig interface{}) *mockSensitiveDoguConfigRepository_Update_Call {
	return &mockSensitiveDoguConfigRepository_Update_Call{Call: _e.mock.On("Update", ctx, doguConfig)}
}

func (_c *mockSensitiveDoguConfigRepository_Update_Call) Run(run func(ctx context.Context, doguConfig config.DoguConfig)) *mockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(config.DoguConfig))
	})
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_Update_Call) Return(_a0 config.DoguConfig, _a1 error) *mockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockSensitiveDoguConfigRepository_Update_Call) RunAndReturn(run func(context.Context, config.DoguConfig) (config.DoguConfig, error)) *mockSensitiveDoguConfigRepository_Update_Call {
	_c.Call.Return(run)
	return _c
}

// newMockSensitiveDoguConfigRepository creates a new instance of mockSensitiveDoguConfigRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockSensitiveDoguConfigRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockSensitiveDoguConfigRepository {
	mock := &mockSensitiveDoguConfigRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

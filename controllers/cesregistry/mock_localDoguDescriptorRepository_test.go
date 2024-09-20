// Code generated by mockery v2.46.0. DO NOT EDIT.

package cesregistry

import (
	context "context"

	core "github.com/cloudogu/cesapp-lib/core"
	dogu "github.com/cloudogu/k8s-registry-lib/dogu"

	mock "github.com/stretchr/testify/mock"
)

// MocklocalDoguDescriptorRepository is an autogenerated mock type for the localDoguDescriptorRepository type
type MocklocalDoguDescriptorRepository struct {
	mock.Mock
}

type MocklocalDoguDescriptorRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MocklocalDoguDescriptorRepository) EXPECT() *MocklocalDoguDescriptorRepository_Expecter {
	return &MocklocalDoguDescriptorRepository_Expecter{mock: &_m.Mock}
}

// Add provides a mock function with given fields: _a0, _a1, _a2
func (_m *MocklocalDoguDescriptorRepository) Add(_a0 context.Context, _a1 dogu.SimpleDoguName, _a2 *core.Dogu) error {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for Add")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleDoguName, *core.Dogu) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MocklocalDoguDescriptorRepository_Add_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Add'
type MocklocalDoguDescriptorRepository_Add_Call struct {
	*mock.Call
}

// Add is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.SimpleDoguName
//   - _a2 *core.Dogu
func (_e *MocklocalDoguDescriptorRepository_Expecter) Add(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MocklocalDoguDescriptorRepository_Add_Call {
	return &MocklocalDoguDescriptorRepository_Add_Call{Call: _e.mock.On("Add", _a0, _a1, _a2)}
}

func (_c *MocklocalDoguDescriptorRepository_Add_Call) Run(run func(_a0 context.Context, _a1 dogu.SimpleDoguName, _a2 *core.Dogu)) *MocklocalDoguDescriptorRepository_Add_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleDoguName), args[2].(*core.Dogu))
	})
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_Add_Call) Return(_a0 error) *MocklocalDoguDescriptorRepository_Add_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_Add_Call) RunAndReturn(run func(context.Context, dogu.SimpleDoguName, *core.Dogu) error) *MocklocalDoguDescriptorRepository_Add_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteAll provides a mock function with given fields: _a0, _a1
func (_m *MocklocalDoguDescriptorRepository) DeleteAll(_a0 context.Context, _a1 dogu.SimpleDoguName) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for DeleteAll")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.SimpleDoguName) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MocklocalDoguDescriptorRepository_DeleteAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteAll'
type MocklocalDoguDescriptorRepository_DeleteAll_Call struct {
	*mock.Call
}

// DeleteAll is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.SimpleDoguName
func (_e *MocklocalDoguDescriptorRepository_Expecter) DeleteAll(_a0 interface{}, _a1 interface{}) *MocklocalDoguDescriptorRepository_DeleteAll_Call {
	return &MocklocalDoguDescriptorRepository_DeleteAll_Call{Call: _e.mock.On("DeleteAll", _a0, _a1)}
}

func (_c *MocklocalDoguDescriptorRepository_DeleteAll_Call) Run(run func(_a0 context.Context, _a1 dogu.SimpleDoguName)) *MocklocalDoguDescriptorRepository_DeleteAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.SimpleDoguName))
	})
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_DeleteAll_Call) Return(_a0 error) *MocklocalDoguDescriptorRepository_DeleteAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_DeleteAll_Call) RunAndReturn(run func(context.Context, dogu.SimpleDoguName) error) *MocklocalDoguDescriptorRepository_DeleteAll_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: _a0, _a1
func (_m *MocklocalDoguDescriptorRepository) Get(_a0 context.Context, _a1 dogu.DoguVersion) (*core.Dogu, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, dogu.DoguVersion) (*core.Dogu, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, dogu.DoguVersion) *core.Dogu); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, dogu.DoguVersion) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MocklocalDoguDescriptorRepository_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MocklocalDoguDescriptorRepository_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 dogu.DoguVersion
func (_e *MocklocalDoguDescriptorRepository_Expecter) Get(_a0 interface{}, _a1 interface{}) *MocklocalDoguDescriptorRepository_Get_Call {
	return &MocklocalDoguDescriptorRepository_Get_Call{Call: _e.mock.On("Get", _a0, _a1)}
}

func (_c *MocklocalDoguDescriptorRepository_Get_Call) Run(run func(_a0 context.Context, _a1 dogu.DoguVersion)) *MocklocalDoguDescriptorRepository_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(dogu.DoguVersion))
	})
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_Get_Call) Return(_a0 *core.Dogu, _a1 error) *MocklocalDoguDescriptorRepository_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_Get_Call) RunAndReturn(run func(context.Context, dogu.DoguVersion) (*core.Dogu, error)) *MocklocalDoguDescriptorRepository_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetAll provides a mock function with given fields: _a0, _a1
func (_m *MocklocalDoguDescriptorRepository) GetAll(_a0 context.Context, _a1 []dogu.DoguVersion) (map[dogu.DoguVersion]*core.Dogu, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 map[dogu.DoguVersion]*core.Dogu
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []dogu.DoguVersion) (map[dogu.DoguVersion]*core.Dogu, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []dogu.DoguVersion) map[dogu.DoguVersion]*core.Dogu); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[dogu.DoguVersion]*core.Dogu)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []dogu.DoguVersion) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MocklocalDoguDescriptorRepository_GetAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAll'
type MocklocalDoguDescriptorRepository_GetAll_Call struct {
	*mock.Call
}

// GetAll is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 []dogu.DoguVersion
func (_e *MocklocalDoguDescriptorRepository_Expecter) GetAll(_a0 interface{}, _a1 interface{}) *MocklocalDoguDescriptorRepository_GetAll_Call {
	return &MocklocalDoguDescriptorRepository_GetAll_Call{Call: _e.mock.On("GetAll", _a0, _a1)}
}

func (_c *MocklocalDoguDescriptorRepository_GetAll_Call) Run(run func(_a0 context.Context, _a1 []dogu.DoguVersion)) *MocklocalDoguDescriptorRepository_GetAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]dogu.DoguVersion))
	})
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_GetAll_Call) Return(_a0 map[dogu.DoguVersion]*core.Dogu, _a1 error) *MocklocalDoguDescriptorRepository_GetAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MocklocalDoguDescriptorRepository_GetAll_Call) RunAndReturn(run func(context.Context, []dogu.DoguVersion) (map[dogu.DoguVersion]*core.Dogu, error)) *MocklocalDoguDescriptorRepository_GetAll_Call {
	_c.Call.Return(run)
	return _c
}

// NewMocklocalDoguDescriptorRepository creates a new instance of MocklocalDoguDescriptorRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMocklocalDoguDescriptorRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MocklocalDoguDescriptorRepository {
	mock := &MocklocalDoguDescriptorRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

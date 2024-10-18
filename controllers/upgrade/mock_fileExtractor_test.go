// Code generated by mockery v2.46.2. DO NOT EDIT.

package upgrade

import (
	context "context"

	exec "github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	mock "github.com/stretchr/testify/mock"
)

// mockFileExtractor is an autogenerated mock type for the fileExtractor type
type mockFileExtractor struct {
	mock.Mock
}

type mockFileExtractor_Expecter struct {
	mock *mock.Mock
}

func (_m *mockFileExtractor) EXPECT() *mockFileExtractor_Expecter {
	return &mockFileExtractor_Expecter{mock: &_m.Mock}
}

// ExtractK8sResourcesFromContainer provides a mock function with given fields: ctx, k8sExecPod
func (_m *mockFileExtractor) ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod exec.ExecPod) (map[string]string, error) {
	ret := _m.Called(ctx, k8sExecPod)

	if len(ret) == 0 {
		panic("no return value specified for ExtractK8sResourcesFromContainer")
	}

	var r0 map[string]string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, exec.ExecPod) (map[string]string, error)); ok {
		return rf(ctx, k8sExecPod)
	}
	if rf, ok := ret.Get(0).(func(context.Context, exec.ExecPod) map[string]string); ok {
		r0 = rf(ctx, k8sExecPod)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, exec.ExecPod) error); ok {
		r1 = rf(ctx, k8sExecPod)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockFileExtractor_ExtractK8sResourcesFromContainer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExtractK8sResourcesFromContainer'
type mockFileExtractor_ExtractK8sResourcesFromContainer_Call struct {
	*mock.Call
}

// ExtractK8sResourcesFromContainer is a helper method to define mock.On call
//   - ctx context.Context
//   - k8sExecPod exec.ExecPod
func (_e *mockFileExtractor_Expecter) ExtractK8sResourcesFromContainer(ctx interface{}, k8sExecPod interface{}) *mockFileExtractor_ExtractK8sResourcesFromContainer_Call {
	return &mockFileExtractor_ExtractK8sResourcesFromContainer_Call{Call: _e.mock.On("ExtractK8sResourcesFromContainer", ctx, k8sExecPod)}
}

func (_c *mockFileExtractor_ExtractK8sResourcesFromContainer_Call) Run(run func(ctx context.Context, k8sExecPod exec.ExecPod)) *mockFileExtractor_ExtractK8sResourcesFromContainer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(exec.ExecPod))
	})
	return _c
}

func (_c *mockFileExtractor_ExtractK8sResourcesFromContainer_Call) Return(_a0 map[string]string, _a1 error) *mockFileExtractor_ExtractK8sResourcesFromContainer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockFileExtractor_ExtractK8sResourcesFromContainer_Call) RunAndReturn(run func(context.Context, exec.ExecPod) (map[string]string, error)) *mockFileExtractor_ExtractK8sResourcesFromContainer_Call {
	_c.Call.Return(run)
	return _c
}

// newMockFileExtractor creates a new instance of mockFileExtractor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockFileExtractor(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockFileExtractor {
	mock := &mockFileExtractor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
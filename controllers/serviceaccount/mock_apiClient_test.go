// Code generated by mockery v2.42.1. DO NOT EDIT.

package serviceaccount

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockServiceAccountApiClient is an autogenerated mock type for the serviceAccountApiClient type
type mockServiceAccountApiClient struct {
	mock.Mock
}

type serviceAccountApiClient_Expecter struct {
	mock *mock.Mock
}

func (_m *mockServiceAccountApiClient) EXPECT() *serviceAccountApiClient_Expecter {
	return &serviceAccountApiClient_Expecter{mock: &_m.Mock}
}

// createServiceAccount provides a mock function with given fields: ctx, baseUrl, apiKey, consumer, params
func (_m *mockServiceAccountApiClient) createServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string, params []string) (Credentials, error) {
	ret := _m.Called(ctx, baseUrl, apiKey, consumer, params)

	if len(ret) == 0 {
		panic("no return value specified for createServiceAccount")
	}

	var r0 Credentials
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string) (Credentials, error)); ok {
		return rf(ctx, baseUrl, apiKey, consumer, params)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, []string) Credentials); ok {
		r0 = rf(ctx, baseUrl, apiKey, consumer, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Credentials)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, []string) error); ok {
		r1 = rf(ctx, baseUrl, apiKey, consumer, params)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// serviceAccountApiClient_createServiceAccount_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'createServiceAccount'
type serviceAccountApiClient_createServiceAccount_Call struct {
	*mock.Call
}

// createServiceAccount is a helper method to define mock.On call
//   - ctx context.Context
//   - baseUrl string
//   - apiKey string
//   - consumer string
//   - params []string
func (_e *serviceAccountApiClient_Expecter) createServiceAccount(ctx interface{}, baseUrl interface{}, apiKey interface{}, consumer interface{}, params interface{}) *serviceAccountApiClient_createServiceAccount_Call {
	return &serviceAccountApiClient_createServiceAccount_Call{Call: _e.mock.On("createServiceAccount", ctx, baseUrl, apiKey, consumer, params)}
}

func (_c *serviceAccountApiClient_createServiceAccount_Call) Run(run func(ctx context.Context, baseUrl string, apiKey string, consumer string, params []string)) *serviceAccountApiClient_createServiceAccount_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string), args[4].([]string))
	})
	return _c
}

func (_c *serviceAccountApiClient_createServiceAccount_Call) Return(_a0 Credentials, _a1 error) *serviceAccountApiClient_createServiceAccount_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *serviceAccountApiClient_createServiceAccount_Call) RunAndReturn(run func(context.Context, string, string, string, []string) (Credentials, error)) *serviceAccountApiClient_createServiceAccount_Call {
	_c.Call.Return(run)
	return _c
}

// deleteServiceAccount provides a mock function with given fields: ctx, baseUrl, apiKey, consumer
func (_m *mockServiceAccountApiClient) deleteServiceAccount(ctx context.Context, baseUrl string, apiKey string, consumer string) error {
	ret := _m.Called(ctx, baseUrl, apiKey, consumer)

	if len(ret) == 0 {
		panic("no return value specified for deleteServiceAccount")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, baseUrl, apiKey, consumer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// serviceAccountApiClient_deleteServiceAccount_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'deleteServiceAccount'
type serviceAccountApiClient_deleteServiceAccount_Call struct {
	*mock.Call
}

// deleteServiceAccount is a helper method to define mock.On call
//   - ctx context.Context
//   - baseUrl string
//   - apiKey string
//   - consumer string
func (_e *serviceAccountApiClient_Expecter) deleteServiceAccount(ctx interface{}, baseUrl interface{}, apiKey interface{}, consumer interface{}) *serviceAccountApiClient_deleteServiceAccount_Call {
	return &serviceAccountApiClient_deleteServiceAccount_Call{Call: _e.mock.On("deleteServiceAccount", ctx, baseUrl, apiKey, consumer)}
}

func (_c *serviceAccountApiClient_deleteServiceAccount_Call) Run(run func(ctx context.Context, baseUrl string, apiKey string, consumer string)) *serviceAccountApiClient_deleteServiceAccount_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *serviceAccountApiClient_deleteServiceAccount_Call) Return(_a0 error) *serviceAccountApiClient_deleteServiceAccount_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *serviceAccountApiClient_deleteServiceAccount_Call) RunAndReturn(run func(context.Context, string, string, string) error) *serviceAccountApiClient_deleteServiceAccount_Call {
	_c.Call.Return(run)
	return _c
}

// newMockServiceAccountApiClient creates a new instance of serviceAccountApiClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockServiceAccountApiClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockServiceAccountApiClient {
	mock := &mockServiceAccountApiClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
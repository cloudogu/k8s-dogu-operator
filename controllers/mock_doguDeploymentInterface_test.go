// Code generated by mockery v2.46.2. DO NOT EDIT.

package controllers

import mock "github.com/stretchr/testify/mock"

// mockDoguDeploymentInterface is an autogenerated mock type for the doguDeploymentInterface type
type mockDoguDeploymentInterface struct {
	mock.Mock
}

type mockDoguDeploymentInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockDoguDeploymentInterface) EXPECT() *mockDoguDeploymentInterface_Expecter {
	return &mockDoguDeploymentInterface_Expecter{mock: &_m.Mock}
}

// newMockDoguDeploymentInterface creates a new instance of mockDoguDeploymentInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDoguDeploymentInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockDoguDeploymentInterface {
	mock := &mockDoguDeploymentInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

package controllers

import (
	"context"

	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClient struct {
	mock.Mock
	client.Client // embedded, damit alle anderen Methoden nicht implementiert werden müssen
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

type MockStatusWriter struct {
	mock.Mock
	client.StatusWriter
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	args := m.Called(ctx, obj, subResource)
	return args.Error(0)
}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	args := m.Called(ctx, obj, patch)
	return args.Error(0)
}

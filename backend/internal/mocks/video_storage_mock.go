package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockVideoStorage struct {
	mock.Mock
}

func (m *MockVideoStorage) GenerateSignedUploadURL(ctx context.Context, objectName string, contentType string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, objectName, contentType, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockVideoStorage) FileExists(ctx context.Context, objectName string) (bool, error) {
	args := m.Called(ctx, objectName)
	return args.Bool(0), args.Error(1)
}

func (m *MockVideoStorage) GetFileSize(ctx context.Context, objectName string) (int64, error) {
	args := m.Called(ctx, objectName)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVideoStorage) DeleteFile(ctx context.Context, objectName string) error {
	args := m.Called(ctx, objectName)
	return args.Error(0)
}

func (m *MockVideoStorage) GetStorageURL(objectName string) string {
	args := m.Called(objectName)
	return args.String(0)
}

func (m *MockVideoStorage) GetPublicURL(objectName string) string {
	args := m.Called(objectName)
	return args.String(0)
}

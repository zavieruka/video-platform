package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishVideoUploaded(ctx context.Context, event *models.VideoUploadedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockPublisher) PublishProcessingComplete(ctx context.Context, event *models.VideoProcessingCompleteEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

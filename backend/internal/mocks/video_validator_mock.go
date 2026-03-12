package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateUploadRequest(req *models.UploadURLRequest) *errors.AppError {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*errors.AppError) // Return AppError, not error
}

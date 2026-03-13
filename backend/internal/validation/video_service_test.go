package validation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/mocks"
	"github.com/zavieruka/video-platform/backend/internal/models"
	"github.com/zavieruka/video-platform/backend/internal/services"
)

func newTestVideoService() (
	*services.VideoServiceImpl,
	*mocks.MockVideoRepository,
	*mocks.MockVideoStorage,
	*mocks.MockValidator,
	*mocks.MockPublisher,
) {
	mockRepo := new(mocks.MockVideoRepository)
	mockStorage := new(mocks.MockVideoStorage)
	mockValidator := new(mocks.MockValidator)
	mockPublisher := new(mocks.MockPublisher)

	service := services.NewVideoService(
		mockRepo,
		mockStorage,
		mockValidator,
		1,
		mockPublisher,
		"test-bucket-source",
		true,
	)

	return service, mockRepo, mockStorage, mockValidator, mockPublisher
}

func TestVideoService_RequestUploadURL_Success(t *testing.T) {
	service, mockRepo, mockStorage, mockValidator, _ := newTestVideoService()

	ctx := context.Background()
	req := &models.UploadURLRequest{
		Title:       "Test Video",
		Description: "Test Description",
		FileName:    "test.mp4",
		FileSize:    1024 * 1024 * 10,
		MimeType:    "video/mp4",
	}

	mockValidator.On("ValidateUploadRequest", req).Return(nil)

	var capturedObjectName string
	mockStorage.On(
		"GenerateSignedUploadURL",
		mock.Anything,
		mock.MatchedBy(func(objectName string) bool {
			capturedObjectName = objectName

			return len(objectName) > 11 &&
				objectName[:7] == "videos/" &&
				objectName[len(objectName)-4:] == ".mp4"
		}),
		"video/mp4",
		1*time.Hour,
	).Return("https://signed-url.example.com", nil)

	var capturedStorageObjectName string
	mockStorage.On("GetStorageURL", mock.Anything).
		Run(func(args mock.Arguments) {
			capturedStorageObjectName = args.String(0)
		}).
		Return("gs://test-bucket-source/mock-object")

	var capturedPublicObjectName string
	mockStorage.On("GetPublicURL", mock.Anything).
		Run(func(args mock.Arguments) {
			capturedPublicObjectName = args.String(0)
		}).
		Return("https://storage.googleapis.com/test-bucket-source/mock-object")

	var capturedVideo *models.Video
	mockRepo.On("Create", ctx, mock.MatchedBy(func(v *models.Video) bool {
		capturedVideo = v

		return v.Title == "Test Video" &&
			v.Description == "Test Description" &&
			v.FileName == "test.mp4" &&
			v.FileSize == 1024*1024*10 &&
			v.MimeType == "video/mp4" &&
			v.Status == models.StatusPending &&
			len(v.ID) > 0 &&
			len(v.ObjectName) > 0 &&
			v.StorageURL == "gs://test-bucket-source/mock-object" &&
			v.PublicURL == "https://storage.googleapis.com/test-bucket-source/mock-object" &&
			!v.CreatedAt.IsZero() &&
			!v.UpdatedAt.IsZero()
	})).Return(nil)

	response, err := service.RequestUploadURL(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, response)

	assert.NotEmpty(t, response.VideoID)
	assert.Equal(t, "https://signed-url.example.com", response.UploadURL)
	assert.Equal(t, "Test Video", response.Metadata.Title)
	assert.Equal(t, "Test Description", response.Metadata.Description)
	assert.Equal(t, "test.mp4", response.Metadata.FileName)
	assert.Equal(t, int64(1024*1024*10), response.Metadata.FileSize)
	assert.Equal(t, "video/mp4", response.Metadata.MimeType)
	assert.Equal(t, models.StatusPending, response.Metadata.Status)
	assert.Equal(t, capturedObjectName, response.Metadata.ObjectName)

	require.NotNil(t, capturedVideo)
	assert.Equal(t, capturedVideo.ID, response.VideoID)

	assert.Equal(t, capturedObjectName, capturedVideo.ObjectName)
	assert.Equal(t, capturedObjectName, capturedStorageObjectName)
	assert.Equal(t, capturedObjectName, capturedPublicObjectName)

	assert.Equal(t, "gs://test-bucket-source/mock-object", capturedVideo.StorageURL)
	assert.Equal(t, "https://storage.googleapis.com/test-bucket-source/mock-object", capturedVideo.PublicURL)

	mockValidator.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestVideoService_RequestUploadURL_ValidationFails(t *testing.T) {
	service, mockRepo, mockStorage, mockValidator, _ := newTestVideoService()

	ctx := context.Background()
	req := &models.UploadURLRequest{
		Title:    "",
		FileName: "test.mp4",
		FileSize: 1024,
		MimeType: "video/mp4",
	}

	expectedError := errors.NewValidationError("Title is required", map[string]interface{}{"field": "title"})
	mockValidator.On("ValidateUploadRequest", req).Return(expectedError)

	response, err := service.RequestUploadURL(ctx, req)

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, response)

	mockValidator.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "GenerateSignedUploadURL")
	mockStorage.AssertNotCalled(t, "GetStorageURL")
	mockStorage.AssertNotCalled(t, "GetPublicURL")
	mockRepo.AssertNotCalled(t, "Create")
}

func TestVideoService_RequestUploadURL_StorageError(t *testing.T) {
	service, mockRepo, mockStorage, mockValidator, _ := newTestVideoService()

	ctx := context.Background()
	req := &models.UploadURLRequest{
		Title:    "Test Video",
		FileName: "test.mp4",
		FileSize: 1024,
		MimeType: "video/mp4",
	}

	mockValidator.On("ValidateUploadRequest", req).Return(nil)

	storageError := fmt.Errorf("storage service unavailable")
	mockStorage.On(
		"GenerateSignedUploadURL",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("", storageError)

	response, err := service.RequestUploadURL(ctx, req)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, storageError)

	mockValidator.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "GetStorageURL")
	mockStorage.AssertNotCalled(t, "GetPublicURL")
	mockRepo.AssertNotCalled(t, "Create")
}

func TestVideoService_RequestUploadURL_RepositoryError(t *testing.T) {
	service, mockRepo, mockStorage, mockValidator, _ := newTestVideoService()

	ctx := context.Background()
	req := &models.UploadURLRequest{
		Title:    "Test Video",
		FileName: "test.mp4",
		FileSize: 1024,
		MimeType: "video/mp4",
	}

	mockValidator.On("ValidateUploadRequest", req).Return(nil)
	mockStorage.On(
		"GenerateSignedUploadURL",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return("https://signed-url.example.com", nil)
	mockStorage.On("GetStorageURL", mock.Anything).Return("gs://bucket/videos/video-id.mp4")
	mockStorage.On("GetPublicURL", mock.Anything).Return("https://storage.googleapis.com/bucket/videos/video-id.mp4")

	dbError := fmt.Errorf("database connection failed")
	mockRepo.On("Create", ctx, mock.Anything).Return(dbError)

	response, err := service.RequestUploadURL(ctx, req)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, dbError)

	mockValidator.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestVideoService_RequestUploadURL_NilRequest(t *testing.T) {
	service, mockRepo, mockStorage, mockValidator, _ := newTestVideoService()

	ctx := context.Background()

	expectedError := errors.NewValidationError(
		"request is required",
		map[string]interface{}{"field": "request"},
	)

	mockValidator.
		On("ValidateUploadRequest", (*models.UploadURLRequest)(nil)).
		Return(expectedError)

	response, err := service.RequestUploadURL(ctx, nil)

	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, response)

	mockValidator.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "GenerateSignedUploadURL")
	mockStorage.AssertNotCalled(t, "GetStorageURL")
	mockStorage.AssertNotCalled(t, "GetPublicURL")
	mockRepo.AssertNotCalled(t, "Create")
}

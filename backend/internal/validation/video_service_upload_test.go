package validation

import (
	"context"
	"fmt"
	"strings"
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

// ============================================================================
// RequestUploadURL
// ============================================================================

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

// ============================================================================
// ConfirmUpload
// ============================================================================

func TestVideoService_ConfirmUpload_Success(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(video.FileSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusUploaded, (*string)(nil)).Return(nil).Once()

	var capturedEvent *models.VideoUploadedEvent
	mockPublisher.On("PublishVideoUploaded", ctx, mock.MatchedBy(func(event *models.VideoUploadedEvent) bool {
		capturedEvent = event
		return event.VideoID == videoID &&
			event.ObjectName == video.ObjectName &&
			event.Bucket == "test-bucket-source" &&
			event.FileSize == video.FileSize &&
			event.MimeType == video.MimeType
	})).Return(nil).Once()

	updatedVideo := &models.Video{
		ID:         videoID,
		Title:      video.Title,
		Status:     models.StatusUploaded,
		ObjectName: video.ObjectName,
	}
	mockRepo.On("GetByID", ctx, videoID).Return(updatedVideo, nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.StatusUploaded, result.Status)

	require.NotNil(t, capturedEvent)
	assert.Equal(t, videoID, capturedEvent.VideoID)
	assert.Equal(t, video.ObjectName, capturedEvent.ObjectName)
	assert.Equal(t, video.FileSize, capturedEvent.FileSize)
	assert.Equal(t, video.MimeType, capturedEvent.MimeType)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestVideoService_ConfirmUpload_GetByIDFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	repoErr := fmt.Errorf("repository unavailable")
	mockRepo.On("GetByID", ctx, videoID).Return((*models.Video)(nil), repoErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "FileExists")
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockRepo.AssertNotCalled(t, "UpdateStatus")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_WrongStatus(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := &models.Video{
		ID:     videoID,
		Status: models.StatusUploaded,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in pending status")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "FileExists")
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockRepo.AssertNotCalled(t, "UpdateStatus")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_ExpiredURL(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	video.UploadURLExpiresAt = time.Now().UTC().Add(-1 * time.Hour)

	var capturedErrorMsg *string

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, mock.MatchedBy(func(msg *string) bool {
		capturedErrorMsg = msg
		return msg != nil && strings.Contains(strings.ToLower(*msg), "expired")
	})).Return(nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, strings.ToLower(err.Error()), "expired")

	require.NotNil(t, capturedErrorMsg)
	assert.Contains(t, strings.ToLower(*capturedErrorMsg), "expired")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "FileExists")
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_ExpiredURL_UpdateStatusFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	video.UploadURLExpiresAt = time.Now().UTC().Add(-1 * time.Hour)

	updateErr := fmt.Errorf("failed to persist failed status")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, mock.MatchedBy(func(msg *string) bool {
		return msg != nil && strings.Contains(strings.ToLower(*msg), "expired")
	})).Return(updateErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, strings.ToLower(err.Error()), "expired")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "FileExists")
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_FileExistsFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	storageErr := fmt.Errorf("storage service unavailable")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(false, storageErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, storageErr)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockRepo.AssertNotCalled(t, "UpdateStatus")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_FileNotFound(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(false, nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found in storage")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "GetFileSize")
	mockRepo.AssertNotCalled(t, "UpdateStatus")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_GetFileSizeFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	storageErr := fmt.Errorf("failed to read file metadata")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(int64(0), storageErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, storageErr)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateStatus")
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_FileSizeMismatch(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	expectedSize := int64(1024 * 1024 * 10)
	actualSize := int64(1024 * 1024 * 5)

	video := newPendingVideo(videoID)
	video.FileSize = expectedSize

	var capturedErrorMsg *string

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(actualSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, mock.MatchedBy(func(msg *string) bool {
		capturedErrorMsg = msg
		return msg != nil &&
			strings.Contains(*msg, "File size mismatch") &&
			strings.Contains(*msg, fmt.Sprintf("expected %d", expectedSize)) &&
			strings.Contains(*msg, fmt.Sprintf("got %d", actualSize))
	})).Return(nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, strings.ToLower(err.Error()), "size mismatch")

	require.NotNil(t, capturedErrorMsg)
	assert.Contains(t, *capturedErrorMsg, "File size mismatch")
	assert.Contains(t, *capturedErrorMsg, fmt.Sprintf("expected %d", expectedSize))
	assert.Contains(t, *capturedErrorMsg, fmt.Sprintf("got %d", actualSize))

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_FileSizeMismatch_UpdateStatusFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	expectedSize := int64(1024 * 1024 * 10)
	actualSize := int64(1024 * 1024 * 5)

	video := newPendingVideo(videoID)
	video.FileSize = expectedSize

	updateErr := fmt.Errorf("failed to persist failed status")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(actualSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, mock.MatchedBy(func(msg *string) bool {
		return msg != nil && strings.Contains(*msg, "File size mismatch")
	})).Return(updateErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "File size mismatch")
	assert.Contains(t, err.Error(), fmt.Sprintf("expected %d bytes", expectedSize))
	assert.Contains(t, err.Error(), fmt.Sprintf("got %d bytes", actualSize))

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_UpdateStatusToUploadedFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	updateErr := fmt.Errorf("failed to persist uploaded status")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(video.FileSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusUploaded, (*string)(nil)).Return(updateErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, strings.ToLower(err.Error()), "failed")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_PublishingDisabled(t *testing.T) {
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
		false,
	)

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(video.FileSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusUploaded, (*string)(nil)).Return(nil).Once()

	updatedVideo := &models.Video{
		ID:     videoID,
		Status: models.StatusUploaded,
	}
	mockRepo.On("GetByID", ctx, videoID).Return(updatedVideo, nil).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.StatusUploaded, result.Status)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertNotCalled(t, "PublishVideoUploaded")
}

func TestVideoService_ConfirmUpload_PublishEventFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	pubsubErr := fmt.Errorf("pubsub service unavailable")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(video.FileSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusUploaded, (*string)(nil)).Return(nil).Once()
	mockPublisher.On("PublishVideoUploaded", ctx, mock.Anything).Return(pubsubErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, pubsubErr)
	assert.Contains(t, err.Error(), "failed to publish processing event")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestVideoService_ConfirmUpload_RefetchAfterSuccessFails(t *testing.T) {
	service, mockRepo, mockStorage, _, mockPublisher := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.ConfirmUploadRequest{}

	video := newPendingVideo(videoID)
	repoErr := fmt.Errorf("failed to reload updated video")

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil).Once()
	mockStorage.On("FileExists", ctx, video.ObjectName).Return(true, nil).Once()
	mockStorage.On("GetFileSize", ctx, video.ObjectName).Return(video.FileSize, nil).Once()
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusUploaded, (*string)(nil)).Return(nil).Once()
	mockPublisher.On("PublishVideoUploaded", ctx, mock.Anything).Return(nil).Once()
	mockRepo.On("GetByID", ctx, videoID).Return((*models.Video)(nil), repoErr).Once()

	result, err := service.ConfirmUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

// ============================================================================
// FailUpload
// ============================================================================

func TestVideoService_FailUpload_Success(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.FailUploadRequest{
		Error:   "UPLOAD_FAILED",
		Message: "Connection timeout",
	}

	video := &models.Video{
		ID:     videoID,
		Status: models.StatusPending,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

	expectedErrorMsg := "UPLOAD_FAILED: Connection timeout"
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, &expectedErrorMsg).Return(nil)

	result, err := service.FailUpload(ctx, videoID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, videoID, result.ID)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.Contains(t, result.Message, "marked as failed")

	mockRepo.AssertExpectations(t)
}

func TestVideoService_FailUpload_WrongStatus(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.FailUploadRequest{
		Error:   "UPLOAD_FAILED",
		Message: "Connection timeout",
	}

	video := &models.Video{
		ID:     videoID,
		Status: models.StatusUploaded,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

	result, err := service.FailUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in pending status")

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateStatus")
}

func TestVideoService_FailUpload_VideoNotFound(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "nonexistent"
	req := &models.FailUploadRequest{
		Error:   "UPLOAD_FAILED",
		Message: "Connection timeout",
	}

	notFoundErr := errors.NewNotFoundError("Video", videoID)
	mockRepo.On("GetByID", ctx, videoID).Return(nil, notFoundErr)

	result, err := service.FailUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpdateStatus")
}

func TestVideoService_FailUpload_UpdateStatusFails(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"
	req := &models.FailUploadRequest{
		Error:   "UPLOAD_FAILED",
		Message: "Connection timeout",
	}

	video := &models.Video{
		ID:     videoID,
		Status: models.StatusPending,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

	dbError := fmt.Errorf("database connection failed")
	expectedErrorMsg := "UPLOAD_FAILED: Connection timeout"
	mockRepo.On("UpdateStatus", ctx, videoID, models.StatusFailed, &expectedErrorMsg).Return(dbError)

	result, err := service.FailUpload(ctx, videoID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, dbError)

	mockRepo.AssertExpectations(t)
}

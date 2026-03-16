package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

// ============================================================================
// DeleteVideo
// ============================================================================

func TestVideoService_DeleteVideo_StatusPending(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusPending,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)
	mockRepo.On("Delete", ctx, videoID).Return(nil)

	err := service.DeleteVideo(ctx, videoID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "DeleteFile")
}

func TestVideoService_DeleteVideo_StatusUploaded(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusUploaded,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)
	mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(nil)
	mockRepo.On("Delete", ctx, videoID).Return(nil)

	err := service.DeleteVideo(ctx, videoID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestVideoService_DeleteVideo_StatusReady(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusReady,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)
	mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(nil)
	mockRepo.On("Delete", ctx, videoID).Return(nil)

	err := service.DeleteVideo(ctx, videoID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestVideoService_DeleteVideo_StatusFailed(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusFailed,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)
	mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(fmt.Errorf("file not found"))
	mockRepo.On("Delete", ctx, videoID).Return(nil)

	err := service.DeleteVideo(ctx, videoID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestVideoService_DeleteVideo_StatusProcessing_Blocked(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:     videoID,
		Status: models.StatusProcessing,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

	err := service.DeleteVideo(ctx, videoID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete video while processing")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestVideoService_DeleteVideo_AlreadyDeleted(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "nonexistent"

	notFoundErr := errors.NewNotFoundError("Video", videoID)
	mockRepo.On("GetByID", ctx, videoID).Return(nil, notFoundErr)

	err := service.DeleteVideo(ctx, videoID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestVideoService_DeleteVideo_StorageDeleteFails(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusUploaded,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

	storageError := fmt.Errorf("storage service unavailable")
	mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(storageError)

	err := service.DeleteVideo(ctx, videoID)

	require.Error(t, err)
	assert.ErrorIs(t, err, storageError)
	assert.Contains(t, err.Error(), "failed to delete storage object")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestVideoService_DeleteVideo_RepositoryDeleteFails(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	video := &models.Video{
		ID:         videoID,
		Status:     models.StatusUploaded,
		ObjectName: "videos/video-123.mp4",
	}

	mockRepo.On("GetByID", ctx, videoID).Return(video, nil)
	mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(nil)

	dbError := fmt.Errorf("database deletion failed")
	mockRepo.On("Delete", ctx, videoID).Return(dbError)

	err := service.DeleteVideo(ctx, videoID)

	require.Error(t, err)
	assert.ErrorIs(t, err, dbError)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestVideoService_DeleteVideo_GetByIDFails(t *testing.T) {
	service, mockRepo, mockStorage, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	dbError := fmt.Errorf("database connection failed")
	mockRepo.On("GetByID", ctx, videoID).Return(nil, dbError)

	err := service.DeleteVideo(ctx, videoID)

	require.Error(t, err)
	assert.ErrorIs(t, err, dbError)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertNotCalled(t, "DeleteFile")
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestVideoService_DeleteVideo_AllStatuses(t *testing.T) {
	tests := []struct {
		name                string
		status              models.VideoStatus
		shouldDeleteStorage bool
		shouldBlockDelete   bool
	}{
		{"pending - no storage delete", models.StatusPending, false, false},
		{"uploaded - delete storage", models.StatusUploaded, true, false},
		{"processing - blocked", models.StatusProcessing, false, true},
		{"ready - delete storage", models.StatusReady, true, false},
		{"failed - delete storage (ignore errors)", models.StatusFailed, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockRepo, mockStorage, _, _ := newTestVideoService()

			ctx := context.Background()
			videoID := "video-123"

			video := &models.Video{
				ID:         videoID,
				Status:     tt.status,
				ObjectName: "videos/video-123.mp4",
			}

			mockRepo.On("GetByID", ctx, videoID).Return(video, nil)

			if tt.shouldBlockDelete {
				err := service.DeleteVideo(ctx, videoID)
				require.Error(t, err)
				mockStorage.AssertNotCalled(t, "DeleteFile")
				mockRepo.AssertNotCalled(t, "Delete")
				return
			}

			if tt.shouldDeleteStorage {
				if tt.status == models.StatusFailed {
					mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(fmt.Errorf("ignored"))
				} else {
					mockStorage.On("DeleteFile", ctx, video.ObjectName).Return(nil)
				}
			}

			mockRepo.On("Delete", ctx, videoID).Return(nil)

			err := service.DeleteVideo(ctx, videoID)
			require.NoError(t, err)

			mockRepo.AssertExpectations(t)
			if tt.shouldDeleteStorage {
				mockStorage.AssertExpectations(t)
			} else {
				mockStorage.AssertNotCalled(t, "DeleteFile")
			}
		})
	}
}

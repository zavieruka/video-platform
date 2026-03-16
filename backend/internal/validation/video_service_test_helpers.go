package validation

import (
	"fmt"
	"time"

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

func newPendingVideo(videoID string) *models.Video {
	return &models.Video{
		ID:                 videoID,
		Title:              "Test Video",
		Status:             models.StatusPending,
		ObjectName:         fmt.Sprintf("videos/%s.mp4", videoID),
		FileSize:           1024 * 1024 * 10,
		MimeType:           "video/mp4",
		UploadURLExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
}

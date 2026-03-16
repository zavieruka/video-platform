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
// GetVideo
// ============================================================================

func TestVideoService_GetVideo_Success(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "video-123"

	expectedVideo := &models.Video{
		ID:     videoID,
		Title:  "Test Video",
		Status: models.StatusUploaded,
	}

	mockRepo.On("GetByID", ctx, videoID).Return(expectedVideo, nil)

	result, err := service.GetVideo(ctx, videoID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, videoID, result.ID)
	assert.Equal(t, "Test Video", result.Title)

	mockRepo.AssertExpectations(t)
}

func TestVideoService_GetVideo_NotFound(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	videoID := "nonexistent"

	notFoundErr := errors.NewNotFoundError("Video", videoID)
	mockRepo.On("GetByID", ctx, videoID).Return(nil, notFoundErr)

	result, err := service.GetVideo(ctx, videoID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, notFoundErr, err)

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// ListVideos
// ============================================================================

func TestVideoService_ListVideos_Success(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	limit := 20
	offset := 0

	videos := []*models.Video{
		{ID: "video-1", Title: "Video 1", Status: models.StatusReady},
		{ID: "video-2", Title: "Video 2", Status: models.StatusUploaded},
	}
	totalCount := 50

	mockRepo.On("List", ctx, limit, offset).Return(videos, totalCount, nil)

	result, err := service.ListVideos(ctx, limit, offset)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Videos, 2)
	assert.Equal(t, totalCount, result.TotalCount)
	assert.Equal(t, limit, result.Limit)
	assert.Equal(t, offset, result.Offset)
	assert.Equal(t, "video-1", result.Videos[0].ID)
	assert.Equal(t, "video-2", result.Videos[1].ID)

	mockRepo.AssertExpectations(t)
}

func TestVideoService_ListVideos_EmptyResult(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	limit := 20
	offset := 0

	mockRepo.On("List", ctx, limit, offset).Return([]*models.Video{}, 0, nil)

	result, err := service.ListVideos(ctx, limit, offset)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Videos)
	assert.Equal(t, 0, result.TotalCount)

	mockRepo.AssertExpectations(t)
}

func TestVideoService_ListVideos_LimitClamping(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"negative limit defaults to 20", -10, 20},
		{"limit over 100 clamped to 100", 150, 100},
		{"valid limit unchanged", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockRepo, _, _, _ := newTestVideoService()

			ctx := context.Background()
			offset := 0

			mockRepo.On("List", ctx, tt.expectedLimit, offset).Return([]*models.Video{}, 0, nil)

			result, err := service.ListVideos(ctx, tt.inputLimit, offset)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLimit, result.Limit)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestVideoService_ListVideos_OffsetClamping(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	limit := 20
	inputOffset := -10
	expectedOffset := 0

	mockRepo.On("List", ctx, limit, expectedOffset).Return([]*models.Video{}, 0, nil)

	result, err := service.ListVideos(ctx, limit, inputOffset)

	require.NoError(t, err)
	assert.Equal(t, expectedOffset, result.Offset)

	mockRepo.AssertExpectations(t)
}

func TestVideoService_ListVideos_RepositoryError(t *testing.T) {
	service, mockRepo, _, _, _ := newTestVideoService()

	ctx := context.Background()
	limit := 20
	offset := 0

	dbError := fmt.Errorf("database query failed")
	mockRepo.On("List", ctx, limit, offset).Return(nil, 0, dbError)

	result, err := service.ListVideos(ctx, limit, offset)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, dbError)

	mockRepo.AssertExpectations(t)
}

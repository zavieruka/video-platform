package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

func TestVideo_ToResponse(t *testing.T) {
	// Use fixed timestamps for deterministic tests
	baseTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		video    *models.Video
		validate func(t *testing.T, response *models.VideoResponse)
	}{
		{
			name: "basic video without processing",
			video: &models.Video{
				ID:          "video-123",
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
				Status:      models.StatusUploaded,
				ObjectName:  "videos/video-123.mp4",
				StorageURL:  "gs://bucket/videos/video-123.mp4",
				PublicURL:   "https://storage.googleapis.com/bucket/videos/video-123.mp4",
				CreatedAt:   baseTime,
				UpdatedAt:   baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				assert.Equal(t, "video-123", response.ID)
				assert.Equal(t, "Test Video", response.Title)
				assert.Equal(t, "Test Description", response.Description)
				assert.Equal(t, "test.mp4", response.FileName)
				assert.Equal(t, int64(1024*1024*10), response.FileSize)
				assert.Equal(t, "video/mp4", response.MimeType)
				assert.Equal(t, models.StatusUploaded, response.Status)
				assert.Equal(t, "videos/video-123.mp4", response.ObjectName)
				assert.Equal(t, "gs://bucket/videos/video-123.mp4", response.StorageURL)
				assert.Equal(t, "https://storage.googleapis.com/bucket/videos/video-123.mp4", response.PublicURL)
				assert.Equal(t, baseTime, response.CreatedAt)
				assert.Equal(t, baseTime, response.UpdatedAt)
				assert.Nil(t, response.ProcessingStatus)
				assert.Nil(t, response.ProcessedVideos)
				assert.Nil(t, response.ManifestURL)
				assert.Nil(t, response.LastError)
			},
		},
		{
			name: "video with processing status - job started",
			video: &models.Video{
				ID:     "video-456",
				Title:  "Processing Video",
				Status: models.StatusProcessing,
				ProcessingJobID: func() *string {
					s := "job-123"
					return &s
				}(),
				ProcessingStartedAt: func() *time.Time {
					t := baseTime
					return &t
				}(),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				require.NotNil(t, response.ProcessingStatus)
				assert.Equal(t, "job-123", response.ProcessingStatus.JobID)
				require.NotNil(t, response.ProcessingStatus.StartedAt)
				assert.Equal(t, baseTime, *response.ProcessingStatus.StartedAt)
				assert.Nil(t, response.ProcessingStatus.EndedAt)
				assert.Equal(t, 0.0, response.ProcessingStatus.DurationSeconds)
			},
		},
		{
			name: "video with completed processing",
			video: &models.Video{
				ID:     "video-789",
				Title:  "Ready Video",
				Status: models.StatusReady,
				ProcessingJobID: func() *string {
					s := "job-456"
					return &s
				}(),
				ProcessingStartedAt: func() *time.Time {
					t := baseTime
					return &t
				}(),
				ProcessingEndedAt: func() *time.Time {
					t := baseTime.Add(5 * time.Minute).Add(30 * time.Second)
					return &t
				}(),
				ProcessedVideos: map[string]models.ProcessedVideo{
					"1080p": {
						Resolution: "1080p",
						PublicURL:  "https://storage.googleapis.com/bucket/video-789/1080p.m3u8",
						Bitrate:    5000000,
					},
					"720p": {
						Resolution: "720p",
						PublicURL:  "https://storage.googleapis.com/bucket/video-789/720p.m3u8",
						Bitrate:    2500000,
					},
				},
				ManifestURL: func() *string {
					s := "https://storage.googleapis.com/bucket/video-789/manifest.m3u8"
					return &s
				}(),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				require.NotNil(t, response.ProcessingStatus)
				require.NotNil(t, response.ProcessingStatus.EndedAt)
				assert.Equal(t, 330.0, response.ProcessingStatus.DurationSeconds) // 5:30 = 330 seconds
				require.NotNil(t, response.ProcessedVideos)
				assert.Len(t, response.ProcessedVideos, 2)

				// Verify both resolutions are present
				resolutions := make(map[string]bool)
				for _, pv := range response.ProcessedVideos {
					resolutions[pv.Resolution] = true
				}
				assert.True(t, resolutions["1080p"])
				assert.True(t, resolutions["720p"])

				require.NotNil(t, response.ManifestURL)
				assert.Equal(t, "https://storage.googleapis.com/bucket/video-789/manifest.m3u8", *response.ManifestURL)
			},
		},
		{
			name: "video with error",
			video: &models.Video{
				ID:     "video-error",
				Title:  "Failed Video",
				Status: models.StatusFailed,
				LastError: func() *string {
					s := "Transcoding failed: invalid format"
					return &s
				}(),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				assert.Equal(t, models.StatusFailed, response.Status)
				require.NotNil(t, response.LastError)
				assert.Contains(t, *response.LastError, "Transcoding failed")
			},
		},
		{
			name: "calculate processing duration - exact",
			video: &models.Video{
				ID:     "video-duration",
				Title:  "Duration Test",
				Status: models.StatusReady,
				ProcessingJobID: func() *string {
					s := "job-789"
					return &s
				}(),
				ProcessingStartedAt: func() *time.Time {
					t := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
					return &t
				}(),
				ProcessingEndedAt: func() *time.Time {
					t := time.Date(2026, 3, 5, 10, 5, 30, 0, time.UTC)
					return &t
				}(),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				require.NotNil(t, response.ProcessingStatus)
				assert.Equal(t, 330.0, response.ProcessingStatus.DurationSeconds) // 5 min 30 sec
			},
		},
		{
			name: "processing started but not ended - zero duration",
			video: &models.Video{
				ID:     "video-ongoing",
				Title:  "Ongoing Processing",
				Status: models.StatusProcessing,
				ProcessingJobID: func() *string {
					s := "job-999"
					return &s
				}(),
				ProcessingStartedAt: func() *time.Time {
					t := baseTime
					return &t
				}(),
				ProcessingEndedAt: nil, // Still processing
				CreatedAt:         baseTime,
				UpdatedAt:         baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				require.NotNil(t, response.ProcessingStatus)
				assert.Equal(t, 0.0, response.ProcessingStatus.DurationSeconds)
			},
		},
		{
			name: "no processing job ID - nil processing status",
			video: &models.Video{
				ID:                  "video-no-job",
				Title:               "No Job",
				Status:              models.StatusUploaded,
				ProcessingJobID:     nil,
				ProcessingStartedAt: nil,
				ProcessingEndedAt:   nil,
				CreatedAt:           baseTime,
				UpdatedAt:           baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				assert.Nil(t, response.ProcessingStatus, "Should be nil when no job ID")
			},
		},
		{
			name: "empty processed videos map",
			video: &models.Video{
				ID:              "video-empty-processed",
				Title:           "Empty Processed",
				Status:          models.StatusReady,
				ProcessedVideos: map[string]models.ProcessedVideo{}, // Empty but not nil
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
			},
			validate: func(t *testing.T, response *models.VideoResponse) {
				require.NotNil(t, response)
				assert.Nil(t, response.ProcessedVideos, "Empty map should convert to nil in response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.video.ToResponse()
			require.NotNil(t, response, "ToResponse should never return nil")
			tt.validate(t, response)
		})
	}
}

func TestVideoStatus_Values(t *testing.T) {
	// Test that status constants are properly defined
	tests := []struct {
		name     string
		status   models.VideoStatus
		expected string
	}{
		{"pending status", models.StatusPending, "pending"},
		{"uploaded status", models.StatusUploaded, "uploaded"},
		{"processing status", models.StatusProcessing, "processing"},
		{"ready status", models.StatusReady, "ready"},
		{"failed status", models.StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This validates the constant values are correct
			assert.Equal(t, models.VideoStatus(tt.expected), tt.status)

			// If VideoStatus has a String() method, test it
			// Otherwise this is validating the type itself
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

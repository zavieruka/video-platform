package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

func TestVideoValidator_ValidateUploadRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.UploadURLRequest
		maxSizeMB      int
		allowedFormats []string
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid request",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4", "mov", "avi"},
			expectError:    false,
		},
		{
			name: "case insensitive format check",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.MP4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4", "mov"},
			expectError:    false,
		},
		{
			name: "mov format",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mov",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/quicktime",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4", "mov"},
			expectError:    false,
		},
		{
			name: "long title should be valid",
			request: &models.UploadURLRequest{
				Title:       "This is a very long title that contains many characters but should still be valid as long as it's not empty",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    false,
		},
		{
			name: "empty description should be valid",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    false,
		},
		{
			name: "filename with multiple extensions",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "video.final.MP4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    false,
		},
		{
			name: "missing title",
			request: &models.UploadURLRequest{
				Title:       "",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "Title is required",
		},
		{
			name: "title with only whitespace",
			request: &models.UploadURLRequest{
				Title:       "   \t\n  ",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "Title is required",
		},
		{
			name: "missing filename",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "File name is required",
		},
		{
			name: "filename with only whitespace",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "   ",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "File name is required",
		},
		{
			name: "filename with no extension",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "videofile",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "must have an extension",
		},
		{
			name: "file too large",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 600,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "exceeds maximum",
		},
		{
			name: "zero file size",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    0,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "must be greater than 0",
		},
		{
			name: "negative file size",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    -100,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "must be greater than 0",
		},
		{
			name: "exact max size boundary - should pass",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 500,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    false,
		},
		{
			name: "max size plus one byte - should fail",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024*1024*500 + 1,
				MimeType:    "video/mp4",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4"},
			expectError:    true,
			errorContains:  "exceeds maximum",
		},
		{
			name: "unsupported format",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.exe",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "application/x-msdownload",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4", "mov"},
			expectError:    true,
			errorContains:  "extension not allowed",
		},
		{
			name: "MIME type and extension mismatch",
			request: &models.UploadURLRequest{
				Title:       "Test Video",
				Description: "Test Description",
				FileName:    "test.mp4",
				FileSize:    1024 * 1024 * 10,
				MimeType:    "video/quicktime",
			},
			maxSizeMB:      500,
			allowedFormats: []string{"mp4", "mov"},
			expectError:    true,
			errorContains:  "does not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewVideoValidator(tt.maxSizeMB, tt.allowedFormats)
			err := validator.ValidateUploadRequest(tt.request)

			if tt.expectError {
				require.NotNil(t, err, "Expected an error but got nil")
				if tt.errorContains != "" {
					assert.Contains(t, err.Message, tt.errorContains, "Error message should contain expected text")
				}
			} else {
				assert.Nil(t, err, "Expected nil but got: %v", err)
			}
		})
	}
}

func TestVideoValidator_ValidateUploadRequest_NilRequest(t *testing.T) {
	validator := NewVideoValidator(500, []string{"mp4"})

	err := validator.ValidateUploadRequest(nil)

	require.NotNil(t, err)
	assert.Contains(t, err.Message, "request is required")
}

func TestVideoValidator_EmptyAllowedFormats(t *testing.T) {
	validator := NewVideoValidator(500, []string{})

	req := &models.UploadURLRequest{
		Title:       "Test Video",
		Description: "Test Description",
		FileName:    "test.mp4",
		FileSize:    1024 * 1024 * 10,
		MimeType:    "video/mp4",
	}

	err := validator.ValidateUploadRequest(req)

	require.NotNil(t, err)
	assert.Contains(t, err.Message, "not allowed")
}

package validation

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zavieruka/video-platform/backend/internal/errors"
	"github.com/zavieruka/video-platform/backend/internal/models"
)

type VideoValidator struct {
	maxUploadSize     int64
	allowedMimeTypes  map[string]bool
	allowedExtensions map[string]bool
	mimeToExtensions  map[string][]string
}

func NewVideoValidator(maxUploadSizeMB int, allowedFormats []string) *VideoValidator {
	maxSizeBytes := int64(maxUploadSizeMB) * 1024 * 1024

	allowedMimeTypes := map[string]bool{
		"video/mp4":        true,
		"video/quicktime":  true, // .mov
		"video/x-msvideo":  true, // .avi
		"video/x-matroska": true, // .mkv
	}

	allowedExtensions := map[string]bool{
		".mp4": true,
		".mov": true,
		".avi": true,
		".mkv": true,
	}

	mimeToExtensions := map[string][]string{
		"video/mp4":        {".mp4"},
		"video/quicktime":  {".mov"},
		"video/x-msvideo":  {".avi"},
		"video/x-matroska": {".mkv"},
	}

	return &VideoValidator{
		maxUploadSize:     maxSizeBytes,
		allowedMimeTypes:  allowedMimeTypes,
		allowedExtensions: allowedExtensions,
		mimeToExtensions:  mimeToExtensions,
	}
}

func (v *VideoValidator) ValidateUploadRequest(req *models.UploadURLRequest) *errors.AppError {
	if err := v.ValidateTitle(req.Title); err != nil {
		return err
	}

	if err := v.ValidateFileName(req.FileName); err != nil {
		return err
	}

	if err := v.ValidateMimeType(req.MimeType); err != nil {
		return err
	}

	if err := v.ValidateFileSize(req.FileSize); err != nil {
		return err
	}

	if err := v.ValidateExtensionMatchesMimeType(req.FileName, req.MimeType); err != nil {
		return err
	}

	return nil
}

func (v *VideoValidator) ValidateTitle(title string) *errors.AppError {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.NewValidationError("Title is required", map[string]interface{}{
			"field": "title",
		})
	}

	if len(title) > 200 {
		return errors.NewValidationError("Title must be 200 characters or less", map[string]interface{}{
			"field":     "title",
			"maxLength": 200,
			"length":    len(title),
		})
	}

	return nil
}

func (v *VideoValidator) ValidateFileName(fileName string) *errors.AppError {
	if fileName == "" {
		return errors.NewValidationError("File name is required", map[string]interface{}{
			"field": "fileName",
		})
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return errors.NewValidationError("File name must have an extension", map[string]interface{}{
			"field":    "fileName",
			"fileName": fileName,
		})
	}

	if !v.allowedExtensions[ext] {
		allowedExts := []string{}
		for ext := range v.allowedExtensions {
			allowedExts = append(allowedExts, ext)
		}
		return errors.NewValidationError("File extension not allowed", map[string]interface{}{
			"field":             "fileName",
			"extension":         ext,
			"allowedExtensions": allowedExts,
		})
	}

	return nil
}

func (v *VideoValidator) ValidateMimeType(mimeType string) *errors.AppError {
	if mimeType == "" {
		return errors.NewValidationError("MIME type is required", map[string]interface{}{
			"field": "mimeType",
		})
	}

	if !v.allowedMimeTypes[mimeType] {
		allowedTypes := []string{}
		for mt := range v.allowedMimeTypes {
			allowedTypes = append(allowedTypes, mt)
		}
		return errors.NewValidationError("MIME type not allowed", map[string]interface{}{
			"field":            "mimeType",
			"mimeType":         mimeType,
			"allowedMimeTypes": allowedTypes,
		})
	}

	return nil
}

func (v *VideoValidator) ValidateFileSize(size int64) *errors.AppError {
	if size <= 0 {
		return errors.NewValidationError("File size must be greater than 0", map[string]interface{}{
			"field":    "fileSize",
			"fileSize": size,
		})
	}

	if size > v.maxUploadSize {
		return errors.NewValidationError(
			fmt.Sprintf("File size exceeds maximum allowed size of %d MB", v.maxUploadSize/(1024*1024)),
			map[string]interface{}{
				"field":        "fileSize",
				"maxSize":      v.maxUploadSize,
				"receivedSize": size,
			},
		)
	}

	return nil
}

func (v *VideoValidator) ValidateExtensionMatchesMimeType(fileName, mimeType string) *errors.AppError {
	ext := strings.ToLower(filepath.Ext(fileName))
	expectedExts, exists := v.mimeToExtensions[mimeType]

	if !exists {
		return errors.NewValidationError("Unknown MIME type", map[string]interface{}{
			"field":    "mimeType",
			"mimeType": mimeType,
		})
	}

	for _, expectedExt := range expectedExts {
		if ext == expectedExt {
			return nil
		}
	}

	return errors.NewValidationError("File extension does not match MIME type", map[string]interface{}{
		"field":              "fileName",
		"extension":          ext,
		"mimeType":           mimeType,
		"expectedExtensions": expectedExts,
	})
}

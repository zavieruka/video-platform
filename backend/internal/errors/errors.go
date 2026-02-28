package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string                 `json:"error"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       "validation_error",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

func NewNotFoundError(resource string, id string) *AppError {
	return &AppError{
		Code:    "not_found",
		Message: fmt.Sprintf("%s with ID %s not found", resource, id),
		Details: map[string]interface{}{
			"resource": resource,
			"id":       id,
		},
		StatusCode: http.StatusNotFound,
	}
}

func NewStorageError(message string, err error) *AppError {
	details := make(map[string]interface{})
	if err != nil {
		details["underlying_error"] = err.Error()
	}
	return &AppError{
		Code:       "storage_error",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewDatabaseError(message string, err error) *AppError {
	details := make(map[string]interface{})
	if err != nil {
		details["underlying_error"] = err.Error()
	}
	return &AppError{
		Code:       "database_error",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewInternalError(message string, err error) *AppError {
	details := make(map[string]interface{})
	if err != nil {
		details["underlying_error"] = err.Error()
	}
	return &AppError{
		Code:       "internal_error",
		Message:    message,
		Details:    details,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:       "conflict",
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:       "bad_request",
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func IsNotFound(err error) bool {
	if appErr, ok := errors.AsType[*AppError](err); ok {
		return appErr.Code == "not_found"
	}
	return false
}

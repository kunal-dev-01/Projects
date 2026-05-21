package errors

import (
	"errors"
	"fmt"
	"net/http"

	"shared/logging" 

	"github.com/gofiber/fiber/v2"
)

type AppError struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Err     error             `json:"-"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

/* ───────────────── Error Constructors ───────────────── */

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(err error, code int, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func BadRequest(message string) *AppError { return New(http.StatusBadRequest, message) }
func Unauthorized(message string) *AppError { return New(http.StatusUnauthorized, message) }
func Forbidden(message string) *AppError    { return New(http.StatusForbidden, message) }
func NotFound(message string) *AppError     { return New(http.StatusNotFound, message) }
func Conflict(message string) *AppError     { return New(http.StatusConflict, message) }
func Unprocessable(message string) *AppError {
	return New(http.StatusUnprocessableEntity, message)
}
func TooManyRequests(message string) *AppError { return New(http.StatusTooManyRequests, message) }

func Validation(message string, fields map[string]string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Fields:  fields,
	}
}

func Internal(err error) *AppError {
	return Wrap(err, http.StatusInternalServerError, "something went wrong, please try again later")
}

func IsConflict(err error) bool {
	if e, ok := err.(*AppError); ok {
		return e.Code == http.StatusConflict
	}
	return false
}


/* ───────────────── Centralized Fiber Middleware ───────────────── */

func FiberErrorHandler(c *fiber.Ctx, err error) error {
	code, message, fields := parseError(err)

	// Log metrics/errors strictly based on domain statuses
	log := logging.Get().WithFiberCtx(c)
	var appErr *AppError

	if errors.As(err, &appErr) {
		if code >= 500 {
			log.Error().Err(appErr.Err).Msg("internal app error")
		} else {
			log.Warn().Str("error", appErr.Message).Msg("client request error")
		}
	} else {
		log.Error().Err(err).Msg("unhandled server error")
	}

	// Prepare standard API contract structure
	errBody := fiber.Map{
		"code":    code,
		"message": message,
	}
	if len(fields) > 0 {
		errBody["fields"] = fields
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   errBody,
	})
}

func parseError(err error) (code int, message string, fields map[string]string) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code, appErr.Message, appErr.Fields
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code, fiberErr.Message, nil
	}

	return http.StatusInternalServerError, "something went wrong", nil
}
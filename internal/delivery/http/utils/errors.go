package utils

import (
	"errors"
	"log/slog"
	"net/http"

	"foro-unsaac-backend/internal/domain"

	"github.com/gin-gonic/gin"
)

// handleError maps domain errors to HTTP responses
// Follows the error flow: domain error → HTTP status code + JSON response
func HandleError(c *gin.Context, err error, logger *slog.Logger) {

	// Check if error is a domain.AppError
	if appErr, ok := errors.AsType[domain.AppError](err); ok {
		response := gin.H{"error": appErr.Message}

		// Include detail for client errors (not 500+) if available
		if appErr.Code < 500 && appErr.Detail != "" {
			response["detail"] = appErr.Detail
		}

		c.JSON(appErr.Code, response)
		return
	}

	// Log unhandled errors (server errors)
	logger.ErrorContext(c.Request.Context(), "unhandled error", "error", err)

	// Generic server error response
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}

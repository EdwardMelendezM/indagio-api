package utils

import (
	"errors"
	"log/slog"
	"net/http"

	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
)

func HandleError(c *gin.Context, err error, logger *slog.Logger) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		response := gin.H{"error": appErr.Message}
		if appErr.Code < 500 && appErr.Detail != "" {
			response["detail"] = appErr.Detail
		}
		c.JSON(appErr.Code, response)
		return
	}

	logger.ErrorContext(c.Request.Context(), "unhandled error", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

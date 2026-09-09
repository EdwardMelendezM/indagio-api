package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Data interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

func Success(c *gin.Context, statusCode int, data interface{}) {
	if statusCode == http.StatusNoContent {
		c.Status(statusCode)
		return
	}

	c.JSON(statusCode, SuccessResponse{
		Data: data,
	})
}

func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error: message,
	})
}

func NotFound(c *gin.Context) {
	Error(c, http.StatusNotFound, "Not Found")
}

func Paginated(c *gin.Context, statusCode int, data interface{}, total, page, limit int) {
	c.JSON(statusCode, PaginatedResponse{
		Data:  data,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

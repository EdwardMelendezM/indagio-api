package utils

import (
	"fmt"
	"foro-unsaac-backend/internal/config"
	"net/http"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// LoadSwagger configures and registers Swagger endpoints
func LoadSwagger(router *gin.Engine, cfg *config.AppConfig) {
	// Configure Swagger metadata
	host := fmt.Sprintf("localhost:%s", cfg.Port)

	// Swagger UI route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Swagger JSON endpoint
	router.GET("/docs/swagger.json", func(c *gin.Context) {
		swaggerJSON := map[string]interface{}{
			"swagger": "2.0",
			"info": map[string]interface{}{
				"title":       "Hilos API",
				"version":     "1.0",
				"description": "University Forum Backend API with Clean Architecture",
				"contact": map[string]interface{}{
					"name": "Backend Team",
					"url":  "https://github.com/unsaac/foro-unsaac-api",
				},
				"license": map[string]interface{}{
					"name": "MIT",
					"url":  "https://opensource.org/licenses/MIT",
				},
			},
			"host":     host,
			"basePath": "/",
			"schemes":  []string{"http", "https"},
			"securityDefinitions": map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":        "apiKey",
					"name":        "Authorization",
					"in":          "header",
					"description": "Type \"Bearer\" followed by space and JWT token.",
				},
			},
		}
		c.JSON(http.StatusOK, swaggerJSON)
	})

	logger.Info("swagger documentation available", "url", fmt.Sprintf("http://%s/swagger/index.html", host))
}

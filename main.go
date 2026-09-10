// Package main provides the entry point for Indagio Backend API
//
// @title			Indagio API
// @version			1.0
// @description		API with Clean Architecture
// @host			localhost:8080
// @basePath			/
// @schemes			http https
//
// @securityDefinitions.apikey BearerAuth
// @in				header
// @name			Authorization
// @description		Type "Bearer" followed by space and JWT token.
//
// @contact.name	Backend Team
// @contact.email	backend@indagio.com
// @contact.url		https://github.com/unsaac/indagio-api
//
// @license.name	MIT
// @license.url		https://opensource.org/licenses/MIT
package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"indagio-api/internal/config"
	adminhttp "indagio-api/internal/delivery/http/admin_user"
	answerhttp "indagio-api/internal/delivery/http/answers"
	authhttp "indagio-api/internal/delivery/http/auth"
	exporthttp "indagio-api/internal/delivery/http/exports"
	instrumenthttp "indagio-api/internal/delivery/http/instruments"
	mediahttp "indagio-api/internal/delivery/http/media"
	participanthttp "indagio-api/internal/delivery/http/participants"
	projecthttp "indagio-api/internal/delivery/http/projects"
	synchttp "indagio-api/internal/delivery/http/sync"
	userhttp "indagio-api/internal/delivery/http/user"
	"indagio-api/internal/delivery/worker"
	"indagio-api/internal/docs"
	adminrepo "indagio-api/internal/repository/admin_user/postgres"
	answerrepo "indagio-api/internal/repository/answers/postgres"
	exportrepo "indagio-api/internal/repository/exports/postgres"
	instrumentrepo "indagio-api/internal/repository/instruments/postgres"
	jobrepo "indagio-api/internal/repository/jobs/postgres"
	otprepo "indagio-api/internal/repository/otp/postgres"
	participantrepo "indagio-api/internal/repository/participants/postgres"
	projectrepo "indagio-api/internal/repository/projects/postgres"
	storage "indagio-api/internal/repository/storage/cloudflare"
	syncrepo "indagio-api/internal/repository/sync/postgres"
	userrepo "indagio-api/internal/repository/users/postgres"
	adminuc "indagio-api/internal/usecase/admin_user"
	answeruc "indagio-api/internal/usecase/answers"
	authuc "indagio-api/internal/usecase/auth"
	exportuc "indagio-api/internal/usecase/exports"
	instrumentuc "indagio-api/internal/usecase/instruments"
	participantuc "indagio-api/internal/usecase/participants"
	projectuc "indagio-api/internal/usecase/projects"
	syncuc "indagio-api/internal/usecase/sync"
	useruc "indagio-api/internal/usecase/user"
	"indagio-api/internal/utils"
	"indagio-api/middleware"
)

func main() {
	_ = docs.SwaggerInfo
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(context.Background()); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(1)
	}

	emailSvc, err := utils.NewEmailService(cfg.Email)
	if err != nil {
		logger.Error("build email service", "error", err)
		os.Exit(1)
	}
	passwordSvc := utils.NewPasswordService()
	tokenSvc := utils.NewTokenService(cfg.Auth)
	adminSvc := utils.NewAdminService(db)

	userRepo := userrepo.NewUserRepository(db)
	otpRepo := otprepo.NewOTPRepository(db)
	jobRepo := jobrepo.NewPostgresJobRepository(db)
	adminRepo := adminrepo.NewAdminUserRepository(db)
	projectRepo := projectrepo.NewProjectRepository(db)
	participantRepo := participantrepo.NewParticipantRepository(db)
	instrumentRepo := instrumentrepo.NewInstrumentRepository(db)
	answerRepo := answerrepo.NewAnswerRepository(db)
	syncRepo := syncrepo.NewSyncRepository(db)
	exportRepo := exportrepo.NewExportRepository(db)
	storageRepo, err := storage.NewR2Repository(cfg.Storage)
	if err != nil {
		logger.Error("build storage repository", "error", err)
		os.Exit(1)
	}

	authUC := authuc.NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
	userUC := useruc.NewUserUsecase(userRepo)
	projectUC := projectuc.NewProjectUsecase(projectRepo)
	participantUC := participantuc.NewParticipantUsecase(participantRepo)
	instrumentUC := instrumentuc.NewInstrumentUsecase(instrumentRepo)
	answerUC := answeruc.NewAnswerUsecase(answerRepo)
	syncUC := syncuc.NewSyncUsecase(syncRepo)
	exportUC := exportuc.NewExportUsecase(exportRepo)
	adminUC := adminuc.NewAdminUsecase(adminRepo, userRepo, otpRepo, tokenSvc, emailSvc)

	authHandler := authhttp.NewAuthHandler(authUC, logger)
	userHandler := userhttp.NewUserHandler(authUC, userUC, logger)
	projectHandler := projecthttp.NewProjectHandler(projectUC, logger)
	participantHandler := participanthttp.NewParticipantHandler(participantUC, logger)
	instrumentHandler := instrumenthttp.NewInstrumentHandler(instrumentUC, logger)
	answerHandler := answerhttp.NewAnswerHandler(answerUC, logger)
	mediaHandler := mediahttp.NewMediaHandler(storageRepo, logger, cfg.Storage.PublicDomain, int64(cfg.Storage.VideoMaxSizeMB)*1024*1024, cfg.Storage.VideoMaxDurationS)
	syncHandler := synchttp.NewSyncHandler(syncUC)
	exportHandler := exporthttp.NewExportHandler(exportUC)
	adminHandler := adminhttp.NewAdminHandler(adminUC, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(os.Stdout))

	api := r.Group("/api")
	authMiddleware := middleware.AuthMiddleware(tokenSvc)
	adminAuthMiddleware := middleware.AdminAuthMiddleware(tokenSvc, adminSvc)

	authhttp.RegisterAuthRoutes(api, authHandler, authMiddleware)
	userhttp.RegisterUserRoutes(api, userHandler, authMiddleware)
	projecthttp.RegisterProjectRoutes(api, projectHandler, authMiddleware)
	participanthttp.RegisterParticipantRoutes(api, participantHandler, authMiddleware)
	instrumenthttp.RegisterInstrumentRoutes(api, instrumentHandler, authMiddleware)
	answerhttp.RegisterAnswerRoutes(api, answerHandler, authMiddleware)
	mediahttp.RegisterMediaRoutes(api, mediaHandler, authMiddleware)
	synchttp.RegisterSyncRoutes(api, syncHandler)
	exporthttp.RegisterExportRoutes(api, exportHandler, authMiddleware)
	adminhttp.RegisterAdminRoutes(api, adminHandler, adminAuthMiddleware)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	jobWorker := worker.NewJobWorker(jobRepo, emailSvc, logger)
	jobCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	go jobWorker.Run(jobCtx)

	srv := &http.Server{
		Addr:           ":" + cfg.App.Port,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

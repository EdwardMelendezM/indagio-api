// Package main provides the entry point for Hilos Backend API
//
// @title			Hilos API
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
// @contact.url		https://github.com/unsaac/foro-unsaac-api
//
// @license.name	MIT
// @license.url		https://opensource.org/licenses/MIT
package main

import (
	"context"

	"foro-unsaac-backend/internal/delivery/http/report"
	"foro-unsaac-backend/internal/delivery/http/scholarships"

	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"foro-unsaac-backend/db"
	"foro-unsaac-backend/internal/config"
	_ "foro-unsaac-backend/internal/docs"
	"foro-unsaac-backend/internal/domain"
	"foro-unsaac-backend/internal/utils"
	"foro-unsaac-backend/middleware"

	"foro-unsaac-backend/internal/delivery/http/admin_user"
	"foro-unsaac-backend/internal/delivery/http/allowed_domain"
	"foro-unsaac-backend/internal/delivery/http/auth"
	"foro-unsaac-backend/internal/delivery/http/avatar"
	"foro-unsaac-backend/internal/delivery/http/avatar_border"
	"foro-unsaac-backend/internal/delivery/http/category"
	"foro-unsaac-backend/internal/delivery/http/chat_mvp"
	"foro-unsaac-backend/internal/delivery/http/comment"
	"foro-unsaac-backend/internal/delivery/http/conversation"
	"foro-unsaac-backend/internal/delivery/http/dashboard"
	"foro-unsaac-backend/internal/delivery/http/featured"
	"foro-unsaac-backend/internal/delivery/http/media"
	"foro-unsaac-backend/internal/delivery/http/message"
	"foro-unsaac-backend/internal/delivery/http/moderation"
	"foro-unsaac-backend/internal/delivery/http/notification"
	"foro-unsaac-backend/internal/delivery/http/push"
	"foro-unsaac-backend/internal/delivery/http/reaction"
	"foro-unsaac-backend/internal/delivery/http/reshare"
	"foro-unsaac-backend/internal/delivery/http/sticker"
	"foro-unsaac-backend/internal/delivery/http/subscription"
	"foro-unsaac-backend/internal/delivery/http/thread"
	"foro-unsaac-backend/internal/delivery/http/user"

	"foro-unsaac-backend/internal/delivery/worker"
	"foro-unsaac-backend/internal/delivery/ws"
	chatmvpws "foro-unsaac-backend/internal/delivery/ws/chat_mvp"

	adminUser2 "foro-unsaac-backend/internal/usecase/admin_user"
	alloweddomain2 "foro-unsaac-backend/internal/usecase/allowed_domain"
	auth2 "foro-unsaac-backend/internal/usecase/auth"
	avatar2 "foro-unsaac-backend/internal/usecase/avatar"
	avatarborder2 "foro-unsaac-backend/internal/usecase/avatar_border"
	category2 "foro-unsaac-backend/internal/usecase/category"
	comment2 "foro-unsaac-backend/internal/usecase/comment"
	conversation2 "foro-unsaac-backend/internal/usecase/conversation"
	dashboard2 "foro-unsaac-backend/internal/usecase/dashboard"
	featured2 "foro-unsaac-backend/internal/usecase/featured"
	message2 "foro-unsaac-backend/internal/usecase/message"
	moderation2 "foro-unsaac-backend/internal/usecase/moderation"
	moderationprefilter2 "foro-unsaac-backend/internal/usecase/moderation_prefilter"
	notification2 "foro-unsaac-backend/internal/usecase/notification"
	presence2 "foro-unsaac-backend/internal/usecase/presence"
	push2 "foro-unsaac-backend/internal/usecase/push"
	reaction2 "foro-unsaac-backend/internal/usecase/reaction"
	reshare2 "foro-unsaac-backend/internal/usecase/reshare"
	scholarships2 "foro-unsaac-backend/internal/usecase/scholarships"
	sticker2 "foro-unsaac-backend/internal/usecase/sticker"
	thread2 "foro-unsaac-backend/internal/usecase/thread"
	user2 "foro-unsaac-backend/internal/usecase/user"

	postgresAdminUser "foro-unsaac-backend/internal/repository/admin_user/postgres"
	allowedDomainsPostgres "foro-unsaac-backend/internal/repository/allowed_domains/postgres"
	postgresAvatarBorders "foro-unsaac-backend/internal/repository/avatar_borders/postgres"
	postgrescategories "foro-unsaac-backend/internal/repository/categories/postgres"
	postgrescomments "foro-unsaac-backend/internal/repository/comments/postgres"
	postgresConversations "foro-unsaac-backend/internal/repository/conversations/postgres"
	dashboardpostgres "foro-unsaac-backend/internal/repository/dashboard/postgres"
	postgresFeatured "foro-unsaac-backend/internal/repository/featured/postgres"
	postgresJobs "foro-unsaac-backend/internal/repository/jobs/postgres"
	postgresMessages "foro-unsaac-backend/internal/repository/messages/postgres"
	moderationpostgres "foro-unsaac-backend/internal/repository/moderation/postgres"
	postgresNotifications "foro-unsaac-backend/internal/repository/notifications/postgres"
	postgresotp "foro-unsaac-backend/internal/repository/otp/postgres"
	postgrespresence "foro-unsaac-backend/internal/repository/presence/postgres"
	postgresPushToken "foro-unsaac-backend/internal/repository/push_tokens/postgres"
	postgresreactions "foro-unsaac-backend/internal/repository/reactions/postgres"
	postgresreshares "foro-unsaac-backend/internal/repository/reshares/postgres"
	postgresScholarships "foro-unsaac-backend/internal/repository/scholarships/postgres"
	postgresstickers "foro-unsaac-backend/internal/repository/stickers/postgres"
	cloudflareRepo "foro-unsaac-backend/internal/repository/storage/cloudflare"
	postgressubscriptions "foro-unsaac-backend/internal/repository/subscriptions/postgres"
	postgresthreads "foro-unsaac-backend/internal/repository/threads/postgres"
	postgresusers "foro-unsaac-backend/internal/repository/users/postgres"
)

func main() {
	// Load .env in development — ignored if file is missing (production uses real env vars)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	loggerInstance := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// ─────────────────────────────────────
	// CONFIG — single source of truth
	// ─────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		loggerInstance.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// ─────────────────────────────────────
	// DATABASE
	// ─────────────────────────────────────
	database, err := db.Connect(cfg.Database)
	if err != nil {
		loggerInstance.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// ─────────────────────────────────────
	// REPOSITORY LAYER
	// ─────────────────────────────────────
	r2Repo, err := cloudflareRepo.NewR2Repository(cfg.Storage)
	userRepo := postgresusers.NewUserRepository(database)
	otpRepo := postgresotp.NewOTPRepository(database)
	categoryRepo := postgrescategories.NewCategoryRepository(database)
	threadRepo := postgresthreads.NewThreadRepository(database)
	commentRepo := postgrescomments.NewCommentRepository(database)
	featuredRepo := postgresFeatured.NewFeaturedRepository(database)
	reactionRepo := postgresreactions.NewReactionRepository(database)
	notificationRepo := postgresNotifications.NewNotificationRepository(database, userRepo)
	adminUserRepo := postgresAdminUser.NewAdminUserRepository(database)
	pushTokenRepo := postgresPushToken.NewPushTokenRepository(database)
	jobRepo := postgresJobs.NewPostgresJobRepository(database)
	subscriptionRepo := postgressubscriptions.NewSubscriptionRepository(database)
	allowedDomainRepo := allowedDomainsPostgres.NewAllowedDomainRepository(database)
	conversationRepo := postgresConversations.NewConversationRepository(database)
	messageRepo := postgresMessages.NewMessageRepository(database)
	presenceRepo := postgrespresence.NewPresenceRepository(database)
	dashboardRepo := dashboardpostgres.NewDashboardRepository(database)
	reshareRepo := postgresreshares.NewReshareRepository(database)
	scholarshipRepo := postgresScholarships.NewScholarshipRepository(database)
	// Sticker repos (PR 2/3). The usecase / handler layers arrive in
	// PR 4; we keep the repo around so the worker can call it for the
	// recovery job.
	stickerRepo := postgresstickers.NewStickerRepository(database)

	// ─────────────────────────────────────
	// UTILITY SERVICES
	// ─────────────────────────────────────
	hub := notification.NewNotificationHub()
	passwordSvc := utils.NewPasswordService()
	tokenSvc := utils.NewTokenService(cfg.Auth)
	emailSvc, err := utils.NewEmailService(cfg.Email)
	adminSvc := utils.NewAdminService(database)

	// Push notification provider (no-op if not configured)
	var pushProvider domain.PushProvider
	if cfg.Push.ExpoProjectID != "" {
		pushProvider = utils.NewExpoPushProvider(cfg.Push.ExpoAccessToken, cfg.Push.ExpoProjectID)
		loggerInstance.Info("push notifications enabled", "project_id", cfg.Push.ExpoProjectID)
	} else {
		pushProvider = &utils.NoOpPushProvider{}
		loggerInstance.Warn("EXPO_PROJECT_ID not set — push notifications disabled")
	}

	pushUC := push2.NewPushNotificationUsecase(pushTokenRepo, pushProvider, jobRepo, subscriptionRepo, loggerInstance)
	if err != nil {
		loggerInstance.Error("create email service", "error", err)
	}

	// ─────────────────────────────────────
	// USE CASE LAYER
	// ─────────────────────────────────────
	videoQueue := worker.NewVideoQueue(500)

	authUC := auth2.NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo, allowedDomainRepo)
	allowedDomainUC := alloweddomain2.NewAllowedDomainUsecase(allowedDomainRepo)
	categoryUC := category2.NewCategoryUsecase(categoryRepo)

	// Moderation pre-filter (synchronous content check before DB write)
	preFilter := moderationprefilter2.NewModerationPreFilter(nil) // nil blocked domains — uses hardcoded list

	// Moderation components (async worker + repository) — must be created before threadUC/commentUC
	moderationRepo := moderationpostgres.NewModerationRepository(database)
	modQueue := worker.NewModerationJobQueue()
	moderationUC := moderation2.NewModerationUsecase(
		moderationRepo,
		threadRepo,
		commentRepo,
		modQueue,
		preFilter,
		notificationRepo,
		adminUserRepo,
		userRepo,
		loggerInstance,
	)

	threadUC := thread2.NewThreadUsecase(
		threadRepo,
		categoryRepo,
		userRepo,
		adminUserRepo,
		r2Repo,
		notificationRepo,
		videoQueue,
		pushUC,
		cfg.Storage.PublicDomain,
		preFilter,
		moderationUC,
	)
	// WebSocket presence tracker (in-memory) - created before notificationUC since it gates push
	presenceTracker := ws.NewPresenceTracker()

	notificationUC := notification2.NewNotificationUsecase(
		notificationRepo,
		threadRepo,
		commentRepo,
		hub,
		pushUC,
	)
	reactionUC := reaction2.NewReactionUsecase(reactionRepo, userRepo, commentRepo, notificationUC)
	reshareUC := reshare2.NewReshareUsecase(reshareRepo, threadRepo, userRepo, notificationUC, preFilter, moderationUC, loggerInstance)
	avatarUC := avatar2.NewAvatarUsecase(userRepo, r2Repo, preFilter, moderationUC, loggerInstance)
	avatarBorderRepo := postgresAvatarBorders.NewAvatarBorderRepository(database)
	avatarBorderUC := avatarborder2.NewAvatarBorderUsecase(avatarBorderRepo, userRepo, loggerInstance)
	adminUserUC := adminUser2.NewAdminUsecase(
		adminUserRepo,
		userRepo,
		otpRepo,
		tokenSvc,
		emailSvc,
	)
	conversationUC := conversation2.NewConversationUsecase(conversationRepo, userRepo, messageRepo)
	userUC := user2.NewUserUsecase(userRepo)

	// Chat Hub (will be wired as Broadcaster)
	chatHub := ws.NewHub(conversationRepo, nil, presenceTracker, loggerInstance)

	// Presence usecase for chat: persists is_online + broadcasts presence_update
	// frames to conversation participants. Depends on chatHub as its broadcaster,
	// so it's wired via SetPresenceUsecase after construction.
	presenceUC := presence2.NewPresenceUsecase(presenceRepo, conversationRepo, chatHub, loggerInstance)
	chatHub.SetPresenceUsecase(presenceUC)

	dashboardUC := dashboard2.NewDashboardUsecase(dashboardRepo, chatHub, loggerInstance)

	// Sticker queue (PR 3) and usecase (PR 4). The queue is constructed
	// first so the usecase can accept it; the usecase is constructed
	// before messageUC so messageUC can use its resolver.
	stickerQueue := worker.NewStickerQueue(500)
	stickerUC := sticker2.NewStickerUsecase(
		stickerRepo,
		r2Repo,
		preFilter,
		moderationUC,
		nil, /* msgUC — wired via back-pointer below */
		userRepo,
		stickerQueue,
		sticker2.DefaultStickerLimits(),
		loggerInstance,
	)

	messageUC := message2.NewMessageUsecase(messageRepo, conversationRepo, chatHub, pushUC, chatHub.GetPresenceTracker(), userRepo, stickerUC, loggerInstance)

	// Back-pointer: the sticker usecase needs the message usecase for
	// SendStickerMessage. We replace the nil with the real messageUC.
	// (Go closures don't apply here; the usecase is a struct, so we
	// re-construct it with the back-pointer. Cheap because the
	// constructor doesn't allocate large state.)
	stickerUC = sticker2.NewStickerUsecase(
		stickerRepo,
		r2Repo,
		preFilter,
		moderationUC,
		messageUC,
		userRepo,
		stickerQueue,
		sticker2.DefaultStickerLimits(),
		loggerInstance,
	)

	// Comment usecase is constructed here (after stickerUC) because it
	// depends on StickerResolver for the sticker-in-comments feature.
	commentUC := comment2.NewCommentUsecase(commentRepo, threadRepo, userRepo, notificationUC, preFilter, moderationUC, stickerUC, stickerRepo)
	featuredUC := featured2.NewFeaturedUsecaseWithChecker(featuredRepo, threadRepo, nil)

	scholarshipUC := scholarships2.NewScholarshipUsecase(
		scholarshipRepo,
		adminUserRepo,
		userRepo,
		pushUC,
		loggerInstance,
	)

	// Shared context for background workers (cancel on shutdown)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start moderation worker pool (2 goroutines) if OpenAI API key is available
	if cfg.App.OpenAIAPIKey != "" {
		openAIAnalyzer := utils.NewOpenAIModerationAnalyzer(cfg.App.OpenAIAPIKey, r2Repo)
		modWorker := worker.NewModerationWorker(
			modQueue,
			moderationUC,
			openAIAnalyzer,
			moderationRepo,
			threadRepo,
			commentRepo,
			stickerRepo, // PR 6: for sticker moderation actions
			r2Repo,      // PR 6: for sticker R2 cleanup (best-effort)
			notificationRepo,
			adminUserRepo,
			userRepo,
			loggerInstance,
		)
		go modWorker.Start(ctx)
	}

	// ─────────────────────────────────────
	// DELIVERY LAYER
	// ─────────────────────────────────────
	mediaHandler := media.NewMediaHandler(
		r2Repo,
		loggerInstance,
		cfg.Storage.PublicDomain,
		int64(cfg.Storage.VideoMaxSizeMB)*1024*1024,
		cfg.Storage.VideoMaxDurationS,
	)
	authHandler := auth.NewAuthHandler(authUC, loggerInstance)
	categoryHandler := category.NewCategoryHandler(categoryUC, loggerInstance)
	threadHandler := thread.NewThreadHandlerWithFeatured(threadUC, featuredUC, loggerInstance)
	commentHandler := comment.NewCommentHandler(commentUC, loggerInstance)
	featuredHandler := featured.NewFeaturedHandler(featuredUC, loggerInstance)
	reactionHandler := reaction.NewReactionHandler(reactionUC, loggerInstance)
	reshareHandler := reshare.NewReshareHandler(reshareUC, loggerInstance)
	avatarHandler := avatar.NewAvatarHandler(avatarUC, loggerInstance)
	avatarBorderHandler := avatar_border.NewAvatarBorderHandler(avatarBorderUC, loggerInstance)
	notificationHandler := notification.NewNotificationHandler(notificationUC, hub, loggerInstance)
	adminUserHandler := admin_user.NewAdminHandler(adminUserUC, loggerInstance)
	pushHandler := push.NewPushHandler(pushUC)
	userHandler := user.NewUserHandler(authUC, userUC, loggerInstance)
	subscriptionHandler := subscription.NewSubscriptionHandler(pushUC, loggerInstance)
	allowedDomainHandler := allowed_domain.NewAllowedDomainHandler(allowedDomainUC)
	moderationHandler := moderation.NewModerationHandler(moderationUC, adminUserUC, userUC)
	reportHandler := report.NewReportHandler(moderationUC)
	dashboardHandler := dashboard.NewDashboardHandler(dashboardUC)
	conversationHandler := conversation.NewConversationHandler(conversationUC, userRepo, loggerInstance)
	messageHandler := message.NewMessageHandler(messageUC, userRepo, loggerInstance)
	scholarshipHandler := scholarships.NewScholarshipHandler(scholarshipUC, loggerInstance)

	// Chat-mvp handler — serves the /api/messages/* namespace with the
	// frozen wire contract from docs/47 §2.1. Thin adapter over the
	// existing conversation/message/push usecases.
	chatMvpHandler := chat_mvp.NewChatMvpHandler(conversationUC, messageUC, pushUC, userRepo, presenceTracker, loggerInstance)

	// Sticker handler (PR 5) + presign handler. The presign handler
	// needs a callback to fetch the caller's current quota — we
	// implement it as a closure that uses the sticker repo.
	stickerHandler := sticker.NewStickerHandler(stickerUC, loggerInstance)
	limits := sticker2.DefaultStickerLimits()
	stickerPresignHandler := media.NewStickerPresignHandler(
		r2Repo,
		loggerInstance,
		func(userID uuid.UUID) (stickers, packs int, err error) {
			ctx := context.Background()
			n, err := stickerRepo.CountStickersByUser(ctx, userID)
			if err != nil {
				return 0, 0, err
			}
			p, err := stickerRepo.CountPacksByUser(ctx, userID)
			if err != nil {
				return 0, 0, err
			}
			return n, p, nil
		},
		limits.MaxPerUser,
		limits.MaxPacksPerUser,
	)

	// Border presign: separate from MediaHandler because the route is
	// admin-only and the key prefix is global (borders/<uuid>.<ext>)
	// rather than per-user. See media/border_presign.go.
	borderPresignHandler := media.NewBorderPresignHandler(media.BorderPresignDeps{
		Storage:      r2Repo,
		Logger:       loggerInstance,
		PublicDomain: cfg.Storage.PublicDomain,
	})

	// WebSocket handler for chat
	wsRouter := ws.NewRouter(chatHub, messageUC, conversationRepo, loggerInstance)
	wsHandler := ws.NewWSHandler(chatHub, wsRouter, tokenSvc, loggerInstance)

	// Shared rate limiter — same in-process budget for legacy /ws/chat
	// AND chat-mvp /api/messages/ws. Both routers reference this
	// instance so a user hammering one surface can't bypass the
	// limit by switching to the other.
	chatRateLimiter := ws.NewRateLimiter(ws.DefaultRateLimitWindow, ws.DefaultRateLimitMax)
	wsRouter.SetRateLimiter(chatRateLimiter)

	// Chat-mvp WS surface (/api/messages/ws) — subprotocol JWT,
	// distinct frame types, dedicated router. Shares the same hub
	// (so presence/typing fan-out works) but uses its own Dispatcher.
	chatMvpWsRouter := chatmvpws.NewRouter(chatHub, messageUC, conversationRepo, presenceTracker, chatRateLimiter, loggerInstance)
	chatMvpWsHandler := chatmvpws.NewWSHandler(chatHub, chatMvpWsRouter, tokenSvc, loggerInstance)

	// ─────────────────────────────────────
	// BACKGROUND WORKER
	// ─────────────────────────────────────
	hotScoreJob := worker.NewHotScoreJob(threadUC)
	worker.NewJobRunner(hotScoreJob, 5*time.Minute, 30*time.Second, loggerInstance).Start(ctx)

	jobWorker := worker.NewJobWorker(jobRepo, pushTokenRepo, pushProvider, emailSvc, loggerInstance)
	go jobWorker.Run(ctx)

	// Video processing workers (2 parallel workers over the in-memory queue)
	const videoWorkerCount = 2
	for i := 1; i <= videoWorkerCount; i++ {
		vw := worker.NewVideoWorker(i, videoQueue, threadRepo, r2Repo, cfg.Storage.VideoMaxDurationS)
		go vw.Run(ctx)
	}

	// Video recovery job: re-enqueues videos stranded in PROCESANDO for >2 minutes
	// (grace period enforced by GetProcessingVideos SQL). Catches missed jobs after
	// server restart or worker crashes.
	videoRecoveryJob := worker.NewVideoRecoveryJob(threadRepo, videoQueue, loggerInstance)
	worker.NewJobRunner(videoRecoveryJob, 5*time.Minute, 30*time.Second, loggerInstance).Start(ctx)

	// ─────────────────────────────────────
	// Sticker worker (PR 3 + 4)
	// ─────────────────────────────────────
	// The in-memory sticker queue is the channel between the HTTP commit
	// handler (PR 5) and the worker. The worker hands the job to the
	// StickerUsecase which does the actual download → resize → upload →
	// moderate pipeline.
	// (stickerQueue is constructed above, before stickerUC, so the
	// usecase can accept it.)
	{
		const stickerWorkerCount = 2
		for i := 1; i <= stickerWorkerCount; i++ {
			sw := worker.NewStickerWorker(i, stickerQueue, stickerUC, loggerInstance)
			go sw.Run(ctx)
		}
		stickerRecoveryJob := worker.NewStickerRecoveryJob(stickerRepo, stickerQueue, loggerInstance)
		worker.NewJobRunner(stickerRecoveryJob, 5*time.Minute, 30*time.Second, loggerInstance).Start(ctx)
	}

	// Scholarship deadline reminder worker (Phase 3) — runs once a day,
	// queries scholarships whose apply_deadline falls on a 7/3/1 day-mark,
	// records an idempotency row, and enqueues a push per bookmark.
	if cfg.App.ScholarshipsRemindersEnabled {
		scholarshipReminderJob := worker.NewScholarshipReminderJob(scholarshipUC)
		worker.NewJobRunner(scholarshipReminderJob, 24*time.Hour, 2*time.Minute, loggerInstance).Start(ctx)
	}

	// Start WebSocket hub for chat
	go chatHub.Run()

	// ─────────────────────────────────────
	// ROUTER
	// ─────────────────────────────────────
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	utils.LoadSwagger(r, &cfg.App)

	// Liveness probe — no DB, no auth, returns 200 as long as the
	// process is up. Dokku's ps:healthcheck can target this so a
	// crashlooping container is detected and stopped instead of
	// being restarted forever.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		sseAuthMiddleware := middleware.SSEAuthMiddleware(tokenSvc)
		authMiddleware := middleware.AuthMiddleware(tokenSvc)
		adminMiddleware := middleware.AdminAuthMiddleware(tokenSvc, adminSvc)
		optionalAuthMiddleware := middleware.OptionalAuthMiddleware(tokenSvc)

		// avatar_border routes register BEFORE user routes to keep the
		// "static /users/me/border before wildcard /users/:id" ordering
		// explicit. gin's router handles this correctly regardless of
		// order, but matching the project's pattern keeps the surface
		// consistent (see /users/me/avatar).
		avatar_border.RegisterAvatarBorderRoutes(api, avatarBorderHandler, optionalAuthMiddleware, authMiddleware)

		category.RegisterCategoryRoutes(api, categoryHandler, adminMiddleware)
		reaction.RegisterReactionRoutes(api, reactionHandler, authMiddleware)
		media.RegisterMediaRoutes(api, mediaHandler, authMiddleware)
		media.RegisterStickerPresignRoute(api, stickerPresignHandler, authMiddleware)
		sticker.RegisterStickerRoutes(api, stickerHandler, authMiddleware, adminMiddleware)
		auth.RegisterAuthRoutes(api, authHandler, authMiddleware)
		thread.RegisterThreadRoutes(api, threadHandler, authMiddleware, adminMiddleware)
		comment.RegisterCommentRoutes(api, commentHandler, authMiddleware, adminMiddleware)
		featured.RegisterFeaturedRoutes(api, featuredHandler)
		reshare.RegisterReshareRoutes(api, reshareHandler, authMiddleware, adminMiddleware)
		avatar.RegisterAvatarRoutes(api, avatarHandler, authMiddleware, adminMiddleware)
		notification.RegisterNotificationRoutes(api, notificationHandler, authMiddleware, sseAuthMiddleware)
		admin_user.RegisterAdminRoutes(api, adminUserHandler, adminMiddleware)
		push.RegisterPushRoutes(api, pushHandler, authMiddleware)
		user.RegisterUserRoutes(api, userHandler, authMiddleware)
		subscription.RegisterSubscriptionRoutes(api, subscriptionHandler, authMiddleware)
		conversation.RegisterConversationRoutes(api, conversationHandler, authMiddleware)
		message.RegisterMessageRoutes(api, messageHandler, authMiddleware)
		scholarships.RegisterScholarshipRoutes(api, scholarshipHandler, authMiddleware)

		// chat-mvp REST surface. Mounted under /api/messages with auth.
		chat_mvp.RegisterRoutes(api, chatMvpHandler, authMiddleware)

		// Admin routes for domain management
		admin := api.Group("/admin")
		admin.Use(adminMiddleware)
		allowed_domain.RegisterAllowedDomainRoutes(admin, allowedDomainHandler)
		moderation.RegisterModerationRoutes(admin, moderationHandler, adminMiddleware)
		avatar_border.RegisterAvatarBorderAdminRoutes(admin, avatarBorderHandler, adminMiddleware)
		media.RegisterBorderPresignRoute(admin, borderPresignHandler, adminMiddleware)
		report.RegisterReportRoutes(admin, reportHandler)
		dashboard.RegisterDashboardRoutes(admin, dashboardHandler)
		featured.RegisterFeaturedAdminRoutes(admin, featuredHandler, adminMiddleware)
	}

	ws.RegisterWSRoutes(r, wsHandler)

	// chat-mvp WS endpoint. JWT lives in Sec-WebSocket-Protocol header
	// (`chat-mvp.<token>`); no auth middleware on the route.
	chatmvpws.RegisterRoutes(r, chatMvpWsHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		loggerInstance.Info("[STARTING_SERVER]", "[PORT]", cfg.App.Port, "[ENV]", cfg.App.Env)
		if err := r.Run(":" + cfg.App.Port); err != nil && err != http.ErrServerClosed {
			loggerInstance.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	loggerInstance.Info("Shutting down server...")

	if pushUC != nil {
		pushUC.Stop()
	}

	loggerInstance.Info("Server exited gracefully")
}

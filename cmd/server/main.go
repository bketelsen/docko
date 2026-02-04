package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bketelsen/docko/internal/ai"
	"github.com/bketelsen/docko/internal/auth"
	"github.com/bketelsen/docko/internal/config"
	"github.com/bketelsen/docko/internal/database"
	"github.com/bketelsen/docko/internal/document"
	"github.com/bketelsen/docko/internal/handler"
	"github.com/bketelsen/docko/internal/inbox"
	"github.com/bketelsen/docko/internal/middleware"
	"github.com/bketelsen/docko/internal/network"
	"github.com/bketelsen/docko/internal/processing"
	"github.com/bketelsen/docko/internal/queue"
	"github.com/bketelsen/docko/internal/storage"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize auth service and sync admin user
	authService := auth.NewService(db, cfg)
	if err := authService.SyncAdminUser(ctx); err != nil {
		slog.Error("failed to sync admin user", "error", err)
		os.Exit(1)
	}

	// Initialize storage
	store, err := storage.New(cfg.Storage.Path)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	slog.Info("storage initialized", "path", cfg.Storage.Path)

	// Initialize queue
	q := queue.New(db, queue.DefaultConfig())

	// Initialize document service
	docService := document.New(db, store, q)

	// Initialize inbox service
	inboxSvc := inbox.New(db, docService, cfg)

	// Initialize network service
	networkSvc := network.New(db, docService, cfg)

	// Check processing dependencies (pdftoppm, cwebp)
	if err := processing.CheckDependencies(); err != nil {
		slog.Warn("processing dependencies missing", "error", err)
		// Don't fatal - app can run, just processing will fail
	}

	// Initialize status broadcaster for SSE updates
	broadcaster := processing.NewStatusBroadcaster()

	// Initialize processor and register with queue
	processor := processing.New(db, docService, store, "static/images/placeholder.webp", broadcaster)
	q.RegisterHandler(document.JobTypeProcess, processor.HandleJob)

	// Initialize AI service and processor
	aiSvc := ai.NewService(db)
	aiProcessor := processing.NewAIProcessor(aiSvc, broadcaster)
	q.RegisterHandler(processing.JobTypeAI, aiProcessor.HandleJob)

	// Start queue workers
	queueCtx, queueCancel := context.WithCancel(context.Background())
	q.Start(queueCtx, document.QueueDefault)
	go q.Start(queueCtx, processing.QueueAI)

	// Start background cleanup of expired sessions
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := authService.CleanupExpiredSessions(context.Background()); err != nil {
				slog.Warn("failed to cleanup expired sessions", "error", err)
			}
		}
	}()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	middleware.Setup(e, cfg)

	h := handler.New(cfg, db, authService, docService, inboxSvc, networkSvc, aiSvc, q, broadcaster)
	h.RegisterRoutes(e)

	// Start inbox watcher in background
	inboxCtx, inboxCancel := context.WithCancel(context.Background())
	go func() {
		if err := inboxSvc.Start(inboxCtx); err != nil && err != context.Canceled {
			slog.Error("inbox service error", "error", err)
		}
	}()

	// Start network service
	networkCtx, networkCancel := context.WithCancel(context.Background())
	if err := networkSvc.Start(networkCtx); err != nil {
		slog.Error("failed to start network service", "error", err)
		os.Exit(1)
	}

	go func() {
		addr := ":" + cfg.Port
		slog.Info("starting server", "url", "http://localhost:"+cfg.Port, "env", cfg.Env)
		if err := e.Start(addr); err != nil {
			slog.Info("shutting down server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	// Stop queue workers
	slog.Info("stopping queue workers...")
	queueCancel()

	// Stop inbox watcher
	slog.Info("stopping inbox watcher...")
	inboxCancel()
	if err := inboxSvc.Stop(); err != nil {
		slog.Error("failed to stop inbox watcher", "error", err)
	}

	// Stop network service
	slog.Info("stopping network service...")
	networkCancel()
	if err := networkSvc.Stop(); err != nil {
		slog.Error("failed to stop network service", "error", err)
	}

	// Stop queue workers
	slog.Info("stopping queue workers...")
	queueCancel()
	q.Stop()

	slog.Info("server stopped")
}

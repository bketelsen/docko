package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docko/internal/auth"
	"docko/internal/config"
	"docko/internal/database"
	"docko/internal/document"
	"docko/internal/handler"
	"docko/internal/inbox"
	"docko/internal/middleware"
	"docko/internal/processing"
	"docko/internal/queue"
	"docko/internal/storage"

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

	// Check processing dependencies (pdftoppm, cwebp)
	if err := processing.CheckDependencies(); err != nil {
		slog.Warn("processing dependencies missing", "error", err)
		// Don't fatal - app can run, just processing will fail
	}

	// Initialize processor and register with queue
	processor := processing.New(db, docService, store, "static/images/placeholder.webp")
	q.RegisterHandler(document.JobTypeProcess, processor.HandleJob)

	// Start queue workers
	queueCtx, queueCancel := context.WithCancel(context.Background())
	q.Start(queueCtx, document.QueueDefault)

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

	h := handler.New(cfg, db, authService, docService, inboxSvc)
	h.RegisterRoutes(e)

	// Start inbox watcher in background
	inboxCtx, inboxCancel := context.WithCancel(context.Background())
	go func() {
		if err := inboxSvc.Start(inboxCtx); err != nil && err != context.Canceled {
			slog.Error("inbox service error", "error", err)
		}
	}()

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

	// Stop queue workers
	slog.Info("stopping queue workers...")
	queueCancel()
	q.Stop()

	slog.Info("server stopped")
}

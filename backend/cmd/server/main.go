package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/timemachine-auto/timemachine/internal/aiagent"
	"github.com/timemachine-auto/timemachine/internal/config"
	auditrepo "github.com/timemachine-auto/timemachine/internal/domain/audit"
	authrepo "github.com/timemachine-auto/timemachine/internal/domain/auth"
	bookingrepo "github.com/timemachine-auto/timemachine/internal/domain/booking"
	faqrepo "github.com/timemachine-auto/timemachine/internal/domain/faq"
	kbrepo "github.com/timemachine-auto/timemachine/internal/domain/knowledge"
	leadrepo "github.com/timemachine-auto/timemachine/internal/domain/lead"
	reviewrepo "github.com/timemachine-auto/timemachine/internal/domain/review"
	svcrepo "github.com/timemachine-auto/timemachine/internal/domain/service"
	"github.com/timemachine-auto/timemachine/internal/httpserver"
	maxpkg "github.com/timemachine-auto/timemachine/internal/integrations/max"
	"github.com/timemachine-auto/timemachine/internal/priceimport"
	"github.com/timemachine-auto/timemachine/pkg/db"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("svc", "timemachine")

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}
	if !cfg.IsProd() {
		log.Warn("running in non-production mode", "env", cfg.AppEnv,
			"max_enabled", cfg.Max.Enabled, "ai_enabled", cfg.AI.Enabled)
	}
	cfg.SetLogger(log)

	pool, err := db.New(ctx, cfg.DB)
	if err != nil {
		log.Error("db init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Domain repos
	svcRepo := svcrepo.NewRepo(pool)
	bookingRepo := bookingrepo.NewRepo(pool)
	leadsRepo := leadrepo.NewRepo(pool)
	faqRepo := faqrepo.NewRepo(pool)
	kbRepo := kbrepo.NewRepo(pool)
	reviewsRepo := reviewrepo.NewRepo(pool)
	authRepo := authrepo.NewRepo(pool)
	auditRepo := auditrepo.NewRepo(pool)
	prices := priceimport.NewService(pool, svcRepo)

	// MAX integration
	maxClient := maxpkg.BuildClient(cfg.Max, log)
	notifier := maxpkg.NewNotifier(maxClient, cfg.Max.NotifyChatID, log)
	maxStore := maxpkg.NewStore(pool)

	// AI agent
	ai := aiagent.BuildAI(cfg.AI)
	agent := aiagent.NewAgent(cfg.AI, log, kbRepo, ai)

	deps := &httpserver.Deps{
		Cfg:      cfg,
		DB:       pool,
		Services: svcRepo,
		Booking:  bookingRepo,
		Leads:    leadsRepo,
		FAQ:      faqRepo,
		KB:       kbRepo,
		Reviews:  reviewsRepo,
		Auth:     authRepo,
		Audit:    auditRepo,
		Prices:   prices,
		Max:      maxClient,
		Notifier: notifier,
		MaxStore: maxStore,
		Agent:    agent,
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpserver.Router(deps),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")
	shutdownCtx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer scancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
	log.Info("bye")
}

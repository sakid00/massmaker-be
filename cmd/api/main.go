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

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/sakid00/massmaker-be/internal/config"
	"github.com/sakid00/massmaker-be/internal/enmasse"
	apphttp "github.com/sakid00/massmaker-be/internal/http"
	"github.com/sakid00/massmaker-be/internal/media"
	"github.com/sakid00/massmaker-be/internal/service"
	"github.com/sakid00/massmaker-be/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	if !cfg.IsProduction() {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("goose dialect", "err", err)
		os.Exit(1)
	}
	sqlDB := stdlib.OpenDBFromPool(db.Pool)
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		sqlDB.Close()
		slog.Error("migrations", "err", err)
		os.Exit(1)
	}
	sqlDB.Close()

	em := enmasse.New(cfg.EnmasseURL, cfg.JWTServiceSecret)
	adminSvc := service.NewAdmin(db, cfg, em)
	if err := adminSvc.Bootstrap(ctx); err != nil {
		slog.Error("staff bootstrap", "err", err)
		os.Exit(1)
	}

	cat := service.NewCatalogue(db, em)
	events := service.NewEvents(db)
	self := service.NewMakerSelf(db, em)
	consents := service.NewConsents(db, cfg.ConsentHashPepper, cfg.LegalDocumentVersion)
	r2, err := media.New(ctx, media.Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Bucket:          cfg.R2Bucket,
		PublicBaseURL:   cfg.R2PublicBaseURL,
	})
	if err != nil {
		slog.Error("r2 client", "err", err)
		os.Exit(1)
	}

	orders := service.NewOrders(db, em, r2)
	handler := apphttp.NewRouter(cat, events, adminSvc, em, self, consents, orders, apphttp.RouterOptions{
		CORSOrigins:     cfg.CORSOrigins,
		JWTAccessSecret: cfg.JWTAccessSecret,
		JWTIssuer:       cfg.JWTIssuer,
		Media:           r2,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Env, "enmasse", cfg.EnmasseURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

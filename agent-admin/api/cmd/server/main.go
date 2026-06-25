package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/agent-admin/api/internal/app"
	_ "github.com/lib/pq"
)

func main() {
	cfg := app.LoadConfig()
	repo := initRepository(cfg)
	serverApp, err := app.NewServer(cfg, repo)
	if err != nil {
		log.Fatalf("initialize server: %v", err)
	}
	if cfg.SchedulerEnabled && repo != nil {
		app.NewScheduler(cfg, repo).Start()
		log.Printf("agent-admin scheduler enabled")
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           serverApp.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("agent-admin-api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("agent-admin-api stopped: %v", err)
	}
}

func initRepository(cfg app.Config) *app.Repository {
	if cfg.DatabaseURL == "" {
		log.Printf("DATABASE_URL is empty; only health and proxied auth endpoints are available")
		return nil
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	if cfg.MigrationEnabled {
		if err := app.RunMigrations(context.Background(), db); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
	}
	return app.NewRepository(db)
}

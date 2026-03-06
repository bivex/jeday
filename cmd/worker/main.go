package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeday/auth/internal/auth/repository"
	"github.com/jeday/auth/internal/auth/service"
	"github.com/jeday/auth/internal/config"
	"github.com/jeday/auth/internal/db"
)

func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DBSource)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	authRepo := repository.NewPostgresRepository(queries, pool)
	authService := service.NewAuthService(authRepo, cfg.TokenSymmetricKey)

	log.Println("Password Hardening Worker started...")

	ticker := time.NewTicker(cfg.WorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker shutting down...")
			return
		case <-ticker.C:
			count, err := authService.UpgradePasswords(ctx, cfg.WorkerUpgradeLimit)
			if err != nil {
				log.Printf("Error upgrading passwords: %v", err)
				continue
			}
			if count > 0 {
				log.Printf("Hardened %d passwords", count)
			}
		}
	}
}

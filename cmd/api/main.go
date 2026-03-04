package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savsgio/atreugo/v11"

	"github.com/jeday/auth/internal/auth/delivery"
	"github.com/jeday/auth/internal/auth/repository"
	"github.com/jeday/auth/internal/auth/service"
	"github.com/jeday/auth/internal/config"
	"github.com/jeday/auth/internal/db"
)

func main() {
	delivery.InitLogger()
	// 1. Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	// 2. Setup Database connection (pgx)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBSource)
	if err != nil {
		log.Fatalf("cannot connect to db: %v", err)
	}
	defer pool.Close()

	// 3. Initialize layers
	queries := db.New(pool)
	authRepo := repository.NewPostgresRepository(queries, pool)
	authService := service.NewAuthService(authRepo, cfg.TokenSymmetricKey)

	// 4. Initialize Atreugo Server
	server := atreugo.New(atreugo.Config{
		Addr:    cfg.ServerAddress,
		Prefork: false,
	})

	// Start pprof server
	// In prefork mode, multiple processes will try to bind to 6060.
	// Only one will succeed, which is fine for profiling.
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			// Silent error as others will fail in prefork
		}
	}()

	// 5. Setup Routes & Delivery
	authHandler := delivery.NewAuthHandler(authService)
	authHandler.RegisterRoutes(server)

	// Add Healthcheck
	server.GET("/health", func(ctx *atreugo.RequestCtx) error {
		return ctx.TextResponse("OK")
	})

	// 6. Graceful shutdown gracefully
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := server.Shutdown(); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("Server exiting")
}

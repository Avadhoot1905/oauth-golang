package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oauth-golang/internal/config"
	router "oauth-golang/internal/http"
	"oauth-golang/internal/storage"
)

func main() {
	// Load configuration from .env file
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection with GORM (includes auto-migration)
	storage.InitDB(cfg.DatabaseURL)

	// Get the underlying *sql.DB for repository initialization
	sqlDB, err := storage.DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	// Initialize repositories (DB interaction layer)
	userRepo := storage.NewUserRepository(sqlDB)
	storageService := &storage.Storage{DB: storage.DB}
	tokenRepo := storage.NewTokenRepository(sqlDB)

	// Initialize HTTP router with all handlers (API input layer)
	handler := router.NewRouter(cfg, userRepo, storageService, tokenRepo)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting OAuth microservice on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

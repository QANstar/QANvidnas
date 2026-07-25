package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"qanvidnas/internal/config"
	"qanvidnas/internal/database"
	"qanvidnas/internal/router"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting QANvidnas server...")

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: Could not load config.yaml: %v", err)
		log.Println("Using default configuration")
		cfg = config.Default()
	}

	// Initialize database
	db, err := database.Init(cfg.DataDir + "/qanvidnas.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup router
	r := router.Setup(db, cfg)

	// Start server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server listening on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

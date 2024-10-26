package main

import (
	"log"
	"net/http"

	"url-shortener/config"
	"url-shortener/db"
	"url-shortener/routes"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db.InitDatabase(cfg)

	// Setup routes
	router := routes.SetupRoutes(cfg)

	// Start the server
	log.Printf("Server is running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

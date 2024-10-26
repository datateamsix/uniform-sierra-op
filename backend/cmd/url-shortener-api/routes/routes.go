package routes

import (
	"url-shortener/config"
	"url-shortener/controllers"
	"url-shortener/middlewares"

	"github.com/gorilla/mux"
)

func SetupRoutes(cfg config.Config) *mux.Router {
	router := mux.NewRouter()

	// Public Routes
	router.HandleFunc("/shorten", controllers.ShortenURL(&cfg)).Methods("POST")
	router.HandleFunc("/{shortCode}", controllers.RedirectURL()).Methods("GET")

	// Apply Middlewares
	router.Use(middlewares.LoggingMiddleware)
	router.Use(middlewares.RateLimitMiddleware)

	return router
}

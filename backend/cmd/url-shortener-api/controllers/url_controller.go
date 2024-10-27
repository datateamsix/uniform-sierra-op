package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"url-shortener/config"
	"url-shortener/db"
	"url-shortener/models"
	"url-shortener/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShortenURLRequest represents the expected payload for shortening URLs.
type ShortenURLRequest struct {
	URL                string     `json:"url"`
	IntendedLiveDate   *time.Time `json:"intended_live_date,omitempty"`
	IntendedExpiryDate *time.Time `json:"intended_expiry_date,omitempty"`
}

// ShortenURLResponse represents the response payload.
type ShortenURLResponse struct {
	ShortURL           string     `json:"short_url"`
	Status             string     `json:"status"`
	IntendedLiveDate   *time.Time `json:"intended_live_date,omitempty"`
	IntendedExpiryDate *time.Time `json:"intended_expiry_date,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Message string `json:"message"`
}

// ShortenURL handles the URL shortening logic.
func ShortenURL(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShortenURLRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil || req.URL == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Validate URL Syntax and HTTPS
		if err := utils.ValidateURLSyntax(req.URL); err != nil {
			respondWithError(w, fmt.Sprintf("Invalid URL: %v", err), http.StatusBadRequest)
			return
		}

		// Check URL status
		urlCheckResult, err := utils.CheckURLStatus(req.URL)
		if err != nil {
			log.Println("Error checking URL status:", err)
			respondWithError(w, "Error checking URL status. Please try again.", http.StatusInternalServerError)
			return
		}

		// Determine if the URL is live based on the status code
		isLive := urlCheckResult.StatusCode >= 200 && urlCheckResult.StatusCode < 300
		status := "live"
		if !isLive {
			status = "inactive"
		}

		// Validate dates
		now := time.Now()
		if req.IntendedExpiryDate != nil && req.IntendedExpiryDate.Before(now) {
			respondWithError(w, "Expiry date cannot be in the past", http.StatusBadRequest)
			return
		}
		if req.IntendedLiveDate != nil && req.IntendedExpiryDate != nil && req.IntendedLiveDate.After(*req.IntendedExpiryDate) {
			respondWithError(w, "Live date cannot be after expiry date", http.StatusBadRequest)
			return
		}

		// Proceed to shorten the URL
		shortCode := generateShortCode()

		// Save to database
		urlMapping := models.UrlMapping{
			ShortCode:          shortCode,
			OriginalUrl:        req.URL,
			IntendedLiveDate:   req.IntendedLiveDate,
			IntendedExpiryDate: req.IntendedExpiryDate,
			Status:             status,
			LastCheckedAt:      time.Now(),
		}

		if err := db.DB.Create(&urlMapping).Error; err != nil {
			log.Println("Error saving URL mapping:", err)
			respondWithError(w, "Error creating shortened URL. Please try again.", http.StatusInternalServerError)
			return
		}

		// Construct the shortened URL
		shortURL := constructShortURL(r, shortCode)

		// Respond with the shortened URL and additional information
		response := ShortenURLResponse{
			ShortURL:           shortURL,
			Status:             status,
			IntendedLiveDate:   urlMapping.IntendedLiveDate,
			IntendedExpiryDate: urlMapping.IntendedExpiryDate,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RedirectURL handles redirection from short URLs to original URLs.
func RedirectURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortCode := r.URL.Path[1:] // Remove the leading '/'

		var urlMapping models.UrlMapping
		if err := db.DB.Where("short_code = ?", shortCode).First(&urlMapping).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "URL not found.", http.StatusNotFound)
			} else {
				log.Printf("Error retrieving URL mapping: %v", err)
				http.Error(w, "Internal server error.", http.StatusInternalServerError)
			}
			return
		}

		// Check if the URL has expired
		if urlMapping.IntendedExpiryDate != nil && time.Now().After(*urlMapping.IntendedExpiryDate) {
			http.Error(w, "This URL has expired.", http.StatusGone)
			return
		}

		// Check if the URL is live
		if urlMapping.Status != "live" {
			http.Error(w, "This URL is not currently live.", http.StatusGone)
			return
		}

		http.Redirect(w, r, urlMapping.OriginalUrl, http.StatusFound)
	}
}

// Helper functions
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}

func generateShortCode() string {
	return uuid.New().String()[:8] // Example: use the first 8 characters of a UUID
}

func constructShortURL(r *http.Request, shortCode string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	return fmt.Sprintf("%s://%s/%s", scheme, host, shortCode)
}

package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"url-shortener/config"
	"url-shortener/db"
	"url-shortener/models"
	"url-shortener/utils"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShortenURLRequest represents the expected payload for shortening URLs.
type ShortenURLRequest struct {
	URL              string     `json:"url"`
	IntendedLiveDate *time.Time `json:"intended_live_date,omitempty"` // Optional field
}

// ShortenURLResponse represents the response payload.
type ShortenURLResponse struct {
	ShortURL string `json:"short_url"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Message string `json:"message"`
}

var scheduler *gocron.Scheduler

func init() {
	scheduler = gocron.NewScheduler(time.UTC)
	scheduler.StartAsync()
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

		// Check for Redirects (301/302)
		hasRedirect, redirectCount := utils.CheckRedirects(req.URL)
		if hasRedirect {
			log.Printf("URL has %d redirects: %s", redirectCount, req.URL)
			// Optionally, assign a risk score or take other actions
		}

		// Check against Safe Browsing API
		isSafe := utils.CheckSafeBrowsing(*cfg, req.URL)
		if !isSafe {
			log.Println("URL flagged as unsafe:", req.URL)
			respondWithError(w, "This URL is potentially unsafe.", http.StatusBadRequest)
			return
		}

		// Check if the URL is live by performing a HEAD request
		isLive, err := utils.CheckIfURLIsLive(req.URL)
		if err != nil {
			log.Println("Error checking URL live status:", err)
			respondWithError(w, "Error checking URL status. Please try again.", http.StatusInternalServerError)
			return
		}

		// Initialize status and intended live date
		status := "live"
		if !isLive {
			status = "inactive" // Or "pending"
			// If the user provided an IntendedLiveDate, use it
			if req.IntendedLiveDate != nil {
				// Ensure the date is in the future
				if req.IntendedLiveDate.Before(time.Now()) {
					respondWithError(w, "Intended live date must be in the future.", http.StatusBadRequest)
					return
				}
			}
		}

		// Proceed to shorten the URL
		shortCode := generateShortCode()

		// Save to database
		urlMapping := models.UrlMapping{
			ShortCode:        shortCode,
			OriginalUrl:      req.URL,
			IntendedLiveDate: req.IntendedLiveDate,
			Status:           status,
			LastCheckedAt:    time.Now(),
		}

		if err := db.DB.Create(&urlMapping).Error; err != nil {
			log.Println("Error saving URL mapping:", err)
			respondWithError(w, "Error creating shortened URL. Please try again.", http.StatusInternalServerError)
			return
		}

		// If the URL is not live and an intended live date is set, schedule a check
		if !isLive && req.IntendedLiveDate != nil {
			scheduleURLCheck(urlMapping.ID, *req.IntendedLiveDate)
		}

		// Construct the shortened URL
		shortURL := constructShortURL(r, shortCode)

		// Respond with the shortened URL
		response := ShortenURLResponse{ShortURL: shortURL}
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
				http.Error(w, "Internal server error.", http.StatusInternalServerError)
			}
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

func getIPAddress(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return ip
}

func logMaliciousURL(url, userAgent, ipAddress string) {
	logEntry := models.MaliciousLog{
		URL:       url,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		RiskScore: 5, // Example risk score
		Details:   "Failed Safe Browsing check",
	}

	if err := db.DB.Create(&logEntry).Error; err != nil {
		log.Println("Error logging malicious URL:", err)
	}
}

// scheduleURLCheck schedules a periodic check for the URL status
func scheduleURLCheck(urlID uint, intendedLiveDate time.Time) {
	_, err := scheduler.ScheduleOnce(intendedLiveDate, func() {
		var urlMapping models.UrlMapping
		if err := db.DB.First(&urlMapping, urlID).Error; err != nil {
			log.Println("Error fetching URL mapping for scheduled check:", err)
			return
		}

		// Check if the URL is live
		isLive, err := utils.CheckIfURLIsLive(urlMapping.OriginalUrl)
		if err != nil {
			log.Println("Error during scheduled URL live check:", err)
			return
		}

		if isLive {
			db.DB.Model(&urlMapping).Update("status", "live")
			log.Printf("URL ID %d is now live.\n", urlID)
		} else {
			log.Printf("URL ID %d is still not live.\n", urlID)
		}
	})

	if err != nil {
		log.Println("Error scheduling URL check:", err)
	}
}

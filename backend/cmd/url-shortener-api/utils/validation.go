package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"url-shortener/config"
)

// Custom error types
var (
	ErrInvalidURLSyntax          = errors.New("invalid URL syntax")
	ErrURLNotHTTPS               = errors.New("URL scheme is not HTTPS")
	ErrURLRedirect               = errors.New("URL redirects to another location")
	ErrSafeBrowsingAPIKeyMissing = errors.New("safe browsing API key is not set")
)

// Safe Browsing API Response
type SafeBrowsingResponse struct {
	Matches []struct {
		ThreatType      string `json:"threatType"`
		PlatformType    string `json:"platformType"`
		ThreatEntryType string `json:"threatEntryType"`
		Threat          struct {
			URL string `json:"url"`
		} `json:"threat"`
	} `json:"matches"`
}

// SafeBrowsingResult represents the result of a Safe Browsing check
type SafeBrowsingResult struct {
	IsSafe  bool   `json:"is_safe"`
	Message string `json:"message"`
}

// ValidateURLSyntax ensures the URL is properly formatted and uses HTTPS.
func ValidateURLSyntax(inputURL string) error {
	parsedURL, err := url.Parse(inputURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("%w: %s", ErrInvalidURLSyntax, inputURL)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: %s", ErrURLNotHTTPS, inputURL)
	}

	return nil
}

// CheckRedirects checks if the URL responds with 301 or 302 status codes.
func CheckRedirects(inputURL string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(inputURL)
	if err != nil {
		return fmt.Errorf("failed to check redirects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		return fmt.Errorf("%w: %s (status code: %d)", ErrURLRedirect, inputURL, resp.StatusCode)
	}

	return nil
}

// CheckSafeBrowsing uses Google's Safe Browsing API to check the URL.
func CheckSafeBrowsing(cfg config.Config, inputURL string) (SafeBrowsingResult, error) {
	result := SafeBrowsingResult{IsSafe: true, Message: "URL is safe"}

	apiKey := cfg.SafeBrowsingAPIKey
	if apiKey == "" {
		return result, ErrSafeBrowsingAPIKeyMissing
	}

	endpoint := fmt.Sprintf("https://safebrowsing.googleapis.com/v4/threatMatches:find?key=%s", apiKey)
	requestBody := map[string]interface{}{
		"client": map[string]string{
			"clientId":      "yourapp",
			"clientVersion": "1.0",
		},
		"threatInfo": map[string]interface{}{
			"threatTypes":      []string{"MALWARE", "SOCIAL_ENGINEERING"},
			"platformTypes":    []string{"ANY_PLATFORM"},
			"threatEntryTypes": []string{"URL"},
			"threatEntries": []map[string]string{
				{"url": inputURL},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return result, err
	}

	resp, err := http.Post(endpoint, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	var sbResp SafeBrowsingResponse
	err = json.NewDecoder(resp.Body).Decode(&sbResp)
	if err != nil {
		return result, err
	}

	if len(sbResp.Matches) > 0 {
		result.IsSafe = false
		result.Message = fmt.Sprintf("URL is unsafe: %s. Threat type: %s", inputURL, sbResp.Matches[0].ThreatType)
	}

	return result, nil
}

// URLCheckResult represents the result of URL checks
type URLCheckResult struct {
	StatusCode  int    `json:"status_code"`
	IsHTTPS     bool   `json:"is_https"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

func CheckURLStatus(inputURL string) (URLCheckResult, error) {
	result := URLCheckResult{IsHTTPS: false}

	// Validate URL syntax
	parsedURL, err := url.Parse(inputURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return result, fmt.Errorf("%w: %s", ErrInvalidURLSyntax, inputURL)
	}

	// Check if the URL uses HTTPS
	result.IsHTTPS = parsedURL.Scheme == "https"

	// Check the URL status
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(inputURL)
	if err != nil {
		return result, fmt.Errorf("failed to check URL status: %w", err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Check for redirect and get the redirect URL
	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		result.RedirectURL = resp.Header.Get("Location")
	}

	return result, nil
}

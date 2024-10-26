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
	ErrURLNotLive                = errors.New("URL is not live")
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
func CheckSafeBrowsing(cfg config.Config, inputURL string) error {
	apiKey := cfg.SafeBrowsingAPIKey
	if apiKey == "" {
		return ErrSafeBrowsingAPIKeyMissing
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
		return err
	}

	resp, err := http.Post(endpoint, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var sbResp SafeBrowsingResponse
	err = json.NewDecoder(resp.Body).Decode(&sbResp)
	if err != nil {
		return err
	}

	if len(sbResp.Matches) > 0 {
		return fmt.Errorf("URL is unsafe: %s", inputURL)
	}

	return nil
}

// CheckIfURLIsLive performs a HEAD request to check if the URL is live.
func CheckIfURLIsLive(inputURL string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Head(inputURL)
	if err != nil {
		return fmt.Errorf("failed to check if URL is live: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("%w: %s (status code: %d)", ErrURLNotLive, inputURL, resp.StatusCode)
	}

	return nil
}

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
)

const (
	apiURL  = "https://api.zoom.us/v2/users/me/meetings"
	authURL = "https://zoom.us/oauth/token?grant_type=account_credentials"
)

// OAuthConfig holds the OAuth configuration details.
type OAuthConfig struct {
	AccountID    string `json:"account_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// MeetingDetails holds information about the meeting.
type MeetingDetails struct {
	Topic    string `json:"topic"`
	Type     int    `json:"type"`
	Start    string `json:"start_time,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// ResponseData holds the response data from Zoom.
type ResponseData struct {
	JoinURL string `json:"join_url"`
}

// OAuthTokenResponse represents the OAuth token response.
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func loadOAuthConfig() OAuthConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error finding user home directory: %v", err)
	}

	configFile := filepath.Join(homeDir, ".zoom-meeting.config.json")
	fileContent, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config OAuthConfig
	if err := json.Unmarshal(fileContent, &config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	if config.AccountID == "" || config.ClientID == "" || config.ClientSecret == "" {
		log.Fatalf("Account ID or Client ID or Client Secret not found in config file")
	}

	return config
}

func getOAuthToken(config OAuthConfig) string {
	client := &http.Client{}

	// Encode Client ID and Client Secret
	auth := base64.StdEncoding.EncodeToString([]byte(config.ClientID + ":" + config.ClientSecret))

	// Create request with the required body parameters
	data := "grant_type=account_credentials&account_id=" + config.AccountID
	req, err := http.NewRequest("POST", authURL, bytes.NewBufferString(data))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	// Add headers
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error retrieving OAuth token: %v", err)
	}
	defer resp.Body.Close()

	// Decode response
	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Fatalf("Error decoding OAuth response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		log.Fatalf("Failed to retrieve access token")
	}

	return tokenResp.AccessToken
}

func createZoomMeeting(details MeetingDetails, config OAuthConfig) (string, error) {
	client := &http.Client{}
	meetingDetails, err := json.Marshal(details)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(meetingDetails))
	if err != nil {
		return "", err
	}

	// Use OAuth token for authorization
	req.Header.Add("Authorization", "Bearer "+getOAuthToken(config))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var responseData ResponseData
	if err := json.Unmarshal(data, &responseData); err != nil {
		return "", err
	}

	return responseData.JoinURL, nil
}

func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func openURL(url string) error {
	return open.Run(url)
}

func main() {
	// Load OAuth configuration
	config := loadOAuthConfig()

	// Get current time in ISO 8601 format
	currentTime := time.Now().Format(time.RFC3339)

	// Set your meeting details
	meetingDetails := MeetingDetails{
		Topic:    "My Meeting",
		Type:     2,           // 1 for instant meeting, 2 for scheduled meeting
		Start:    currentTime, // Set your desired time
		Duration: 60,          // Duration in minutes
	}

	// Create Zoom meeting
	meetingLink, err := createZoomMeeting(meetingDetails, config)
	if err != nil {
		log.Fatalf("Error creating meeting: %v", err)
	}

	fmt.Println("Meeting link:", meetingLink)

	// Copy link to clipboard
	if err := copyToClipboard(meetingLink); err != nil {
		log.Fatalf("Error copying to clipboard: %v", err)
	}

	// Open the meeting link
	if err := openURL(meetingLink); err != nil {
		log.Fatalf("Error opening URL: %v", err)
	}
}

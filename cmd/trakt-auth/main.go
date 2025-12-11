package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	baseURL = "https://api.trakt.tv"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int64  `json:"created_at"`
}

type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func main() {
	clientID := flag.String("client-id", "", "Trakt API Client ID (required)")
	clientSecret := flag.String("client-secret", "", "Trakt API Client Secret (required)")
	flag.Parse()

	if *clientID == "" || *clientSecret == "" {
		fmt.Println("Error: Both -client-id and -client-secret are required")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nGet your credentials at: https://trakt.tv/oauth/applications")
		os.Exit(1)
	}

	fmt.Println("=== Trakt OAuth Device Authentication ===")

	// Step 1: Get device code
	deviceCode, err := getDeviceCode(*clientID)
	if err != nil {
		fmt.Printf("Error getting device code: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Show user instructions
	fmt.Printf("Please visit: %s\n", deviceCode.VerificationURL)
	fmt.Printf("And enter this code: %s\n\n", deviceCode.UserCode)
	fmt.Println("Waiting for authorization...")

	// Step 3: Poll for token
	token, err := pollForToken(*clientID, *clientSecret, deviceCode)
	if err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Display results
	fmt.Println("\n=== Success! ===")
	fmt.Printf("\nAdd these values to your config.yaml:\n\n")
	fmt.Printf("clientId: \"%s\"\n", *clientID)
	fmt.Printf("accessToken: \"%s\"\n", token.AccessToken)
	fmt.Printf("\nYour access token does not expire.\n")
	if token.RefreshToken != "" {
		fmt.Printf("Refresh token (for future use): %s\n", token.RefreshToken)
	}
}

func getDeviceCode(clientID string) (*DeviceCodeResponse, error) {
	reqBody := map[string]string{
		"client_id": clientID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/oauth/device/code", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var deviceCode DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceCode); err != nil {
		return nil, err
	}

	return &deviceCode, nil
}

func pollForToken(clientID, clientSecret string, deviceCode *DeviceCodeResponse) (*TokenResponse, error) {
	reqBody := map[string]string{
		"code":          deviceCode.DeviceCode,
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	interval := time.Duration(deviceCode.Interval) * time.Second
	timeout := time.After(time.Duration(deviceCode.ExpiresIn) * time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("authorization timed out")
		case <-ticker.C:
			req, err := http.NewRequest("POST", baseURL+"/oauth/device/token", bytes.NewBuffer(jsonData))
			if err != nil {
				return nil, err
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}

			if resp.StatusCode == http.StatusOK {
				var token TokenResponse
				if err := json.Unmarshal(body, &token); err != nil {
					return nil, err
				}
				return &token, nil
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(body, &errResp); err == nil {
				if errResp.Error == "authorization_pending" {
					fmt.Print(".")
					continue
				} else if errResp.Error == "slow_down" {
					interval += 1 * time.Second
					ticker.Reset(interval)
					continue
				} else if errResp.Error == "expired_token" {
					return nil, fmt.Errorf("device code expired")
				} else if errResp.Error == "access_denied" {
					return nil, fmt.Errorf("access denied by user")
				}
			}

			return nil, fmt.Errorf("unexpected response: %s", string(body))
		}
	}
}

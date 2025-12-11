package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	authURL  = "https://anilist.co/api/v2/oauth/authorize"
	tokenURL = "https://anilist.co/api/v2/oauth/token"
)

type TokenResponse struct {
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

func main() {
	clientID := flag.String("client-id", "", "AniList API Client ID (required)")
	clientSecret := flag.String("client-secret", "", "AniList API Client Secret (required)")
	redirectURI := flag.String("redirect-uri", "https://anilist.co/api/v2/oauth/pin", "Redirect URI (default: pin flow)")
	flag.Parse()

	if *clientID == "" || *clientSecret == "" {
		fmt.Println("Error: Both -client-id and -client-secret are required")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nGet your credentials at: https://anilist.co/settings/developer")
		os.Exit(1)
	}

	fmt.Println("=== AniList OAuth Authentication ===")

	authURLWithParams := buildAuthURL(*clientID, *redirectURI)
	fmt.Printf("Step 1: Visit this URL to authorize:\n%s\n\n", authURLWithParams)

	fmt.Print("Step 2: After authorizing, you'll be redirected to a page.\n")
	fmt.Print("Copy the 'code' parameter from the URL or the code shown on the page.\n")
	fmt.Print("\nEnter the authorization code: ")

	var authCode string
	fmt.Scanln(&authCode)

	if authCode == "" {
		fmt.Println("Error: Authorization code is required")
		os.Exit(1)
	}

	fmt.Println("\nExchanging code for access token...")
	token, err := exchangeCodeForToken(*clientID, *clientSecret, *redirectURI, authCode)
	if err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Success! ===")
	fmt.Printf("\nAdd this value to your config.yaml:\n\n")
	fmt.Printf("accessToken: \"%s\"\n", token.AccessToken)
	fmt.Printf("\nYour access token is valid for %d seconds (approximately 1 year).\n", token.ExpiresIn)
}

func buildAuthURL(clientID, redirectURI string) string {
	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("redirect_uri", redirectURI)
	params.Add("response_type", "code")

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

func exchangeCodeForToken(clientID, clientSecret, redirectURI, authCode string) (*TokenResponse, error) {
	reqBody := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"redirect_uri":  redirectURI,
		"code":          authCode,
	}

	formData := url.Values{}
	for key, value := range reqBody {
		formData.Add(key, value)
	}

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
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

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

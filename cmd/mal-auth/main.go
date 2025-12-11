package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	authURL  = "https://myanimelist.net/v1/oauth2/authorize"
	tokenURL = "https://myanimelist.net/v1/oauth2/token"
)

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // TODO: use this
}

func main() {
	clientID := flag.String("client-id", "", "MyAnimeList API Client ID (required)")
	clientSecret := flag.String("client-secret", "", "MyAnimeList API Client Secret (required)")
	redirectURI := flag.String("redirect-uri", "http://localhost", "Redirect URI (must match your MAL app config)")
	flag.Parse()

	if *clientID == "" || *clientSecret == "" {
		fmt.Println("Error: Both -client-id and -client-secret are required")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nGet your credentials at: https://myanimelist.net/apiconfig")
		fmt.Println("\nIMPORTANT: The redirect-uri must match what you set in your MAL app config.")
		os.Exit(1)
	}

	fmt.Println("=== MyAnimeList OAuth Authentication (PKCE) ===")
	
	codeVerifier, codeChallenge := generatePKCE()

	authURLWithParams := buildAuthURL(*clientID, *redirectURI, codeChallenge)
	fmt.Printf("Step 1: Visit this URL to authorize:\n%s\n\n", authURLWithParams)

	fmt.Print("Step 2: After authorizing, you'll be redirected to a page.\n")
	fmt.Print("Copy the 'code' parameter from the URL.\n")
	fmt.Print("\nEnter the authorization code: ")

	var authCode string
	fmt.Scanln(&authCode)

	if authCode == "" {
		fmt.Println("Error: Authorization code is required")
		os.Exit(1)
	}

	fmt.Println("\nExchanging code for access token...")
	token, err := exchangeCodeForToken(*clientID, *clientSecret, *redirectURI, authCode, codeVerifier)
	if err != nil {
		fmt.Printf("Error getting token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Success! ===")
	fmt.Printf("\nAdd these values to your config.yaml:\n\n")
	fmt.Printf("accessToken: \"%s\"\n", token.AccessToken)
	fmt.Printf("refreshToken: \"%s\"\n", token.RefreshToken)
	fmt.Printf("\nYour access token expires in %d seconds (31 days).\n", token.ExpiresIn)
	fmt.Println("The refresh token can be used to get a new access token when it expires.")
}

func generatePKCE() (string, string) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}

	codeVerifier := base64.RawURLEncoding.EncodeToString(randomBytes)

	// MAL only supports "plain" method, so code_challenge = code_verifier
	// (no SHA256 hashing)
	codeChallenge := codeVerifier

	return codeVerifier, codeChallenge
}

func buildAuthURL(clientID, redirectURI, codeChallenge string) string {
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", clientID)
	params.Add("redirect_uri", redirectURI)
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "plain")

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

func exchangeCodeForToken(clientID, clientSecret, redirectURI, authCode, codeVerifier string) (*TokenResponse, error) {
	formData := url.Values{}
	formData.Add("client_id", clientID)
	formData.Add("client_secret", clientSecret)
	formData.Add("redirect_uri", redirectURI)
	formData.Add("code", authCode)
	formData.Add("code_verifier", codeVerifier)
	formData.Add("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

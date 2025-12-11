package auth

import "net/http"

type Authenticator interface {
	Authenticate(r *http.Request) bool
}

// NoAuth is an authenticator that allows all requests
type NoAuth struct{}

func (n *NoAuth) Authenticate(r *http.Request) bool {
	return true
}

// APIKeyAuth authenticates requests using a config-specified API key in the query parameter
type APIKeyAuth struct {
	APIKey string
}

func (a *APIKeyAuth) Authenticate(r *http.Request) bool {
	providedKey := r.URL.Query().Get("apikey")
	return providedKey == a.APIKey
}

// NewAuthenticator creates an authenticator based on method and config
func NewAuthenticator(method, apiKey string) Authenticator {
	switch method {
	case "apikey":
		return &APIKeyAuth{APIKey: apiKey}
	default:
		return &NoAuth{}
	}
}

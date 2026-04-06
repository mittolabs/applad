// Package oauth implements OAuth2 provider integration for Applad.
// Supports Google, GitHub, and Apple as initial providers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserInfo is the normalized user info returned by any OAuth provider.
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// Provider defines an OAuth2 provider configuration.
type Provider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
	ParseUser    func(body []byte) (*UserInfo, error)
}

// Providers returns the known OAuth2 providers.
func Providers(google, github, apple ProviderConfig) map[string]*Provider {
	providers := map[string]*Provider{}

	if google.ClientID != "" {
		providers["google"] = &Provider{
			Name:         "google",
			ClientID:     google.ClientID,
			ClientSecret: google.ClientSecret,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:       []string{"openid", "email", "profile"},
			ParseUser: func(body []byte) (*UserInfo, error) {
				var u struct {
					ID    string `json:"id"`
					Email string `json:"email"`
					Name  string `json:"name"`
				}
				if err := json.Unmarshal(body, &u); err != nil {
					return nil, err
				}
				return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "google"}, nil
			},
		}
	}

	if github.ClientID != "" {
		providers["github"] = &Provider{
			Name:         "github",
			ClientID:     github.ClientID,
			ClientSecret: github.ClientSecret,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       []string{"user:email"},
			ParseUser: func(body []byte) (*UserInfo, error) {
				var u struct {
					ID    int    `json:"id"`
					Email string `json:"email"`
					Name  string `json:"name"`
					Login string `json:"login"`
				}
				if err := json.Unmarshal(body, &u); err != nil {
					return nil, err
				}
				name := u.Name
				if name == "" {
					name = u.Login
				}
				return &UserInfo{
					ID:       fmt.Sprintf("%d", u.ID),
					Email:    u.Email,
					Name:     name,
					Provider: "github",
				}, nil
			},
		}
	}

	if apple.ClientID != "" {
		providers["apple"] = &Provider{
			Name:         "apple",
			ClientID:     apple.ClientID,
			ClientSecret: apple.ClientSecret,
			AuthURL:      "https://appleid.apple.com/auth/authorize",
			TokenURL:     "https://appleid.apple.com/auth/token",
			UserInfoURL:  "", // Apple returns info in the ID token
			Scopes:       []string{"name", "email"},
			ParseUser: func(body []byte) (*UserInfo, error) {
				// Apple's user info comes from the ID token, handled separately
				var u struct {
					Sub   string `json:"sub"`
					Email string `json:"email"`
				}
				if err := json.Unmarshal(body, &u); err != nil {
					return nil, err
				}
				return &UserInfo{ID: u.Sub, Email: u.Email, Provider: "apple"}, nil
			},
		}
	}

	return providers
}

// ProviderConfig holds client credentials for an OAuth2 provider.
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
}

// GetAuthURL returns the OAuth2 authorization URL for the provider.
func (p *Provider) GetAuthURL(redirectURI, state string) string {
	params := url.Values{
		"client_id":     {p.ClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(p.Scopes, " ")},
		"state":         {state},
	}
	return p.AuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func (p *Provider) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth: token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("oauth: parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("oauth: %s", tokenResp.Error)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oauth: empty access token")
	}
	return tokenResp.AccessToken, nil
}

// GetUserInfo fetches user info from the provider using the access token.
func (p *Provider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if p.UserInfoURL == "" {
		return nil, fmt.Errorf("oauth: provider does not support userinfo endpoint")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return p.ParseUser(body)
}

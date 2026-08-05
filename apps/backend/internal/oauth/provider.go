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
	ID    string `json:"id"`
	Email string `json:"email"`
	// EmailVerified is true only when the provider asserts the email is
	// verified. It gates account linking: an unverified (or attacker-chosen)
	// email must never attach an OAuth identity to an existing account.
	EmailVerified bool   `json:"emailVerified"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
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
	// ClientSecretFn, when set, produces the client secret at exchange time and
	// overrides ClientSecret. Apple needs this: its client secret is not a stored
	// constant but a short-lived ES256 JWT signed from the account's .p8 key.
	ClientSecretFn func(ctx context.Context) (string, error)
}

// Providers returns the known OAuth2 providers configured via environment variables.
// It accepts a map of provider name to ProviderConfig credentials. Only providers
// with a non-empty ClientID are included.
func Providers(configs map[string]ProviderConfig) map[string]*Provider {
	defs := AllProviderDefinitions()
	providers := map[string]*Provider{}

	for name, cfg := range configs {
		if cfg.ClientID == "" {
			continue
		}
		if def, ok := defs[name]; ok {
			providers[name] = def.ToProvider(cfg.ClientID, cfg.ClientSecret)
		}
	}

	return providers
}

// ProvidersWithDomain returns providers including domain-dependent ones (Auth0, Okta, OIDC).
func ProvidersWithDomain(configs map[string]ProviderConfig, auth0Domain, oktaDomain, oidcAuthURL, oidcTokenURL, oidcUserInfoURL string) map[string]*Provider {
	defs := AllProviderDefinitionsWithDomain(auth0Domain, oktaDomain, oidcAuthURL, oidcTokenURL, oidcUserInfoURL)
	providers := map[string]*Provider{}

	for name, cfg := range configs {
		if cfg.ClientID == "" {
			continue
		}
		if def, ok := defs[name]; ok {
			providers[name] = def.ToProvider(cfg.ClientID, cfg.ClientSecret)
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
	clientSecret := p.ClientSecret
	if p.ClientSecretFn != nil {
		s, err := p.ClientSecretFn(ctx)
		if err != nil {
			return "", fmt.Errorf("oauth: build client secret: %w", err)
		}
		clientSecret = s
	}

	data := url.Values{
		"client_id":     {p.ClientID},
		"client_secret": {clientSecret},
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

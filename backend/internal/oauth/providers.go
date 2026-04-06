package oauth

import (
	"encoding/json"
	"fmt"
)

// AllProviderDefinitions returns the definitions for all supported OAuth2 providers.
// These contain the URLs and scopes — client ID/secret are configured per-project.
func AllProviderDefinitions() map[string]ProviderDefinition {
	return map[string]ProviderDefinition{
		"google": {
			Name:        "Google",
			AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:    "https://oauth2.googleapis.com/token",
			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGoogleUser,
		},
		"github": {
			Name:        "GitHub",
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			UserInfoURL: "https://api.github.com/user",
			Scopes:      []string{"user:email"},
			ParseUser:   parseGitHubUser,
		},
		"apple": {
			Name:        "Apple",
			AuthURL:     "https://appleid.apple.com/auth/authorize",
			TokenURL:    "https://appleid.apple.com/auth/token",
			UserInfoURL: "",
			Scopes:      []string{"name", "email"},
			ParseUser:   parseAppleUser,
		},
		"facebook": {
			Name:        "Facebook",
			AuthURL:     "https://www.facebook.com/v18.0/dialog/oauth",
			TokenURL:    "https://graph.facebook.com/v18.0/oauth/access_token",
			UserInfoURL: "https://graph.facebook.com/v18.0/me?fields=id,name,email",
			Scopes:      []string{"email", "public_profile"},
			ParseUser:   parseGenericUser("facebook"),
		},
		"discord": {
			Name:        "Discord",
			AuthURL:     "https://discord.com/api/oauth2/authorize",
			TokenURL:    "https://discord.com/api/oauth2/token",
			UserInfoURL: "https://discord.com/api/users/@me",
			Scopes:      []string{"identify", "email"},
			ParseUser:   parseDiscordUser,
		},
		"twitter": {
			Name:        "Twitter",
			AuthURL:     "https://twitter.com/i/oauth2/authorize",
			TokenURL:    "https://api.twitter.com/2/oauth2/token",
			UserInfoURL: "https://api.twitter.com/2/users/me?user.fields=profile_image_url",
			Scopes:      []string{"tweet.read", "users.read"},
			ParseUser:   parseTwitterUser,
		},
		"microsoft": {
			Name:        "Microsoft",
			AuthURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserInfoURL: "https://graph.microsoft.com/v1.0/me",
			Scopes:      []string{"openid", "email", "profile", "User.Read"},
			ParseUser:   parseMicrosoftUser,
		},
		"slack": {
			Name:        "Slack",
			AuthURL:     "https://slack.com/openid/connect/authorize",
			TokenURL:    "https://slack.com/api/openid.connect.token",
			UserInfoURL: "https://slack.com/api/openid.connect.userInfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("slack"),
		},
		"spotify": {
			Name:        "Spotify",
			AuthURL:     "https://accounts.spotify.com/authorize",
			TokenURL:    "https://accounts.spotify.com/api/token",
			UserInfoURL: "https://api.spotify.com/v1/me",
			Scopes:      []string{"user-read-email", "user-read-private"},
			ParseUser:   parseSpotifyUser,
		},
		"linkedin": {
			Name:        "LinkedIn",
			AuthURL:     "https://www.linkedin.com/oauth/v2/authorization",
			TokenURL:    "https://www.linkedin.com/oauth/v2/accessToken",
			UserInfoURL: "https://api.linkedin.com/v2/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("linkedin"),
		},
		"gitlab": {
			Name:        "GitLab",
			AuthURL:     "https://gitlab.com/oauth/authorize",
			TokenURL:    "https://gitlab.com/oauth/token",
			UserInfoURL: "https://gitlab.com/api/v4/user",
			Scopes:      []string{"read_user", "openid", "email"},
			ParseUser:   parseGitLabUser,
		},
		"bitbucket": {
			Name:        "Bitbucket",
			AuthURL:     "https://bitbucket.org/site/oauth2/authorize",
			TokenURL:    "https://bitbucket.org/site/oauth2/access_token",
			UserInfoURL: "https://api.bitbucket.org/2.0/user",
			Scopes:      []string{"account", "email"},
			ParseUser:   parseGenericUser("bitbucket"),
		},
		"twitch": {
			Name:        "Twitch",
			AuthURL:     "https://id.twitch.tv/oauth2/authorize",
			TokenURL:    "https://id.twitch.tv/oauth2/token",
			UserInfoURL: "https://api.twitch.tv/helix/users",
			Scopes:      []string{"user:read:email"},
			ParseUser:   parseTwitchUser,
		},
		"notion": {
			Name:        "Notion",
			AuthURL:     "https://api.notion.com/v1/oauth/authorize",
			TokenURL:    "https://api.notion.com/v1/oauth/token",
			UserInfoURL: "",
			Scopes:      []string{},
			ParseUser:   parseGenericUser("notion"),
		},
		"stripe": {
			Name:        "Stripe",
			AuthURL:     "https://connect.stripe.com/oauth/authorize",
			TokenURL:    "https://connect.stripe.com/oauth/token",
			UserInfoURL: "",
			Scopes:      []string{"read_write"},
			ParseUser:   parseGenericUser("stripe"),
		},
	}
}

// ProviderDefinition is a provider template without credentials.
type ProviderDefinition struct {
	Name        string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Scopes      []string
	ParseUser   func([]byte) (*UserInfo, error)
}

// ToProvider creates a configured Provider from a definition + credentials.
func (d ProviderDefinition) ToProvider(clientID, clientSecret string) *Provider {
	return &Provider{
		Name:         d.Name,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      d.AuthURL,
		TokenURL:     d.TokenURL,
		UserInfoURL:  d.UserInfoURL,
		Scopes:       d.Scopes,
		ParseUser:    d.ParseUser,
	}
}

// ListAvailableProviders returns names of all supported providers.
func ListAvailableProviders() []string {
	defs := AllProviderDefinitions()
	names := make([]string, 0, len(defs))
	for k := range defs {
		names = append(names, k)
	}
	return names
}

// --- Parser functions ---

func parseGoogleUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "google"}, nil
}

func parseGitHubUser(body []byte) (*UserInfo, error) {
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
	return &UserInfo{ID: fmt.Sprintf("%d", u.ID), Email: u.Email, Name: name, Provider: "github"}, nil
}

func parseAppleUser(body []byte) (*UserInfo, error) {
	var u struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.Sub, Email: u.Email, Provider: "apple"}, nil
}

func parseDiscordUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Global   string `json:"global_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	name := u.Global
	if name == "" {
		name = u.Username
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: name, Provider: "discord"}, nil
}

func parseTwitterUser(body []byte) (*UserInfo, error) {
	var resp struct {
		Data struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &UserInfo{ID: resp.Data.ID, Name: resp.Data.Name, Provider: "twitter"}, nil
}

func parseMicrosoftUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"mail"`
		Name  string `json:"displayName"`
		UPN   string `json:"userPrincipalName"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	email := u.Email
	if email == "" {
		email = u.UPN
	}
	return &UserInfo{ID: u.ID, Email: email, Name: u.Name, Provider: "microsoft"}, nil
}

func parseSpotifyUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "spotify"}, nil
}

func parseGitLabUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.ID), Email: u.Email, Name: u.Name, Provider: "gitlab"}, nil
}

func parseTwitchUser(body []byte) (*UserInfo, error) {
	var resp struct {
		Data []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Data) == 0 {
		return nil, fmt.Errorf("twitch: no user data")
	}
	u := resp.Data[0]
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "twitch"}, nil
}

func parseGenericUser(provider string) func([]byte) (*UserInfo, error) {
	return func(body []byte) (*UserInfo, error) {
		var u struct {
			ID    interface{} `json:"id"`
			Sub   string      `json:"sub"`
			Email string      `json:"email"`
			Name  string      `json:"name"`
		}
		if err := json.Unmarshal(body, &u); err != nil {
			return nil, err
		}
		id := u.Sub
		if id == "" {
			id = fmt.Sprintf("%v", u.ID)
		}
		return &UserInfo{ID: id, Email: u.Email, Name: u.Name, Provider: provider}, nil
	}
}

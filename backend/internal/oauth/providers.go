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
		"amazon": {
			Name:        "Amazon",
			AuthURL:     "https://www.amazon.com/ap/oa",
			TokenURL:    "https://api.amazon.com/auth/o2/token",
			UserInfoURL: "https://api.amazon.com/user/profile",
			Scopes:      []string{"profile"},
			ParseUser:   parseAmazonUser,
		},
		"autodesk": {
			Name:        "Autodesk",
			AuthURL:     "https://developer.api.autodesk.com/authentication/v2/authorize",
			TokenURL:    "https://developer.api.autodesk.com/authentication/v2/token",
			UserInfoURL: "https://api.userprofile.autodesk.com/userinfo",
			Scopes:      []string{"data:read"},
			ParseUser:   parseGenericUser("autodesk"),
		},
		"bitly": {
			Name:        "Bitly",
			AuthURL:     "https://bitly.com/oauth/authorize",
			TokenURL:    "https://api-ssl.bitly.com/oauth/access_token",
			UserInfoURL: "https://api-ssl.bitly.com/v4/user",
			Scopes:      []string{},
			ParseUser:   parseBitlyUser,
		},
		"box": {
			Name:        "Box",
			AuthURL:     "https://account.box.com/api/oauth2/authorize",
			TokenURL:    "https://api.box.com/oauth2/token",
			UserInfoURL: "https://api.box.com/2.0/users/me",
			Scopes:      []string{},
			ParseUser:   parseBoxUser,
		},
		"dailymotion": {
			Name:        "Dailymotion",
			AuthURL:     "https://www.dailymotion.com/oauth/authorize",
			TokenURL:    "https://api.dailymotion.com/oauth/token",
			UserInfoURL: "https://api.dailymotion.com/me?fields=id,email,screenname",
			Scopes:      []string{"email", "userinfo"},
			ParseUser:   parseDailymotionUser,
		},
		"disqus": {
			Name:        "Disqus",
			AuthURL:     "https://disqus.com/api/oauth/2.0/authorize",
			TokenURL:    "https://disqus.com/api/oauth/2.0/access_token",
			UserInfoURL: "https://disqus.com/api/3.0/users/details.json",
			Scopes:      []string{"read", "email"},
			ParseUser:   parseDisqusUser,
		},
		"dropbox": {
			Name:        "Dropbox",
			AuthURL:     "https://www.dropbox.com/oauth2/authorize",
			TokenURL:    "https://api.dropboxapi.com/oauth2/token",
			UserInfoURL: "https://api.dropboxapi.com/2/users/get_current_account",
			Scopes:      []string{"account_info.read"},
			ParseUser:   parseDropboxUser,
		},
		"etsy": {
			Name:        "Etsy",
			AuthURL:     "https://www.etsy.com/oauth/connect",
			TokenURL:    "https://api.etsy.com/v3/public/oauth/token",
			UserInfoURL: "https://openapi.etsy.com/v3/application/users/me",
			Scopes:      []string{"email_r"},
			ParseUser:   parseGenericUser("etsy"),
		},
		"figma": {
			Name:        "Figma",
			AuthURL:     "https://www.figma.com/oauth",
			TokenURL:    "https://www.figma.com/api/oauth/token",
			UserInfoURL: "https://api.figma.com/v1/me",
			Scopes:      []string{"file_read"},
			ParseUser:   parseFigmaUser,
		},
		"hubspot": {
			Name:        "Hubspot",
			AuthURL:     "https://app.hubspot.com/oauth/authorize",
			TokenURL:    "https://api.hubapi.com/oauth/v1/token",
			UserInfoURL: "https://api.hubapi.com/oauth/v1/access-tokens/",
			Scopes:      []string{"oauth"},
			ParseUser:   parseHubspotUser,
		},
		"kakao": {
			Name:        "Kakao",
			AuthURL:     "https://kauth.kakao.com/oauth/authorize",
			TokenURL:    "https://kauth.kakao.com/oauth/token",
			UserInfoURL: "https://kapi.kakao.com/v2/user/me",
			Scopes:      []string{"profile_nickname", "account_email"},
			ParseUser:   parseKakaoUser,
		},
		"line": {
			Name:        "Line",
			AuthURL:     "https://access.line.me/oauth2/v2.1/authorize",
			TokenURL:    "https://api.line.me/oauth2/v2.1/token",
			UserInfoURL: "https://api.line.me/v2/profile",
			Scopes:      []string{"profile", "openid", "email"},
			ParseUser:   parseLineUser,
		},
		"mailchimp": {
			Name:        "Mailchimp",
			AuthURL:     "https://login.mailchimp.com/oauth2/authorize",
			TokenURL:    "https://login.mailchimp.com/oauth2/token",
			UserInfoURL: "https://login.mailchimp.com/oauth2/metadata",
			Scopes:      []string{},
			ParseUser:   parseMailchimpUser,
		},
		"patreon": {
			Name:        "Patreon",
			AuthURL:     "https://www.patreon.com/oauth2/authorize",
			TokenURL:    "https://www.patreon.com/api/oauth2/token",
			UserInfoURL: "https://www.patreon.com/api/oauth2/v2/identity?fields%5Buser%5D=email,full_name",
			Scopes:      []string{"identity", "identity[email]"},
			ParseUser:   parsePatreonUser,
		},
		"paypal": {
			Name:        "PayPal",
			AuthURL:     "https://www.paypal.com/signin/authorize",
			TokenURL:    "https://api.paypal.com/v1/oauth2/token",
			UserInfoURL: "https://api.paypal.com/v1/identity/oauth2/userinfo?schema=paypalv1.1",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parsePayPalUser,
		},
		"podio": {
			Name:        "Podio",
			AuthURL:     "https://podio.com/oauth/authorize",
			TokenURL:    "https://podio.com/oauth/token",
			UserInfoURL: "https://api.podio.com/user/status",
			Scopes:      []string{},
			ParseUser:   parseGenericUser("podio"),
		},
		"reddit": {
			Name:        "Reddit",
			AuthURL:     "https://www.reddit.com/api/v1/authorize",
			TokenURL:    "https://www.reddit.com/api/v1/access_token",
			UserInfoURL: "https://oauth.reddit.com/api/v1/me",
			Scopes:      []string{"identity"},
			ParseUser:   parseRedditUser,
		},
		"salesforce": {
			Name:        "Salesforce",
			AuthURL:     "https://login.salesforce.com/services/oauth2/authorize",
			TokenURL:    "https://login.salesforce.com/services/oauth2/token",
			UserInfoURL: "https://login.salesforce.com/services/oauth2/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("salesforce"),
		},
		"tradeshift": {
			Name:        "Tradeshift",
			AuthURL:     "https://go.tradeshift.com/api/oauth2/authorize",
			TokenURL:    "https://go.tradeshift.com/api/oauth2/token",
			UserInfoURL: "https://go.tradeshift.com/api/rest/external/account/info",
			Scopes:      []string{},
			ParseUser:   parseGenericUser("tradeshift"),
		},
		"wordpress": {
			Name:        "WordPress",
			AuthURL:     "https://public-api.wordpress.com/oauth2/authorize",
			TokenURL:    "https://public-api.wordpress.com/oauth2/token",
			UserInfoURL: "https://public-api.wordpress.com/rest/v1.1/me",
			Scopes:      []string{"auth"},
			ParseUser:   parseWordPressUser,
		},
		"yahoo": {
			Name:        "Yahoo",
			AuthURL:     "https://api.login.yahoo.com/oauth2/request_auth",
			TokenURL:    "https://api.login.yahoo.com/oauth2/get_token",
			UserInfoURL: "https://api.login.yahoo.com/openid/v1/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("yahoo"),
		},
		"yammer": {
			Name:        "Yammer",
			AuthURL:     "https://www.yammer.com/oauth2/authorize",
			TokenURL:    "https://www.yammer.com/oauth2/access_token",
			UserInfoURL: "https://www.yammer.com/api/v1/users/current.json",
			Scopes:      []string{},
			ParseUser:   parseYammerUser,
		},
		"yandex": {
			Name:        "Yandex",
			AuthURL:     "https://oauth.yandex.com/authorize",
			TokenURL:    "https://oauth.yandex.com/token",
			UserInfoURL: "https://login.yandex.ru/info?format=json",
			Scopes:      []string{"login:email", "login:info"},
			ParseUser:   parseYandexUser,
		},
		"zoho": {
			Name:        "Zoho",
			AuthURL:     "https://accounts.zoho.com/oauth/v2/auth",
			TokenURL:    "https://accounts.zoho.com/oauth/v2/token",
			UserInfoURL: "https://accounts.zoho.com/oauth/user/info",
			Scopes:      []string{"AaaServer.profile.Read"},
			ParseUser:   parseZohoUser,
		},
		"zoom": {
			Name:        "Zoom",
			AuthURL:     "https://zoom.us/oauth/authorize",
			TokenURL:    "https://zoom.us/oauth/token",
			UserInfoURL: "https://api.zoom.us/v2/users/me",
			Scopes:      []string{"user:read"},
			ParseUser:   parseZoomUser,
		},
	}
}

// AllProviderDefinitionsWithDomain returns all provider definitions including
// domain-dependent providers (Auth0, Okta) and fully configurable OIDC.
func AllProviderDefinitionsWithDomain(auth0Domain, oktaDomain, oidcAuthURL, oidcTokenURL, oidcUserInfoURL string) map[string]ProviderDefinition {
	defs := AllProviderDefinitions()

	if auth0Domain != "" {
		defs["auth0"] = ProviderDefinition{
			Name:        "Auth0",
			AuthURL:     "https://" + auth0Domain + "/authorize",
			TokenURL:    "https://" + auth0Domain + "/oauth/token",
			UserInfoURL: "https://" + auth0Domain + "/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("auth0"),
		}
	}

	if oktaDomain != "" {
		defs["okta"] = ProviderDefinition{
			Name:        "Okta",
			AuthURL:     "https://" + oktaDomain + "/oauth2/v1/authorize",
			TokenURL:    "https://" + oktaDomain + "/oauth2/v1/token",
			UserInfoURL: "https://" + oktaDomain + "/oauth2/v1/userinfo",
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("okta"),
		}
	}

	if oidcAuthURL != "" && oidcTokenURL != "" {
		defs["oidc"] = ProviderDefinition{
			Name:        "OIDC",
			AuthURL:     oidcAuthURL,
			TokenURL:    oidcTokenURL,
			UserInfoURL: oidcUserInfoURL,
			Scopes:      []string{"openid", "email", "profile"},
			ParseUser:   parseGenericUser("oidc"),
		}
	}

	return defs
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

func parseAmazonUser(body []byte) (*UserInfo, error) {
	var u struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.UserID, Email: u.Email, Name: u.Name, Provider: "amazon"}, nil
}

func parseBitlyUser(body []byte) (*UserInfo, error) {
	var u struct {
		Login string `json:"login"`
		Email string `json:"emails"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.Login, Email: u.Email, Name: u.Name, Provider: "bitly"}, nil
}

func parseBoxUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"login"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "box"}, nil
}

func parseDailymotionUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		ScreenName string `json:"screenname"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.ScreenName, Provider: "dailymotion"}, nil
}

func parseDisqusUser(body []byte) (*UserInfo, error) {
	var resp struct {
		Response struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &UserInfo{ID: resp.Response.ID, Email: resp.Response.Email, Name: resp.Response.Name, Provider: "disqus"}, nil
}

func parseDropboxUser(body []byte) (*UserInfo, error) {
	var u struct {
		AccountID string `json:"account_id"`
		Email     string `json:"email"`
		Name      struct {
			DisplayName string `json:"display_name"`
		} `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.AccountID, Email: u.Email, Name: u.Name.DisplayName, Provider: "dropbox"}, nil
}

func parseFigmaUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"handle"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: u.Name, Provider: "figma"}, nil
}

func parseHubspotUser(body []byte) (*UserInfo, error) {
	var u struct {
		User   string `json:"user"`
		UserID int    `json:"user_id"`
		HubID  int    `json:"hub_id"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.UserID), Email: u.User, Name: u.User, Provider: "hubspot"}, nil
}

func parseKakaoUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID      int64 `json:"id"`
		Account struct {
			Email   string `json:"email"`
			Profile struct {
				Nickname string `json:"nickname"`
			} `json:"profile"`
		} `json:"kakao_account"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.ID), Email: u.Account.Email, Name: u.Account.Profile.Nickname, Provider: "kakao"}, nil
}

func parseLineUser(body []byte) (*UserInfo, error) {
	var u struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.UserID, Name: u.DisplayName, Provider: "line"}, nil
}

func parseMailchimpUser(body []byte) (*UserInfo, error) {
	var u struct {
		UserID int `json:"user_id"`
		Login  struct {
			Email     string `json:"email"`
			LoginName string `json:"login_name"`
		} `json:"login"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.UserID), Email: u.Login.Email, Name: u.Login.LoginName, Provider: "mailchimp"}, nil
}

func parsePatreonUser(body []byte) (*UserInfo, error) {
	var resp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Email    string `json:"email"`
				FullName string `json:"full_name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &UserInfo{ID: resp.Data.ID, Email: resp.Data.Attributes.Email, Name: resp.Data.Attributes.FullName, Provider: "patreon"}, nil
}

func parsePayPalUser(body []byte) (*UserInfo, error) {
	var u struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.UserID, Email: u.Email, Name: u.Name, Provider: "paypal"}, nil
}

func parseRedditUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: u.ID, Name: u.Name, Provider: "reddit"}, nil
}

func parseWordPressUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID          int64  `json:"ID"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.ID), Email: u.Email, Name: u.DisplayName, Provider: "wordpress"}, nil
}

func parseYammerUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.ID), Email: u.Email, Name: u.FullName, Provider: "yammer"}, nil
}

func parseYandexUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID           string `json:"id"`
		DefaultEmail string `json:"default_email"`
		DisplayName  string `json:"display_name"`
		RealName     string `json:"real_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	name := u.DisplayName
	if name == "" {
		name = u.RealName
	}
	return &UserInfo{ID: u.ID, Email: u.DefaultEmail, Name: name, Provider: "yandex"}, nil
}

func parseZohoUser(body []byte) (*UserInfo, error) {
	var u struct {
		ZUID      int64  `json:"ZUID"`
		Email     string `json:"Email"`
		FirstName string `json:"First_Name"`
		LastName  string `json:"Last_Name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	name := u.FirstName
	if u.LastName != "" {
		name = name + " " + u.LastName
	}
	return &UserInfo{ID: fmt.Sprintf("%d", u.ZUID), Email: u.Email, Name: name, Provider: "zoho"}, nil
}

func parseZoomUser(body []byte) (*UserInfo, error) {
	var u struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		First string `json:"first_name"`
		Last  string `json:"last_name"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	name := u.First
	if u.Last != "" {
		name = name + " " + u.Last
	}
	return &UserInfo{ID: u.ID, Email: u.Email, Name: name, Provider: "zoom"}, nil
}

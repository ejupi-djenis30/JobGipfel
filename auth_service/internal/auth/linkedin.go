package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// LinkedInEndpoint is the OAuth2 endpoint for LinkedIn.
var LinkedInEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
	TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
}

// LinkedInUserInfo represents the user info from LinkedIn.
type LinkedInUserInfo struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Picture   string `json:"picture"`
}

// LinkedInEmailResponse represents the email response from LinkedIn.
type LinkedInEmailResponse struct {
	Elements []struct {
		Handle struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"handle~"`
	} `json:"elements"`
}

// LinkedInProfileResponse represents the profile response from LinkedIn API v2.
type LinkedInProfileResponse struct {
	Sub           string `json:"sub"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
}

// LinkedInProvider handles LinkedIn OAuth2 authentication.
type LinkedInProvider struct {
	config *oauth2.Config
}

// NewLinkedInProvider creates a new LinkedIn OAuth provider.
func NewLinkedInProvider(clientID, clientSecret, redirectURL string) *LinkedInProvider {
	return &LinkedInProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"openid",
				"profile",
				"email",
			},
			Endpoint: LinkedInEndpoint,
		},
	}
}

// GetAuthURL returns the URL to redirect the user to for authentication.
func (p *LinkedInProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for tokens.
func (p *LinkedInProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	// LinkedIn requires form-encoded body for token exchange
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", p.config.RedirectURL)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LinkedIn token error: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
		IDToken     string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
	}, nil
}

// GetUserInfo retrieves the user info from LinkedIn using OpenID Connect.
func (p *LinkedInProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*LinkedInUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LinkedIn API error: %s - %s", resp.Status, string(body))
	}

	var profile LinkedInProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	return &LinkedInUserInfo{
		ID:        profile.Sub,
		FirstName: profile.GivenName,
		LastName:  profile.FamilyName,
		Email:     profile.Email,
		Picture:   profile.Picture,
	}, nil
}

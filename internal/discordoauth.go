package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// TokenResponse represents the structure of the JSON response from Discord's token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// DiscordUser represents basic user information from Discord's /users/@me endpoint
type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"` // Only present if 'email' scope is requested
}

// In-memory storage for OAuth2 states. In a production application,
// this should be a secure, persistent storage (e.g., database, secure session store)
// associated with the user's session to prevent CSRF attacks.
var oauthStates = make(map[string]bool)
var mu sync.Mutex // Mutex to protect access to oauthStates

func DiscordOAuthLogin(a *Api) (string, error) {
	// Generate a cryptographically secure random state string for CSRF protection
	state, err := GenerateRandomState()
	if err != nil {
		// http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		log.Err(err).Msg("error generating state.")
		return "", err
	}

	mu.Lock()
	oauthStates[state] = true
	mu.Unlock()

	params := url.Values{}
	params.Add("client_id", Config.DiscordConfig.ClientID)
	params.Add("redirect_uri", Config.DiscordConfig.RedirectURI)
	params.Add("response_type", "code")              // For Authorization Code Grant flow
	params.Add("scope", Config.DiscordConfig.Scopes) // Requested permissions
	params.Add("state", state)                       // CSRF protection
	params.Add("prompt", "none")                     // if already authorized skip the screen if a user selects oauth again

	// create proper authorization redirection url
	// https://discord.com/developers/docs/topics/oauth2#authorization-code-grant-authorization-url-example
	authURL := fmt.Sprintf("%s?%s", Config.DiscordConfig.AuthURL, params.Encode())
	return authURL, nil
}

func DiscordOAuthCallback(c *gin.Context, a *Api) (*Player, error) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDescription := c.Query("error_description")

	if errorParam != "" {
		c.String(http.StatusBadRequest, "Discord authorization error: %s - %s", errorParam, errorDescription)
		log.Printf("Discord authorization error: %s - %s", errorParam, errorDescription)
		return nil, fmt.Errorf(errorParam)
	}

	if code == "" {
		c.String(http.StatusBadRequest, "Authorization code not found in callback")
		log.Error().Msg("Authorization code missing from callback.")
		return nil, fmt.Errorf("authorization code missing from callback")
	}

	mu.Lock()
	stateValid := oauthStates[state]
	delete(oauthStates, state)
	mu.Unlock()

	// Validate the 'state' parameter to prevent CSRF attacks
	if !stateValid {
		c.String(http.StatusUnauthorized, "Invalid or missing state parameter. Possible CSRF attack.")
		log.Error().Str("state", state).Msg("Invalid or missing state parameter")
		return nil, fmt.Errorf("invalid or missing state parameter")
	}
	log.Info().Str("code", code).Msg("Received authorization code")

	tokenResponse, err := exchangeCodeForToken(code)
	if err != nil {
		log.Err(err).Msg("Error exchanging code for token")
		return nil, err
	}
	log.Info().Msg("Successfully obtained access token.")

	// Use the access oken to fetch user information
	discordUser, err := fetchDiscordUserInfo(tokenResponse.AccessToken, tokenResponse.TokenType)
	if err != nil {
		c.String(http.StatusUnauthorized, "Invalid or missing state parameter. Possible CSRF attack.")
		log.Err(err).Msg("Error fetching Discord user info")
		return nil, err
	}
	log.Info().Any("discordUser", discordUser).Msg("successfully grad discord user information")

	sessionId, err := GenerateRandomState()
	if err != nil {
		log.Err(err).Msg("failed to generate random state")
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(state), bcrypt.DefaultCost)
	if err != nil {
		log.Err(err).Msg("failed to hash fake user password for discord oauth workflow")
		return nil, err
	}
	player := &Player{
		Name:             discordUser.Username,
		Password:         string(hashedPassword),
		DiscordID:        discordUser.ID,
		VerificationCode: uuid.NewString(),
		OAuthToken:       tokenResponse.AccessToken,
		SessionId:        sessionId,
	}
	_, err = a.db.NewInsert().
		Model(player).
		On("CONFLICT (discord_id) DO UPDATE").
		Set("oauth_token = EXCLUDED.oauth_token").
		Set("session_id = EXCLUDED.session_id").
		Exec(a.ctx)
	if err != nil {
		log.Err(err).Msg("failed to create or update user from discord oauth workflow")
		return nil, err
	}
	log.Info().Any("player", player).Msg("successfully completed the oauth2 discord login and create/update the user information")

	return player, nil
}

// exchangeCodeForToken makes a POST request to Discord's token endpoint.
func exchangeCodeForToken(code string) (*TokenResponse, error) {
	// Prepare the request body as application/x-www-form-urlencoded
	data := url.Values{}
	data.Add("client_id", Config.DiscordConfig.ClientID)
	data.Add("client_secret", Config.DiscordConfig.ClientSecret)
	data.Add("grant_type", "authorization_code")
	data.Add("code", code)
	data.Add("redirect_uri", Config.DiscordConfig.RedirectURI)

	req, err := http.NewRequest(http.MethodPost, Config.DiscordConfig.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// fetchDiscordUserInfo makes a GET request to Discord's user info endpoint.
func fetchDiscordUserInfo(accessToken, tokenType string) (*DiscordUser, error) {
	req, err := http.NewRequest(http.MethodGet, Config.DiscordConfig.UserAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	// Set the Authorization header with the access token [6, 12]
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", tokenType, accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send user info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var discordUser DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %w", err)
	}

	return &discordUser, nil
}

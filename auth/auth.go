package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mpapenbr/irdata/log"
)

type (
	//nolint:tagliatelle // external definition
	tokenData struct {
		AccessToken           string `json:"access_token"`
		TokenType             string `json:"token_type"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	AuthConfig struct {
		ClientID     string
		ClientSecret string
		Username     string
		Password     string
		AuthFile     string
	}

	Option             func(*tokenManagerConfig)
	tokenManagerConfig struct {
		authConfig *AuthConfig
	}
	TokenManager struct {
		cfg   tokenManagerConfig
		ctx   context.Context
		token *tokenData
	}
)

const (
	//nolint:gosec // this is the official iRacing API endpoint
	tokenURL = "https://oauth.iracing.com/oauth2/token"
)

func NewTokenManager(opts ...Option) (*TokenManager, error) {
	tm := tokenManagerConfig{}
	for _, opt := range opts {
		opt(&tm)
	}
	return &TokenManager{cfg: tm, ctx: context.Background()}, nil
}

func WithAuthConfig(authConfig *AuthConfig) Option {
	return func(tm *tokenManagerConfig) {
		tm.authConfig = authConfig
	}
}

func (tm *TokenManager) Login() error {
	if tm.cfg.authConfig.AuthFile != "" {
		log.Debug("auth file path provided, trying to load auth info from file",
			log.String("auth-file", tm.cfg.authConfig.AuthFile))
		if err := tm.loadAuthInfo(); err != nil {
			log.Debug("failed to load auth info from file, will try to login",
				log.ErrorField(err))
			return tm.doLogin()
		}
		log.Info("successfully loaded auth info from file")
		exp := tm.getExpiresIn(tm.token.AccessToken)
		if exp.After(time.Now()) {
			log.Debug("token is valid")
			tm.setupTokenRefresh()
			return nil
		}
		log.Debug("token is expired, refreshing...")

		exp = tm.getExpiresIn(tm.token.RefreshToken)
		fmt.Printf("%s\n", exp.String())
		if exp.After(time.Now()) {
			log.Debug("refresh token is valid, refreshing access token...")
			tm.doRefresh()
			tm.setupTokenRefresh()
			return nil
		}
		// do nothing, will try to login with credentials
		log.Debug("refresh token is expired, will try to login with credentials")
	}
	return tm.doLogin()
}

func (tm *TokenManager) GetAccessToken() (string, error) {
	if tm.token == nil {
		return "", fmt.Errorf("not logged in")
	}
	return tm.token.AccessToken, nil
}

func (tm *TokenManager) getExpiresIn(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Debug("invalid token format")
		return time.Time{}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Debug("failed to decode token payload", log.ErrorField(err))
		return time.Time{}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		log.Debug("failed to unmarshal token claims", log.ErrorField(err))
		return time.Time{}
	}

	if exp, ok := claims["exp"].(float64); ok {
		log.Debug("token expiration", log.Any("exp_time", time.Unix(int64(exp), 0)))
		return time.Unix(int64(exp), 0)
	}
	return time.Time{}
}

func (tm *TokenManager) setupTokenRefresh() {
	go func() {
		for {
			if tm.token == nil {
				time.Sleep(time.Second)
				continue
			}

			exp := tm.getExpiresIn(tm.token.AccessToken)
			waitUntil := exp.Add(-10 * time.Second)
			duration := time.Until(waitUntil)

			if duration > 0 {
				log.Debug("waiting until token refresh",
					log.Duration("refresh_in", duration))
				time.Sleep(duration)
			}
			log.Debug("refreshing token...")
			if err := tm.doRefresh(); err != nil {
				log.Debug("token refresh failed", log.ErrorField(err))
			}
		}
	}()
}

func (tm *TokenManager) doLogin() error {
	client := &http.Client{}
	data := url.Values{}
	data.Set("username", tm.cfg.authConfig.Username)
	data.Set("password",
		tm.hashSecret(tm.cfg.authConfig.Password, tm.cfg.authConfig.Username))
	data.Set("client_id", tm.cfg.authConfig.ClientID)
	data.Set("client_secret",
		tm.hashSecret(tm.cfg.authConfig.ClientSecret, tm.cfg.authConfig.ClientID))
	data.Set("grant_type", "password_limited")
	data.Set("scope", "iracing.auth")

	req, err := http.NewRequestWithContext(
		tm.ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token tokenData
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	tm.token = &token

	if tm.cfg.authConfig.AuthFile != "" {
		log.Debug("saving auth info to file",
			log.String("auth-file", tm.cfg.authConfig.AuthFile))
		if err := tm.saveAuthInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TokenManager) doRefresh() error {
	client := &http.Client{}
	data := url.Values{}
	data.Set("refresh_token", tm.token.RefreshToken)
	data.Set("client_id", tm.cfg.authConfig.ClientID)
	data.Set("client_secret",
		tm.hashSecret(tm.cfg.authConfig.ClientSecret, tm.cfg.authConfig.ClientID))
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(
		tm.ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var token tokenData
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}
	tm.token = &token

	if tm.cfg.authConfig.AuthFile != "" {
		if err := tm.saveAuthInfo(); err != nil {
			return err
		}
	}
	return nil
}

func (tm *TokenManager) hashSecret(secret, id string) string {
	h := sha256.New()
	h.Write([]byte(secret + strings.ToLower(id)))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (tm *TokenManager) saveAuthInfo() error {
	data, err := json.Marshal(tm.token)
	if err != nil {
		return err
	}
	return os.WriteFile(tm.cfg.authConfig.AuthFile, data, 0o600)
}

func (tm *TokenManager) loadAuthInfo() error {
	data, err := os.ReadFile(tm.cfg.authConfig.AuthFile)
	if err != nil {
		return err
	}
	token := &tokenData{}
	if err := json.Unmarshal(data, token); err != nil {
		return err
	}
	tm.token = token
	return nil
}

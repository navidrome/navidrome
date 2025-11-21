package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
)

type jwtToken struct {
	token     string
	expiresAt time.Time
	mu        sync.RWMutex
}

func (j *jwtToken) get() (string, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if time.Now().Before(j.expiresAt) {
		return j.token, true
	}
	return "", false
}

func (j *jwtToken) set(token string, expiresIn time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.token = token
	j.expiresAt = time.Now().Add(expiresIn)
}

func (c *client) getJWT(ctx context.Context) (string, error) {
	// Check if we have a valid cached token
	if token, valid := c.jwt.get(); valid {
		return token, nil
	}

	// Fetch a new anonymous token
	req, err := http.NewRequestWithContext(ctx, "GET", authBaseURL+"/login/anonymous?jo=p&rto=c", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpDoer.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("deezer: failed to get JWT token: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type authResponse struct {
		JWT          string `json:"jwt"`
		RefreshToken string `json:"refresh_token"`
	}

	var result authResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("deezer: failed to parse auth response: %w", err)
	}

	if result.JWT == "" {
		return "", errors.New("deezer: no JWT token in response")
	}
	// Cache the token for 50 minutes (tokens expire in 1 hour).
	// The 10-minute buffer helps handle clock skew, network delays, or timing issues,
	// ensuring we refresh the token before it actually expires.
	// Note: c.jwt is assumed to be thread-safe.
	// Cache the token for 50 minutes (tokens expire in 1 hour)
	c.jwt.set(result.JWT, 50*time.Minute)
	log.Trace(ctx, "Fetched new Deezer JWT token")

	return result.JWT, nil
}

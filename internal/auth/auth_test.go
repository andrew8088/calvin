package auth

import (
	"context"
	"testing"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"golang.org/x/oauth2"
)

func TestTokenSourcePersistsRefreshedToken(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	initial := &oauth2.Token{
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		Expiry:       time.Unix(100, 0),
	}
	if err := saveToken(initial); err != nil {
		t.Fatalf("saveToken() initial: %v", err)
	}

	refreshed := &oauth2.Token{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		Expiry:       time.Unix(200, 0),
	}

	originalFactory := oauthTokenSource
	oauthTokenSource = func(_ context.Context, _ *oauth2.Config, _ *oauth2.Token) oauth2.TokenSource {
		return staticTokenSource{token: refreshed}
	}
	t.Cleanup(func() {
		oauthTokenSource = originalFactory
	})

	cfg := config.Default()
	cfg.OAuthClientID = "client-id"
	cfg.OAuthClientSecret = "client-secret"

	ts, err := TokenSource(cfg)
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	got, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got.AccessToken != refreshed.AccessToken {
		t.Fatalf("Token().AccessToken = %q, want %q", got.AccessToken, refreshed.AccessToken)
	}

	persisted, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if persisted.AccessToken != refreshed.AccessToken {
		t.Fatalf("persisted access token = %q, want %q", persisted.AccessToken, refreshed.AccessToken)
	}
	if persisted.RefreshToken != refreshed.RefreshToken {
		t.Fatalf("persisted refresh token = %q, want %q", persisted.RefreshToken, refreshed.RefreshToken)
	}
	if !persisted.Expiry.Equal(refreshed.Expiry) {
		t.Fatalf("persisted expiry = %v, want %v", persisted.Expiry, refreshed.Expiry)
	}
}

type staticTokenSource struct {
	token *oauth2.Token
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return s.token, nil
}

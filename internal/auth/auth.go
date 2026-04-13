package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/andrew8088/calvin/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googlecalendar "google.golang.org/api/calendar/v3"
)

var (
	embeddedClientID     = ""
	embeddedClientSecret = ""
)

func oauthConfig(cfg *config.Config) (*oauth2.Config, error) {
	clientID := embeddedClientID
	clientSecret := embeddedClientSecret
	if cfg.OAuthClientID != "" {
		clientID = cfg.OAuthClientID
	}
	if cfg.OAuthClientSecret != "" {
		clientSecret = cfg.OAuthClientSecret
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("no OAuth credentials configured. Set oauth_client_id and oauth_client_secret in config.toml")
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{googlecalendar.CalendarReadonlyScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  fmt.Sprintf("http://localhost:%d/callback", cfg.AuthPort),
	}, nil
}

func RunFlow(cfg *config.Config) error {
	oc, err := oauthConfig(cfg)
	if err != nil {
		return err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			fmt.Fprint(w, "Error: no authorization code received. Close this tab and try again.")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authenticated! You can close this tab and return to your terminal.")
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.AuthPort))
	if err != nil {
		return fmt.Errorf("starting auth server on port %d: %w", cfg.AuthPort, err)
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	url := oc.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Opening browser for Google Calendar authorization...\n")
	fmt.Printf("If your browser doesn't open, visit:\n  %s\n\n", url)

	openBrowser(url)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authorization timed out after 5 minutes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := oc.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchanging auth code for token: %w", err)
	}

	if err := saveToken(token); err != nil {
		return err
	}

	fmt.Println("Authenticated! Token saved.")
	return nil
}

func Revoke() error {
	path := config.TokenPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No credentials found.")
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing token: %w", err)
	}
	fmt.Println("Credentials revoked.")
	return nil
}

func LoadToken() (*oauth2.Token, error) {
	path := config.TokenPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading token: %w", err)
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	return &token, nil
}

func TokenSource(cfg *config.Config) (oauth2.TokenSource, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, err
	}
	oc, err := oauthConfig(cfg)
	if err != nil {
		return nil, err
	}
	ts := oc.TokenSource(context.Background(), token)
	return ts, nil
}

func HasToken() bool {
	_, err := os.Stat(config.TokenPath())
	return err == nil
}

func CheckTokenValid(cfg *config.Config) error {
	ts, err := TokenSource(cfg)
	if err != nil {
		return err
	}
	_, err = ts.Token()
	if err != nil {
		return fmt.Errorf("token invalid: %w", err)
	}
	return nil
}

func saveToken(token *oauth2.Token) error {
	path := config.TokenPath()
	dir := config.DataDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling token: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing token: %w", err)
	}
	return nil
}

func openBrowser(url string) {
	// macOS
	cmd := "open"
	args := []string{url}
	p, err := os.StartProcess("/usr/bin/"+cmd, append([]string{cmd}, args...), &os.ProcAttr{
		Dir:   "/",
		Files: []*os.File{os.Stdin, nil, nil},
	})
	if err == nil {
		p.Release()
	}
}

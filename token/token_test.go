package token

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rivo/tview"
)

func signed(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("any"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseClaims(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	good, err := GenerateToken(time.Hour, &TokenFields{Id: "x", Email: "a@example.com"}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	// GenerateToken stamps the real time; parse it against the real clock.
	if c, err := ParseClaims(good, time.Now()); err != nil || c.Email != "a@example.com" {
		t.Errorf("cheat-code token: %+v, %v", c, err)
	}

	cases := []struct {
		name, tok, wantErr string
	}{
		{"empty", "  ", "enter a token"},
		{"not a JWT", "abc", "not a JWT"},
		{"no email", signed(t, jwt.MapClaims{"exp": now.Add(time.Hour).Unix()}), "no email"},
		{"no exp", signed(t, jwt.MapClaims{"email": "a@example.com"}), "no exp"},
		{"expired", signed(t, jwt.MapClaims{"email": "a@example.com", "exp": now.Add(-time.Second).Unix()}), "expired"},
		{"exp not a number", signed(t, jwt.MapClaims{"email": "a@example.com", "exp": "soon"}), "isn't a number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseClaims(tc.tok, now)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("got %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}

	// Any signature is accepted locally: the backend is the verifier.
	tok := signed(t, jwt.MapClaims{"email": "b@example.com", "exp": now.Add(time.Minute).Unix()})
	if c, err := ParseClaims(tok, now); err != nil || c.Email != "b@example.com" || !c.Expires.Equal(now.Add(time.Minute)) {
		t.Errorf("got %+v, %v", c, err)
	}
}

func pingServer(t *testing.T, status int) (*api.Client, *int) {
	t.Helper()
	pings := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/authping" {
			t.Errorf("unexpected %s", r.URL.Path)
		}
		pings++
		if status != 200 {
			http.Error(w, "Invalid token", status)
		}
	}))
	t.Cleanup(srv.Close)
	return &api.Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}, &pings
}

func TestLogin(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := signed(t, jwt.MapClaims{"email": "a@example.com", "exp": now.Add(time.Hour).Unix()})

	c, pings := pingServer(t, 200)
	if claims, err := Login(tok, c, now); err != nil || claims.Email != "a@example.com" || *pings != 1 {
		t.Errorf("valid: %+v, %v, %d pings", claims, err, *pings)
	}

	t.Setenv(DevSecretEnv, "")
	c, _ = pingServer(t, 403)
	_, err := Login(tok, c, now)
	if err == nil || !strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), DevSecretEnv) {
		t.Errorf("403 without cheat code: %v", err)
	}

	t.Setenv(DevSecretEnv, "s3cret")
	c, _ = pingServer(t, 403)
	if _, err := Login(tok, c, now); err == nil || !strings.Contains(err.Error(), DevSecretEnv) {
		t.Errorf("403 with cheat code should mention %s: %v", DevSecretEnv, err)
	}

	c, pings = pingServer(t, 200)
	if _, err := Login(signed(t, jwt.MapClaims{"email": "a@example.com"}), c, now); err == nil || *pings != 0 {
		t.Errorf("a token without exp must be refused before pinging: %v, %d pings", err, *pings)
	}
}

func TestCheatCodeOnlyWithDevSecret(t *testing.T) {
	var tok string
	t.Setenv(DevSecretEnv, "")
	form := GetTokenInputForm(ui.NewShell(tview.NewApplication()), &tok, func(string) {})
	if form.GetButtonIndex(CheatCodeLabel) >= 0 {
		t.Error("cheat code shown without " + DevSecretEnv)
	}

	t.Setenv(DevSecretEnv, "s3cret")
	form = GetTokenInputForm(ui.NewShell(tview.NewApplication()), &tok, func(string) {})
	if form.GetButtonIndex(CheatCodeLabel) < 0 {
		t.Error("cheat code hidden although " + DevSecretEnv + " is set")
	}
}

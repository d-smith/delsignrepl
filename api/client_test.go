package api

// Fake servers here are written from docs/delsign-api/openapi.yaml (status codes,
// shapes, examples). They don't use any backend code.

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type seen struct {
	method, path, rawPath, auth, contentType, body string
}

// fake serves one canned response and records the request it got.
func fake(t *testing.T, status int, body string) (*Client, *seen) {
	t.Helper()
	got := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = seen{r.Method, r.URL.Path, r.URL.RawPath, r.Header.Get("Authorization"),
			r.Header.Get("Content-Type"), string(b)}
		if status >= 400 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}, got
}

func TestBaseURLFromEnv(t *testing.T) {
	t.Setenv("DELSIGN_API_URL", "")
	if got := BaseURLFromEnv(); got != DefaultBaseURL {
		t.Errorf("default: got %q", got)
	}
	t.Setenv("DELSIGN_API_URL", " http://example.test:9000/ ")
	if got := BaseURLFromEnv(); got != "http://example.test:9000" {
		t.Errorf("override: got %q", got)
	}
}

func TestEveryCallSendsTokenAndContentTypeOnlyWithBody(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		resp     string
		call     func(c *Client) error
		method   string
		path     string
		wantBody bool
	}{
		{"authping", 200, "", func(c *Client) error { return c.AuthPing() }, "GET", "/api/v1/authping", false},
		{"getUser", 200, `{"email":"a@example.com","pubkey":"ab"}`,
			func(c *Client) error { _, err := c.GetUser("a@example.com"); return err }, "GET", "/api/v1/users/a@example.com", false},
		{"keyreg", 200, "", func(c *Client) error {
			return c.RegisterKey(KeyReg{Email: "a@example.com", PubKey: "ab", SignatureForRegistration: "cd"})
		}, "POST", "/api/v1/keyreg", true},
		{"createWallet", 201, `{"id":2}`, func(c *Client) error { _, err := c.CreateWallet(); return err }, "POST", "/api/v1/wallets", false},
		{"listWallets", 200, `[2,5]`, func(c *Client) error { _, err := c.ListWallets(); return err }, "GET", "/api/v1/wallets", false},
		{"createAddress", 201, `{"eoa":"0x17155B71982feD535Fd6aeB084C9330e7d5C290f"}`,
			func(c *Client) error { _, err := c.CreateAddress(2); return err }, "POST", "/api/v1/wallets/2/addresses", false},
		{"walletctx", 200, `[]`, func(c *Client) error { _, err := c.WalletContext(); return err }, "GET", "/api/v1/walletctx", false},
		{"balance", 200, `{"address":"0x17155B71982feD535Fd6aeB084C9330e7d5C290f","amount":"1000000000000000000"}`,
			func(c *Client) error { _, err := c.Balance("0x17155B71982feD535Fd6aeB084C9330e7d5C290f"); return err },
			"GET", "/api/v1/wallets/balance/0x17155B71982feD535Fd6aeB084C9330e7d5C290f", false},
		{"send", 200, `{"txnid":"0x45d1e539f4bf4b3304730d126fef71bbaf7240d3f828d80736f3059c1944fe66"}`,
			func(c *Client) error { _, err := c.Send([]byte(`{"x":1}`)); return err }, "POST", "/api/v1/wallets/send", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, got := fake(t, tc.status, tc.resp)
			if err := tc.call(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.method != tc.method || got.path != tc.path {
				t.Errorf("request: got %s %s, want %s %s", got.method, got.path, tc.method, tc.path)
			}
			if got.auth != "Bearer tok" {
				t.Errorf("Authorization: got %q", got.auth)
			}
			if tc.wantBody && got.contentType != "application/json" {
				t.Errorf("Content-Type with a body: got %q", got.contentType)
			}
			if !tc.wantBody && (got.contentType != "" || got.body != "") {
				t.Errorf("bodyless call sent Content-Type %q, body %q", got.contentType, got.body)
			}
		})
	}
}

func TestResponsesDecode(t *testing.T) {
	c, _ := fake(t, 200, `[2, 5]`)
	if ids, err := c.ListWallets(); err != nil || len(ids) != 2 || ids[0] != 2 || ids[1] != 5 {
		t.Errorf("ListWallets: %v, %v", ids, err)
	}

	c, _ = fake(t, 200, `[{"walletId":2,"address":"0x17155B71982feD535Fd6aeB084C9330e7d5C290f"}]`)
	if pairs, err := c.WalletContext(); err != nil || len(pairs) != 1 || pairs[0].WalletId != 2 {
		t.Errorf("WalletContext: %v, %v", pairs, err)
	}

	c, _ = fake(t, 200, `{"address":"0x17155B71982feD535Fd6aeB084C9330e7d5C290f","amount":"123456789012345678901234567890"}`)
	if b, err := c.Balance("0x17155b71982fed535fd6aeb084c9330e7d5c290f"); err != nil || b.Amount != "123456789012345678901234567890" {
		t.Errorf("Balance: %+v, %v", b, err)
	}

	c, _ = fake(t, 201, `{"id":7}`)
	if id, err := c.CreateWallet(); err != nil || id != 7 {
		t.Errorf("CreateWallet: %d, %v", id, err)
	}
}

// Old client code expected 201 from GET /wallets; the contract says 200.
func TestListWalletsAccepts200(t *testing.T) {
	c, _ := fake(t, http.StatusOK, `[3]`)
	if ids, err := c.ListWallets(); err != nil || len(ids) != 1 {
		t.Fatalf("got %v, %v", ids, err)
	}
}

func TestEmptyListsAreEmpty(t *testing.T) {
	for _, body := range []string{`[]`, `null`} {
		c, _ := fake(t, 200, body)
		if ids, err := c.ListWallets(); err != nil || len(ids) != 0 {
			t.Errorf("ListWallets(%s): %v, %v", body, ids, err)
		}
		c, _ = fake(t, 200, body)
		if pairs, err := c.WalletContext(); err != nil || len(pairs) != 0 {
			t.Errorf("WalletContext(%s): %v, %v", body, pairs, err)
		}
	}
}

func TestNon2xxIsErrorWithTrimmedText(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 413, 415, 500, 502} {
		c, _ := fake(t, status, "Source address is not yours\n")
		_, err := c.CreateAddress(9)
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("%d: want *Error, got %T %v", status, err, err)
		}
		if e.Status != status || e.Text != "Source address is not yours" {
			t.Errorf("%d: got status %d text %q", status, e.Status, e.Text)
		}
		if StatusOf(err) != status {
			t.Errorf("%d: StatusOf = %d", status, StatusOf(err))
		}
	}
}

func TestGetUserEscapesEmail(t *testing.T) {
	c, got := fake(t, 200, `{"email":"a/b@example.com","pubkey":"ab"}`)
	if _, err := c.GetUser("a/b@example.com"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/api/v1/users/a/b@example.com" || got.rawPath != "/api/v1/users/a%2Fb@example.com" {
		t.Errorf("path %q raw %q", got.path, got.rawPath)
	}
}

func TestSendSendsExactBytes(t *testing.T) {
	body := `{"source":"0xAbC","amount":"1000000000000000","expiry":1800000060}`
	c, got := fake(t, 200, `{"txnid":"0x45d1e539f4bf4b3304730d126fef71bbaf7240d3f828d80736f3059c1944fe66"}`)
	r, err := c.Send([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if got.body != body {
		t.Errorf("body changed in transit:\n got  %s\n want %s", got.body, body)
	}
	if r.Txnid != "0x45d1e539f4bf4b3304730d126fef71bbaf7240d3f828d80736f3059c1944fe66" {
		t.Errorf("txnid %q", r.Txnid)
	}
}

func TestTimeoutIsNetworkError(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(func() { close(release); srv.Close() })

	c := &Client{BaseURL: srv.URL, Token: "tok", HTTP: &http.Client{Timeout: 50 * time.Millisecond}}
	err := c.AuthPing()
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("want *NetworkError, got %T %v", err, err)
	}
	if StatusOf(err) != 0 {
		t.Errorf("a timeout has no status, got %d", StatusOf(err))
	}
}

func TestUnreachableIsNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	c := &Client{BaseURL: url, Token: "tok"}
	var netErr *NetworkError
	if err := c.AuthPing(); !errors.As(err, &netErr) {
		t.Fatalf("want *NetworkError, got %T %v", err, err)
	}
}

func TestDefaultClientHasTimeout(t *testing.T) {
	if c := New("tok"); c.HTTP == nil || c.HTTP.Timeout != DefaultTimeout {
		t.Errorf("New() client must have a %v timeout", DefaultTimeout)
	}
}

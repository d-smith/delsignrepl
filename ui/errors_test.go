package ui

import (
	"delsignrepl/api"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDescribe(t *testing.T) {
	okPing := func() error { return nil }
	badToken := func() error { return &api.Error{Status: 403, Text: "Invalid token"} }
	downPing := func() error { return &api.NetworkError{Err: errors.New("refused")} }
	failPing := func() error { t.Error("authping must only be called for a 403"); return nil }

	cases := []struct {
		name string
		err  error
		ping func() error
		want string // substring
	}{
		{"network", &api.NetworkError{Err: errors.New("connection refused")}, failPing, "Can't reach the server: connection refused"},
		{"wrapped network", fmt.Errorf("x: %w", &api.NetworkError{Err: errors.New("timeout")}), failPing, "Can't reach the server"},
		{"local error", errors.New("no signing key"), failPing, "no signing key"},
		{"400", &api.Error{Status: 400, Text: "request expired"}, failPing, "Request rejected: request expired"},
		{"401", &api.Error{Status: 401, Text: "Invalid signature"}, failPing, "local key may not be the registered one"},
		{"403 token", &api.Error{Status: 403, Text: "Invalid token"}, badToken, "Log in again"},
		{"403 permission", &api.Error{Status: 403, Text: "Wallet is not yours"}, okPing, "Not permitted: Wallet is not yours"},
		{"403 ping down", &api.Error{Status: 403, Text: "Wallet is not yours"}, downPing, "couldn't check the token"},
		{"404", &api.Error{Status: 404, Text: "No key registered"}, failPing, "Not found: No key registered"},
		{"409", &api.Error{Status: 409, Text: "Request already processed"}, failPing, "Conflict: Request already processed"},
		{"415", &api.Error{Status: 415, Text: "Content-Type header is not application/json"}, failPing, "bug in the app"},
		{"500", &api.Error{Status: 500, Text: "Internal Server Error"}, failPing, "Server error"},
		{"502", &api.Error{Status: 502, Text: "insufficient funds for gas * price + value"}, failPing,
			"Ethereum node error: insufficient funds for gas * price + value"},
		{"empty text", &api.Error{Status: 400}, failPing, "Request rejected: bad request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describe(tc.err, tc.ping); !strings.Contains(got, tc.want) {
				t.Errorf("got %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

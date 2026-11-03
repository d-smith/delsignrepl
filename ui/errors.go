// Package ui holds TUI helpers shared by the app's screens.
package ui

import (
	"delsignrepl/api"
	"errors"
	"fmt"
)

// LoginPage is the name of the token entry page. A token problem routes the user there.
const LoginPage = "Login"

// Describe turns an error from an API call into a message for the user, following the
// error table in client-guide.md. It switches on the status code, never on the server's
// text. A 403 is checked against GET /authping to tell an expired or invalid token apart
// from a permission problem.
func Describe(err error, token string) string {
	msg, _ := describeErr(err, func() error { return api.New(token).AuthPing() })
	return msg
}

func describe(err error, ping func() error) string {
	msg, _ := describeErr(err, ping)
	return msg
}

// describeErr also reports whether the problem is the token itself (log in again).
func describeErr(err error, ping func() error) (msg string, tokenProblem bool) {
	if err == nil {
		return "", false
	}
	var netErr *api.NetworkError
	if errors.As(err, &netErr) {
		return "Can't reach the server: " + netErr.Err.Error(), false
	}
	var e *api.Error
	if !errors.As(err, &e) {
		return err.Error(), false // a local error, not from the server
	}
	switch {
	case e.Status == 400:
		return "Request rejected: " + textOr(e, "bad request"), false
	case e.Status == 401:
		return "Signature rejected: " + textOr(e, "invalid signature") +
			". Your local key may not be the registered one.", false
	case e.Status == 403:
		pingErr := ping()
		switch {
		case api.StatusOf(pingErr) == 403:
			return "Your session has expired or the token is invalid. Log in again with a new token (t).", true
		case pingErr != nil:
			return "Not permitted: " + textOr(e, "forbidden") +
				fmt.Sprintf(" (couldn't check the token: %v)", pingErr), false
		default:
			return "Not permitted: " + textOr(e, "forbidden"), false
		}
	case e.Status == 404:
		return "Not found: " + textOr(e, "not found"), false
	case e.Status == 409:
		return "Conflict: " + textOr(e, "conflict"), false
	case e.Status == 413 || e.Status == 415:
		return fmt.Sprintf("Client error %d: %s. This is a bug in the app.", e.Status, textOr(e, "rejected")), false
	case e.Status == 502:
		return "Ethereum node error: " + textOr(e, "node unavailable"), false
	case e.Status >= 500:
		return "Server error. Try again later.", false
	default:
		return e.Error(), false
	}
}

func textOr(e *api.Error, fallback string) string {
	if e.Text == "" {
		return fallback
	}
	return e.Text
}

// Package ui holds TUI helpers shared by the app's screens.
package ui

import (
	"delsignrepl/api"
	"errors"
	"fmt"

	"github.com/rivo/tview"
)

// Describe turns an error from an API call into a message for the user, following the
// error table in client-guide.md. It switches on the status code, never on the server's
// text. A 403 is checked against GET /authping to tell an expired or invalid token apart
// from a permission problem.
func Describe(err error, token string) string {
	return describe(err, func() error { return api.New(token).AuthPing() })
}

func describe(err error, ping func() error) string {
	if err == nil {
		return ""
	}
	var netErr *api.NetworkError
	if errors.As(err, &netErr) {
		return "Can't reach the server: " + netErr.Err.Error()
	}
	var e *api.Error
	if !errors.As(err, &e) {
		return err.Error() // a local error, not from the server
	}
	switch {
	case e.Status == 400:
		return "Request rejected: " + textOr(e, "bad request")
	case e.Status == 401:
		return "Signature rejected: " + textOr(e, "invalid signature") +
			". Your local key may not be the registered one."
	case e.Status == 403:
		pingErr := ping()
		switch {
		case api.StatusOf(pingErr) == 403:
			return "Your session has expired or the token is invalid. Log in again with a new token."
		case pingErr != nil:
			return "Not permitted: " + textOr(e, "forbidden") +
				fmt.Sprintf(" (couldn't check the token: %v)", pingErr)
		default:
			return "Not permitted: " + textOr(e, "forbidden")
		}
	case e.Status == 404:
		return "Not found: " + textOr(e, "not found")
	case e.Status == 409:
		return "Conflict: " + textOr(e, "conflict")
	case e.Status == 413 || e.Status == 415:
		return fmt.Sprintf("Client error %d: %s. This is a bug in the app.", e.Status, textOr(e, "rejected"))
	case e.Status == 502:
		return "Ethereum node error: " + textOr(e, "node unavailable")
	case e.Status >= 500:
		return "Server error. Try again later."
	default:
		return e.Error()
	}
}

func textOr(e *api.Error, fallback string) string {
	if e.Text == "" {
		return fallback
	}
	return e.Text
}

// ShowError shows prefix and the described error in a modal; OK switches to returnTo.
func ShowError(pages *tview.Pages, prefix string, err error, token string, returnTo string) {
	modal := tview.NewModal().
		SetText(prefix + ": " + Describe(err, token)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.SwitchToPage(returnTo)
		})
	pages.AddPage("Error", modal, true, false)
	pages.SwitchToPage("Error")
}

// Package state is the session: who's logged in, their key, the wallet context, and what
// the header and Home show about them. It imports no other app package except api, so
// any screen can use it.
package state

import (
	"crypto/rsa"
	"delsignrepl/api"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Email is the logged-in user's email (from the token); Token is the token itself.
var Email, Token string

// PrivateKey is the user's current signing key, loaded from the key store, or nil.
var PrivateKey *rsa.PrivateKey

// WalletId and Address are the context that balance and send use (0 and "" if unset).
var WalletId int
var Address string

// KeyStatus is a short label for the last key check ("" if not checked), KeyDetail its
// full description, and KeyNext the step it calls for ("" if none).
// keys.RefreshKeyStatus sets them.
var KeyStatus, KeyDetail, KeyNext string
var KeyCheckedAt time.Time

// Balance is the context address's balance in wei at BalanceAt, or nil if not fetched.
// BalanceErr says why the last fetch failed ("" if it didn't).
var Balance *big.Int
var BalanceAt time.Time
var BalanceErr string

// Login starts a session for email with token. A different email than before clears
// everything that belonged to the previous one.
func Login(email, token string) {
	if email != Email {
		PrivateKey = nil
		SetContext(0, "")
		KeyStatus, KeyDetail, KeyNext, KeyCheckedAt = "", "", "", time.Time{}
	}
	Email, Token = email, token
}

// SetContext sets the wallet and address for balance and send, forgetting the old
// balance.
func SetContext(walletID int, address string) {
	WalletId, Address = walletID, address
	Balance, BalanceAt, BalanceErr = nil, time.Time{}, ""
}

// ShortAddress shortens 0x + 40 hex digits to 0xAb12...9F3c; anything else is returned
// as it is.
func ShortAddress(a string) string {
	if len(a) != 42 {
		return a
	}
	return a[:6] + "..." + a[38:]
}

// HeaderText is the one-line summary in the header.
func HeaderText() string {
	if Email == "" {
		return "Not logged in. Log in again (t)."
	}
	parts := []string{Email, "key: " + orDash(KeyStatus)}
	if Address == "" {
		parts = append(parts, "no wallet context")
	} else {
		parts = append(parts, fmt.Sprintf("wallet %d %s", WalletId, ShortAddress(Address)), balanceShort())
	}
	return strings.Join(parts, "  |  ")
}

func balanceShort() string {
	switch {
	case Balance != nil:
		return api.FormatEther(Balance) + " ETH"
	case BalanceErr != "":
		return "balance: ?"
	}
	return "balance: -"
}

// HomeText is the Home screen: the session in full, and what to do next.
func HomeText() string {
	var b strings.Builder
	line := func(label, value string) { fmt.Fprintf(&b, "  %-10s %s\n", label, value) }
	b.WriteString("\n")
	if Email == "" {
		line("Signed in", "no")
		b.WriteString("\n  Next: log in (t).\n")
		return b.String()
	}
	line("Signed in", Email)
	key := orDash(KeyStatus)
	if !KeyCheckedAt.IsZero() {
		key += " (checked " + KeyCheckedAt.Format("15:04:05") + ")"
	}
	line("Key", key)
	if Address == "" {
		line("Wallet", "-")
		line("Address", "-")
	} else {
		line("Wallet", fmt.Sprint(WalletId))
		line("Address", Address)
	}
	switch {
	case Balance != nil:
		line("Balance", fmt.Sprintf("%s ETH (%s wei), as of %s",
			api.FormatEther(Balance), Balance.String(), BalanceAt.Format("15:04:05")))
	case BalanceErr != "":
		line("Balance", "couldn't fetch: "+BalanceErr)
	default:
		line("Balance", "-")
	}
	b.WriteString("\n  Next: " + NextStep() + "\n\n  Refresh (R) checks the key and balance again. Help (?) lists the keys.\n")
	return b.String()
}

// NextStep is the one thing Home suggests doing next.
func NextStep() string {
	switch {
	case Email == "":
		return "log in (t)."
	case KeyNext != "":
		return KeyNext
	case Address == "":
		return "open Wallets (w): create a wallet (n) and an address (a), then press Enter on it."
	case Balance != nil && Balance.Sign() == 0:
		return "fund " + ShortAddress(Address) + " from an Anvil account with cast send (see the readme), then Refresh (R)."
	}
	return "Send ETH (s)."
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

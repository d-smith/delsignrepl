package state

import (
	"math/big"
	"strings"
	"testing"
	"time"
)

const addr = "0xAb12000000000000000000000000000000009F3c"

func reset(t *testing.T) {
	t.Helper()
	clear := func() {
		Email, Token, PrivateKey = "", "", nil
		SetContext(0, "")
		KeyStatus, KeyDetail, KeyNext, KeyCheckedAt = "", "", "", time.Time{}
	}
	clear()
	t.Cleanup(clear)
}

func TestShortAddress(t *testing.T) {
	if got := ShortAddress(addr); got != "0xAb12...9F3c" {
		t.Errorf("got %q", got)
	}
	if got := ShortAddress("0x12"); got != "0x12" {
		t.Errorf("a non-address changed: %q", got)
	}
}

func TestHeaderAndHomeFollowTheSession(t *testing.T) {
	reset(t)
	if h := HeaderText(); !strings.Contains(h, "Not logged in") {
		t.Errorf("logged out header: %q", h)
	}
	if n := NextStep(); !strings.Contains(n, "log in (l)") {
		t.Errorf("logged out next: %q", n)
	}

	Login("a@example.com", "tok")
	KeyStatus, KeyNext = "not registered", "register your key (r)."
	if h := HeaderText(); h != "a@example.com  |  key: not registered  |  no wallet context" {
		t.Errorf("header: %q", h)
	}
	if n := NextStep(); n != KeyNext {
		t.Errorf("a key step should come first, got %q", n)
	}

	KeyStatus, KeyNext = "matches", ""
	if n := NextStep(); !strings.Contains(n, "open Wallets (w)") {
		t.Errorf("no context: %q", n)
	}

	SetContext(2, addr)
	if h := HeaderText(); !strings.HasSuffix(h, "wallet 2 0xAb12...9F3c  |  balance: -") {
		t.Errorf("before a balance fetch: %q", h)
	}
	Balance, BalanceAt = big.NewInt(0), time.Now()
	if n := NextStep(); !strings.Contains(n, "fund 0xAb12...9F3c") {
		t.Errorf("zero balance: %q", n)
	}
	Balance = big.NewInt(49_000_000_000_000_000)
	if h := HeaderText(); !strings.HasSuffix(h, "0.049 ETH") {
		t.Errorf("header balance: %q", h)
	}
	home := HomeText()
	for _, want := range []string{"a@example.com", addr, "0.049 ETH (49000000000000000 wei)", "Next: Send ETH (s)."} {
		if !strings.Contains(home, want) {
			t.Errorf("Home lacks %q:\n%s", want, home)
		}
	}

	Balance, BalanceErr = nil, "Can't reach the server"
	if h := HeaderText(); !strings.HasSuffix(h, "balance: ?") {
		t.Errorf("failed fetch: %q", h)
	}
	if !strings.Contains(HomeText(), "couldn't fetch: Can't reach the server") {
		t.Error("Home should say why the balance is missing")
	}
}

func TestLoginAsSomeoneElseClearsTheSession(t *testing.T) {
	reset(t)
	Login("a@example.com", "t1")
	SetContext(2, addr)
	KeyStatus = "matches"

	Login("a@example.com", "t2") // the same user with a fresh token keeps the context
	if Address != addr || KeyStatus != "matches" || Token != "t2" {
		t.Errorf("same email lost the session: %q %q %q", Address, KeyStatus, Token)
	}
	Login("b@example.com", "t3")
	if Address != "" || WalletId != 0 || KeyStatus != "" || Email != "b@example.com" {
		t.Errorf("another email kept the old session: %q %d %q", Address, WalletId, KeyStatus)
	}
}

func TestSetContextForgetsTheOldBalance(t *testing.T) {
	reset(t)
	SetContext(1, addr)
	Balance, BalanceErr = big.NewInt(5), "x"
	SetContext(2, addr)
	if Balance != nil || BalanceErr != "" {
		t.Error("a new context kept the old balance")
	}
}

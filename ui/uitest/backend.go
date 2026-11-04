package uitest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Backend is a stateful fake of the delsign wallet endpoints, written from
// openapi.yaml's paths and shapes: wallets, addresses, wallet context, balances,
// /authping, and GET /users (always 404: no key registered).
type Backend struct {
	t        *testing.T
	mu       sync.Mutex
	wallets  []int
	addrs    map[int][]string // wallet -> addresses, in creation order
	Balances map[string]string
	next     int
}

// StartBackend serves a fake backend for the test and points DELSIGN_API_URL at it.
// wallets maps wallet ids to their addresses; Address(n) makes deterministic addresses.
func StartBackend(t *testing.T, wallets map[int][]string, order ...int) *Backend {
	t.Helper()
	b := &Backend{t: t, addrs: map[int][]string{}, Balances: map[string]string{}, next: 100}
	for _, id := range order {
		b.wallets = append(b.wallets, id)
		b.addrs[id] = wallets[id]
	}
	srv := httptest.NewServer(b)
	t.Cleanup(srv.Close)
	t.Setenv("DELSIGN_API_URL", srv.URL)
	return b
}

// Address returns a valid, distinct address for n.
func Address(n int) string {
	return fmt.Sprintf("0x%040x", n)
}

func (b *Backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	switch {
	case r.Method == "GET" && path == "/authping":
	case r.Method == "GET" && strings.HasPrefix(path, "/users/"):
		http.Error(w, "No key registered", 404)
	case r.Method == "GET" && path == "/wallets":
		json.NewEncoder(w).Encode(append([]int{}, b.wallets...))
	case r.Method == "POST" && path == "/wallets":
		b.next++
		b.wallets = append(b.wallets, b.next)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]int{"id": b.next})
	case r.Method == "POST" && strings.HasPrefix(path, "/wallets/") && strings.HasSuffix(path, "/addresses"):
		id, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(path, "/wallets/"), "/addresses"))
		if err != nil || !b.has(id) {
			http.Error(w, "Wallet not found", 404)
			return
		}
		b.next++
		a := Address(b.next)
		b.addrs[id] = append(b.addrs[id], a)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]string{"eoa": a})
	case r.Method == "GET" && path == "/walletctx":
		pairs := []map[string]any{}
		for _, id := range b.wallets {
			for _, a := range b.addrs[id] {
				pairs = append(pairs, map[string]any{"walletId": id, "address": a})
			}
		}
		json.NewEncoder(w).Encode(pairs)
	case r.Method == "GET" && strings.HasPrefix(path, "/wallets/balance/"):
		a := strings.TrimPrefix(path, "/wallets/balance/")
		amount, ok := b.Balances[a]
		if !ok {
			amount = "0"
		}
		json.NewEncoder(w).Encode(map[string]string{"address": a, "amount": amount})
	default:
		b.t.Errorf("fake backend: unexpected %s %s", r.Method, r.URL.Path)
		http.Error(w, "not found", 404)
	}
}

func (b *Backend) has(id int) bool {
	for _, w := range b.wallets {
		if w == id {
			return true
		}
	}
	return false
}

// Addresses returns wallet id's addresses.
func (b *Backend) Addresses(id int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string{}, b.addrs[id]...)
}

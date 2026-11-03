package send

// The send flow (plan §3) against a fake POST /wallets/send written from openapi.yaml's
// status codes. Signatures are checked with the standard library.

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"delsignrepl/api"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	keyOnce sync.Once
	testKey *rsa.PrivateKey
)

func key(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	keyOnce.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		testKey = k
	})
	return testKey
}

var (
	src = "0x" + strings.Repeat("1a", 20)
	dst = "0x" + strings.Repeat("2B", 20)
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func verifySig(pub *rsa.PublicKey, msg, sigHex string) error {
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(msg))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}

func TestBuildSend(t *testing.T) {
	k := key(t)
	now := time.Unix(1_800_000_000, 0)
	req, body, err := BuildSend(k, src, dst, big.NewInt(1000), now, NewRequestID)
	if err != nil {
		t.Fatal(err)
	}
	if !uuidPattern.MatchString(req.RequestID) {
		t.Errorf("requestId %q doesn't match the spec pattern", req.RequestID)
	}
	if req.Expiry != now.Unix()+120 {
		t.Errorf("expiry %d, want now+120", req.Expiry)
	}
	if req.Amount != "1000" || req.Source != src || req.Dest != dst {
		t.Errorf("fields changed: %+v", req)
	}
	if err := verifySig(&k.PublicKey, src+dst+"1000"+req.RequestID+"1800000120", req.Sig); err != nil {
		t.Errorf("signature doesn't verify: %v", err)
	}
	var raw map[string]any
	json.Unmarshal(body, &raw)
	if _, ok := raw["amount"].(string); !ok {
		t.Errorf("amount isn't a JSON string in %s", body)
	}

	other, _, _ := BuildSend(k, src, dst, big.NewInt(1000), now, NewRequestID)
	if other.RequestID == req.RequestID {
		t.Error("two builds share a requestId")
	}

	for name, wei := range map[string]*big.Int{"zero": big.NewInt(0), "negative": big.NewInt(-1), "nil": nil,
		"too big": new(big.Int).Lsh(big.NewInt(1), 256)} {
		if _, _, err := BuildSend(k, src, dst, wei, now, NewRequestID); err == nil {
			t.Errorf("%s amount accepted", name)
		}
	}
	if _, _, err := BuildSend(nil, src, dst, big.NewInt(1), now, NewRequestID); err == nil {
		t.Error("nil key accepted")
	}
	if _, _, err := BuildSend(k, src, "0x12", big.NewInt(1), now, NewRequestID); err == nil {
		t.Error("bad dest accepted")
	}
}

// fakeSend answers each POST with the next scripted status (200 if the script is empty).
// -1 closes the connection without answering. It records every body received.
type fakeSend struct {
	t      *testing.T
	pub    *rsa.PublicKey
	script []int
	bodies [][]byte
}

func (f *fakeSend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/api/v1/wallets/send" {
		f.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		return
	}
	body, _ := io.ReadAll(r.Body)
	f.bodies = append(f.bodies, body)
	var req api.SendRequest
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		http.Error(w, "bad body", 400)
		return
	}
	if err := verifySig(f.pub, api.SendMessage(req), req.Sig); err != nil {
		http.Error(w, "Invalid signature", 401)
		return
	}
	status := 200
	if len(f.script) > 0 {
		status, f.script = f.script[0], f.script[1:]
	}
	switch status {
	case 200:
		json.NewEncoder(w).Encode(api.SendResult{Txnid: "0x" + strings.Repeat("ab", 32)})
	case -1:
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	case 409:
		http.Error(w, "Request already processed (requestId reused)", 409)
	case 502:
		http.Error(w, "insufficient funds for gas * price + value", 502)
	default:
		http.Error(w, "nope", status)
	}
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func flowFixture(t *testing.T, script ...int) (*Flow, *fakeSend, *clock) {
	t.Helper()
	k := key(t)
	fake := &fakeSend{t: t, pub: &k.PublicKey, script: script}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	c := &clock{time.Unix(1_800_000_000, 0)}
	f := &Flow{Client: &api.Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()},
		Key: k, Now: c.now, NewID: NewRequestID}
	return f, fake, c
}

func TestFlowSent(t *testing.T) {
	f, fake, _ := flowFixture(t)
	r := f.Confirm(src, dst, big.NewInt(5))
	if r.Outcome != Sent || r.Txnid == "" || r.Err != nil {
		t.Fatalf("got %+v", r)
	}
	if f.CanRetry() {
		t.Error("a sent request offers a retry")
	}
	if _, err := f.Retry(); !errors.Is(err, ErrCantRetry) || len(fake.bodies) != 1 {
		t.Errorf("Retry after 200: %v, %d posts", err, len(fake.bodies))
	}
}

func TestFlowRetrySendsIdenticalBytes(t *testing.T) {
	f, fake, c := flowFixture(t, -1, 200)
	r := f.Confirm(src, dst, big.NewInt(5))
	if r.Outcome != NoResponse {
		t.Fatalf("got %+v, want NoResponse", r)
	}
	c.t = c.t.Add(60 * time.Second)
	if !f.CanRetry() {
		t.Fatal("no retry offered within expiry")
	}
	r, err := f.Retry()
	if err != nil || r.Outcome != Sent {
		t.Fatalf("retry: %+v, %v", r, err)
	}
	if len(fake.bodies) != 2 || !bytes.Equal(fake.bodies[0], fake.bodies[1]) {
		t.Errorf("retry didn't resend identical bytes:\n%s\n%s", fake.bodies[0], fake.bodies[1])
	}
}

func TestFlowNoRetryAfterExpiry(t *testing.T) {
	f, fake, c := flowFixture(t, -1)
	f.Confirm(src, dst, big.NewInt(5))
	c.t = c.t.Add(ExpiryWindow)
	if f.CanRetry() {
		t.Error("retry offered at expiry")
	}
	if _, err := f.Retry(); !errors.Is(err, ErrExpired) || len(fake.bodies) != 1 {
		t.Errorf("Retry at expiry: %v, %d posts", err, len(fake.bodies))
	}
}

func TestFlow409NeverRetriesWithNewID(t *testing.T) {
	f, fake, _ := flowFixture(t, -1, 409)
	f.Confirm(src, dst, big.NewInt(5))
	r, err := f.Retry()
	if err != nil || r.Outcome != AlreadyProcessed {
		t.Fatalf("got %+v, %v; want AlreadyProcessed", r, err)
	}
	if f.CanRetry() {
		t.Error("a 409 offers a retry")
	}
	if _, err := f.Retry(); !errors.Is(err, ErrCantRetry) {
		t.Errorf("Retry after 409: %v", err)
	}
	if len(fake.bodies) != 2 || !bytes.Equal(fake.bodies[0], fake.bodies[1]) {
		t.Errorf("want the same body twice and nothing more, got %d posts", len(fake.bodies))
	}
}

func TestFlow502NewIDOnlyAfterConfirm(t *testing.T) {
	f, fake, _ := flowFixture(t, 502, 200)
	r := f.Confirm(src, dst, big.NewInt(5))
	if r.Outcome != NodeError || !strings.Contains(errText(r.Err), "insufficient funds") {
		t.Fatalf("got %+v", r)
	}
	first := f.Request().RequestID
	if _, err := f.Retry(); !errors.Is(err, ErrCantRetry) || len(fake.bodies) != 1 {
		t.Fatalf("Retry after 502: %v, %d posts", err, len(fake.bodies))
	}
	r = f.Confirm(src, dst, big.NewInt(5))
	if r.Outcome != Sent || f.Request().RequestID == first {
		t.Errorf("second confirm: %+v, id %s (first %s)", r, f.Request().RequestID, first)
	}
}

func TestFlowClassifiesOtherStatuses(t *testing.T) {
	for status, want := range map[int]Outcome{400: Rejected, 403: Failed, 500: Failed} {
		f, _, _ := flowFixture(t, status)
		if r := f.Confirm(src, dst, big.NewInt(5)); r.Outcome != want || f.CanRetry() {
			t.Errorf("%d: got %v (retry %v), want %v", status, r.Outcome, f.CanRetry(), want)
		}
	}
	f, _, _ := flowFixture(t)
	other, _ := rsa.GenerateKey(rand.Reader, 1024)
	f.Key = other
	if r := f.Confirm(src, dst, big.NewInt(5)); r.Outcome != SignatureRejected {
		t.Errorf("wrong key: got %v, want SignatureRejected", r.Outcome)
	}
}

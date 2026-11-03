package send

import (
	"crypto/rsa"
	"delsignrepl/api"
	"errors"
	"math/big"
	"time"
)

// Outcome classifies what happened to one POST /wallets/send (plan §3, "The send flow").
type Outcome int

const (
	Sent              Outcome = iota // 200: broadcast, Txnid set
	NoResponse                       // transport error or timeout: may or may not have happened
	AlreadyProcessed                 // 409: this requestId was used; never retried with a new id
	NodeError                        // 502: most likely not sent; the request can't be reused
	SignatureRejected                // 401
	Rejected                         // 400
	Failed                           // anything else (403, 5xx, a local error)
)

// Result is one attempt's outcome.
type Result struct {
	Outcome Outcome
	Txnid   string
	Err     error
}

// ErrCantRetry is returned by Retry when there's no request that may be resent.
var ErrCantRetry = errors.New("this request can't be resent")

// ErrExpired is returned by Retry once the request's expiry has passed.
var ErrExpired = errors.New("this request has expired; check the balance before sending again")

// Flow owns one send form's requests. Confirm builds a new request (new requestId,
// expiry and signature) and posts it. Retry resends the identical bytes, and only after
// a no-response outcome and before expiry.
type Flow struct {
	Client *api.Client
	Key    *rsa.PrivateKey
	Now    func() time.Time
	NewID  func() (string, error)

	req      api.SendRequest
	body     []byte
	canRetry bool
}

// NewFlow returns a Flow using the real clock and random UUIDs.
func NewFlow(c *api.Client, key *rsa.PrivateKey) *Flow {
	return &Flow{Client: c, Key: key, Now: time.Now, NewID: NewRequestID}
}

// Request returns the last request built, for display.
func (f *Flow) Request() api.SendRequest { return f.req }

// Confirm is called once per user confirmation. It builds and signs a fresh request,
// keeps its bytes, and posts them.
func (f *Flow) Confirm(source, dest string, wei *big.Int) Result {
	req, body, err := BuildSend(f.Key, source, dest, wei, f.Now(), f.NewID)
	if err != nil {
		f.canRetry = false
		return Result{Outcome: Failed, Err: err}
	}
	f.req, f.body = req, body
	return f.post()
}

// CanRetry reports whether Retry would resend: the last outcome was no response and the
// request hasn't expired.
func (f *Flow) CanRetry() bool {
	return f.canRetry && f.Now().Unix() < f.req.Expiry
}

// Retry resends the last request's identical bytes.
func (f *Flow) Retry() (Result, error) {
	if !f.canRetry {
		return Result{}, ErrCantRetry
	}
	if f.Now().Unix() >= f.req.Expiry {
		return Result{}, ErrExpired
	}
	return f.post(), nil
}

func (f *Flow) post() Result {
	res, err := f.Client.Send(f.body)
	r := Result{Outcome: classify(err), Txnid: res.Txnid, Err: err}
	f.canRetry = r.Outcome == NoResponse
	return r
}

func classify(err error) Outcome {
	if err == nil {
		return Sent
	}
	var netErr *api.NetworkError
	if errors.As(err, &netErr) {
		return NoResponse
	}
	switch api.StatusOf(err) {
	case 409:
		return AlreadyProcessed
	case 502:
		return NodeError
	case 401:
		return SignatureRejected
	case 400:
		return Rejected
	}
	return Failed
}

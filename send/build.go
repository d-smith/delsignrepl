package send

import (
	"crypto/rsa"
	"delsignrepl/api"
	"delsignrepl/keys"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// ExpiryWindow is how far ahead a send's expiry is set (plan D9; the server allows 5 min).
const ExpiryWindow = 120 * time.Second

// NewRequestID returns a random (v4) UUID.
func NewRequestID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// BuildSend builds and signs a send request for wei from source to dest, expiring 120 s
// after now, with a request id from newID. It returns the request and its JSON body; the
// body bytes are what gets sent (and resent unchanged on a retry). Nothing in the request
// may change after signing.
func BuildSend(priv *rsa.PrivateKey, source, dest string, wei *big.Int, now time.Time, newID func() (string, error)) (api.SendRequest, []byte, error) {
	if priv == nil {
		return api.SendRequest{}, nil, errors.New("no signing key: generate and register one first")
	}
	if wei == nil || wei.Sign() <= 0 || wei.Cmp(api.MaxWei) > 0 {
		return api.SendRequest{}, nil, errors.New("the amount must be between 1 and 2^256-1 wei")
	}
	if !api.ValidAddress(source) || !api.ValidAddress(dest) {
		return api.SendRequest{}, nil, errors.New("addresses must be 0x followed by 40 hex digits")
	}
	id, err := newID()
	if err != nil {
		return api.SendRequest{}, nil, err
	}
	req := api.SendRequest{
		Source:    source,
		Dest:      dest,
		Amount:    wei.String(),
		RequestID: id,
		Expiry:    now.Add(ExpiryWindow).Unix(),
	}
	if req.Sig, err = keys.Sign(api.SendMessage(req), priv); err != nil {
		return api.SendRequest{}, nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return api.SendRequest{}, nil, err
	}
	return req, body, nil
}

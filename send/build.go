package send

import (
	"crypto/rsa"
	"delsignrepl/api"
	"errors"
	"math/big"
	"time"
)

// BuildSend builds and signs a send request for wei from source to dest, expiring 120 s
// after now, with a request id from newID. It returns the request and its JSON body; the
// body bytes are what gets sent (and resent unchanged on a retry).
//
// Phase 1 stub: implemented in Phase 4.
func BuildSend(priv *rsa.PrivateKey, source, dest string, wei *big.Int, now time.Time, newID func() (string, error)) (api.SendRequest, []byte, error) {
	return api.SendRequest{}, nil, errors.New("BuildSend: not implemented")
}

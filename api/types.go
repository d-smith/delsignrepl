package api

type KeyReg struct {
	Email                    string `json:"email"`
	PubKey                   string `json:"pubkey"`
	SignatureForRegistration string `json:"sig4reg"`
}

type WalletInfo struct {
	ID int `json:"id"`
}

// SendRequest is the body of POST /api/v1/wallets/send (openapi.yaml, SendRequest).
// It has exactly these six fields: the server rejects unknown fields with a 400.
// Amount is wei as a canonical decimal string; Expiry is Unix time in seconds.
type SendRequest struct {
	Source    string `json:"source"`
	Dest      string `json:"dest"`
	Amount    string `json:"amount"`
	RequestID string `json:"requestId"`
	Expiry    int64  `json:"expiry"`
	Sig       string `json:"sig"`
}

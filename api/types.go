package api

// KeyReg is the body of POST /api/v1/keyreg. Sig4Rotate is only for replacing a
// registered key; omitempty keeps it out of a first registration, where an empty value
// would be rejected.
type KeyReg struct {
	Email                    string `json:"email"`
	PubKey                   string `json:"pubkey"`
	SignatureForRegistration string `json:"sig4reg"`
	Sig4Rotate               string `json:"sig4rotate,omitempty"`
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

// UserInfo is the response of GET /api/v1/users/{email}.
type UserInfo struct {
	Email  string `json:"email"`
	PubKey string `json:"pubkey"`
}

// EOA is the response of POST /api/v1/wallets/{id}/addresses.
type EOA struct {
	Address string `json:"eoa"`
}

// WalletAddressPair is one entry of GET /api/v1/walletctx.
type WalletAddressPair struct {
	WalletId int    `json:"walletId"`
	Address  string `json:"address"`
}

// Balance is the response of GET /api/v1/wallets/balance/{address}. Amount is wei as a
// decimal string; parse it into a *big.Int, never a float or int64.
type Balance struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// SendResult is the response of POST /api/v1/wallets/send. Txnid is the transaction hash
// of a broadcast (not yet mined) transaction.
type SendResult struct {
	Txnid string `json:"txnid"`
}

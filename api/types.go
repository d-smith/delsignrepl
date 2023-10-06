package api

type KeyReg struct {
	Email                    string `json:"email"`
	PubKey                   string `json:"pubkey"`
	SignatureForRegistration string `json:"sig4reg"`
}

type WalletInfo struct {
	ID int `json:"id"`
}

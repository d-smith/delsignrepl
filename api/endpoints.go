package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// One method per endpoint in openapi.yaml. Any 2xx is success; the specific codes are
// in the spec.

// AuthPing checks the token: nil if valid, an *Error with status 403 if not.
func (c *Client) AuthPing() error {
	return c.Do(http.MethodGet, "/api/v1/authping", nil, nil)
}

// GetUser returns the registered key for email, which must be the token's own email.
// A 404 *Error means no key is registered yet.
func (c *Client) GetUser(email string) (UserInfo, error) {
	var u UserInfo
	err := c.Do(http.MethodGet, "/api/v1/users/"+url.PathEscape(email), nil, &u)
	return u, err
}

// RegisterKey registers or replaces the user's RSA public key.
func (c *Client) RegisterKey(reg KeyReg) error {
	return c.Do(http.MethodPost, "/api/v1/keyreg", reg, nil)
}

// CreateWallet creates an empty wallet and returns its id.
func (c *Client) CreateWallet() (int, error) {
	var w WalletInfo
	err := c.Do(http.MethodPost, "/api/v1/wallets", nil, &w)
	return w.ID, err
}

// ListWallets returns the user's wallet ids (empty if none).
func (c *Client) ListWallets() ([]int, error) {
	var ids []int
	err := c.Do(http.MethodGet, "/api/v1/wallets", nil, &ids)
	return ids, err
}

// CreateAddress creates a new address in one of the user's wallets.
func (c *Client) CreateAddress(walletID int) (string, error) {
	var e EOA
	err := c.Do(http.MethodPost, fmt.Sprintf("/api/v1/wallets/%d/addresses", walletID), nil, &e)
	return e.Address, err
}

// WalletContext returns every (wallet, address) pair the user has (empty if none).
func (c *Client) WalletContext() ([]WalletAddressPair, error) {
	var pairs []WalletAddressPair
	err := c.Do(http.MethodGet, "/api/v1/walletctx", nil, &pairs)
	return pairs, err
}

// Balance returns an address's balance. Amount is wei as a decimal string.
func (c *Client) Balance(address string) (Balance, error) {
	var b Balance
	err := c.Do(http.MethodGet, "/api/v1/wallets/balance/"+url.PathEscape(address), nil, &b)
	return b, err
}

// Send posts a send request body exactly as given, so the bytes that were signed are the
// bytes sent, and a retry can resend them unchanged.
func (c *Client) Send(body []byte) (SendResult, error) {
	var r SendResult
	data, err := c.DoRaw(http.MethodPost, "/api/v1/wallets/send", body)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("decoding send response: %w", err)
	}
	return r, nil
}

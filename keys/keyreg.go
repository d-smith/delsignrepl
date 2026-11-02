package keys

import (
	"crypto/rsa"
	"delsignrepl/api"
	"errors"
)

// BuildKeyReg builds a POST /api/v1/keyreg body registering next for email.
// current is the registered key when replacing it (adds sig4rotate), or nil for a
// first registration.
//
// Phase 1 stub: implemented in Phase 3.
func BuildKeyReg(email string, next, current *rsa.PrivateKey) (api.KeyReg, error) {
	return api.KeyReg{}, errors.New("BuildKeyReg: not implemented")
}

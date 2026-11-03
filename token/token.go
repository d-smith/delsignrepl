package token

import (
	"delsignrepl/api"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

// DevSecretEnv names the env var holding the backend's JWT secret, for the dev-only
// cheat code. It must equal the backend's DELSIGN_JWT_SECRET. Never commit its value.
const DevSecretEnv = "DELSIGN_DEV_JWT_SECRET"

// DevSecret returns the cheat code's signing secret, or "" if the cheat code is off.
func DevSecret() string {
	return os.Getenv(DevSecretEnv)
}

// Claims are the parts of a token this app reads.
type Claims struct {
	Email   string
	Expires time.Time
}

// ParseClaims reads email and exp from a token without verifying its signature: the
// backend verifies it. It rejects tokens the backend is certain to reject: no email, no
// exp, or already expired at now.
func ParseClaims(tok string, now time.Time) (Claims, error) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return Claims{}, errors.New("enter a token")
	}
	claims := jwt.MapClaims{}
	if _, _, err := new(jwt.Parser).ParseUnverified(tok, claims); err != nil {
		return Claims{}, fmt.Errorf("not a JWT: %w", err)
	}
	email, _ := claims["email"].(string)
	if email == "" {
		return Claims{}, errors.New("the token has no email claim")
	}
	expRaw, ok := claims["exp"]
	if !ok {
		return Claims{}, errors.New("the token has no exp claim, which the backend requires")
	}
	expNum, ok := expRaw.(float64)
	if !ok || math.IsNaN(expNum) || math.IsInf(expNum, 0) {
		return Claims{}, errors.New("the token's exp claim isn't a number")
	}
	exp := time.Unix(int64(expNum), 0)
	if !now.Before(exp) {
		return Claims{}, fmt.Errorf("the token expired at %s", exp.Format(time.RFC3339))
	}
	return Claims{Email: email, Expires: exp}, nil
}

// Login checks a token locally, then with GET /authping using c (which must carry the
// token). It returns the token's claims if both accept it.
func Login(tok string, c *api.Client, now time.Time) (Claims, error) {
	claims, err := ParseClaims(tok, now)
	if err != nil {
		return Claims{}, err
	}
	err = c.AuthPing()
	if api.StatusOf(err) == 403 {
		hint := ""
		if DevSecret() != "" {
			hint = fmt.Sprintf(" If you used the cheat code, check that %s matches the backend's DELSIGN_JWT_SECRET.", DevSecretEnv)
		}
		return Claims{}, errors.New("the backend rejected this token (403)." + hint)
	}
	if err != nil {
		return Claims{}, err
	}
	return claims, nil
}

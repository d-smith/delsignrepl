package api

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// Wei amounts are decimal strings on the wire (openapi.yaml: pattern ^(0|[1-9][0-9]*)$,
// maxLength 78, at most 2^256-1). In code they are *big.Int, never int64 or float64.

var (
	weiPattern     = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
	addressPattern = regexp.MustCompile(`^0[xX][0-9a-fA-F]{40}$`)

	// MaxWei is 2^256-1, the largest amount the server accepts.
	MaxWei = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	weiPerEther = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

// ParseWei parses a wei amount in the wire format: decimal digits, no sign, no leading
// zeros, at most 2^256-1.
func ParseWei(s string) (*big.Int, error) {
	if !weiPattern.MatchString(s) {
		return nil, fmt.Errorf("%q isn't a wei amount: use digits only, with no sign or leading zeros", s)
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok || v.Cmp(MaxWei) > 0 {
		return nil, fmt.Errorf("%q is larger than 2^256-1 wei", s)
	}
	return v, nil
}

// FormatEther shows a wei amount in ETH exactly: the integer part, then up to 18
// fraction digits with trailing zeros trimmed.
func FormatEther(wei *big.Int) string {
	neg := wei.Sign() < 0
	q, r := new(big.Int).QuoRem(new(big.Int).Abs(wei), weiPerEther, new(big.Int))
	s := q.String()
	if r.Sign() != 0 {
		s += "." + strings.TrimRight(fmt.Sprintf("%018s", r.String()), "0")
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ValidAddress reports whether s is 0x + 40 hex digits (any letter case).
func ValidAddress(s string) bool {
	return addressPattern.MatchString(s)
}

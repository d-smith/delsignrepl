package api

import "strconv"

// SendMessage returns the string the user's RSA key signs for a send:
// source + dest + amount + requestId + expiry (decimal), with no separators,
// using exactly the strings in the request.
func SendMessage(r SendRequest) string {
	return r.Source + r.Dest + r.Amount + r.RequestID + strconv.FormatInt(r.Expiry, 10)
}

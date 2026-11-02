package api

// SendMessage returns the string the user's RSA key signs for a send:
// source + dest + amount + requestId + expiry (decimal), with no separators,
// using exactly the strings in the request.
//
// Phase 1 stub: returns "" so the vector test compiles and fails. Implemented in Phase 4.
func SendMessage(r SendRequest) string {
	return ""
}

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultBaseURL is the backend's local development address (client-guide.md, "Connecting").
const DefaultBaseURL = "http://localhost:3010"

// DefaultTimeout bounds every call. Calls run on the UI goroutine, so this is also the
// longest the UI can freeze (plan D6).
const DefaultTimeout = 20 * time.Second

// maxResponse caps how much of a response body is read.
const maxResponse = 1 << 20

var sharedHTTP = &http.Client{Timeout: DefaultTimeout}

// Error is a response with a non-2xx status. Text is the server's text/plain message,
// trimmed: show or log it, but switch on Status, never on Text.
type Error struct {
	Status int
	Text   string
}

func (e *Error) Error() string {
	if e.Text == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Text)
}

// NetworkError means no response arrived: the server couldn't be reached, the
// connection failed, or the call timed out. The request may or may not have been
// processed.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string { return "no response from server: " + e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }

// StatusOf returns the HTTP status of an *Error in err's chain, or 0.
func StatusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

// BaseURLFromEnv returns DELSIGN_API_URL without a trailing slash, or DefaultBaseURL.
func BaseURLFromEnv() string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv("DELSIGN_API_URL")), "/"); v != "" {
		return v
	}
	return DefaultBaseURL
}

// Client calls the delsign backend. It is the only code in this app that does HTTP.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New returns a client for the configured base URL that sends token on every call.
func New(token string) *Client {
	return &Client{BaseURL: BaseURLFromEnv(), Token: token, HTTP: sharedHTTP}
}

// DoRaw sends body exactly as given (nil for no body) and returns the response body of a
// 2xx response. A non-2xx response gives an *Error; no response gives a *NetworkError.
// Content-Type is set only when there is a body.
func (c *Client) DoRaw(method, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("building %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = sharedHTTP
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &Error{Status: resp.StatusCode, Text: strings.TrimSpace(string(data))}
	}
	return data, nil
}

// Do marshals in (if not nil) as the JSON body, and decodes a JSON response into out
// (if not nil).
func (c *Client) Do(method, path string, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return fmt.Errorf("encoding %s %s: %w", method, path, err)
		}
	}
	data, err := c.DoRaw(method, path, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding %s %s response: %w", method, path, err)
	}
	return nil
}

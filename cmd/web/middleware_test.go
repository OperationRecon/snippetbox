package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"snippetbox.opre.net/internal/assert"
)

func TestSecureHeader(t *testing.T) {
	// initialize a response recorder
	rr := httptest.NewRecorder()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// send request to secure headers, use ping as the mock http next
	secureHeaders(http.HandlerFunc(ping)).ServeHTTP(rr, r)

	rs := rr.Result()

	/// Check that the middleware has correctly set the
	// headers on the response.
	expected := "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com"
	assert.Equal(t, rs.Header.Get("Content-Security-Policy"), expected)
	expected = "origin-when-cross-origin"
	assert.Equal(t, rs.Header.Get("Referrer-Policy"), expected)
	expected = "nosniff"
	assert.Equal(t, rs.Header.Get("X-Content-Type-Options"), expected)
	expected = "deny"
	assert.Equal(t, rs.Header.Get("X-Frame-Options"), expected)
	expected = "0"
	assert.Equal(t, rs.Header.Get("X-XSS-Protection"), expected)

	// assert that the next handler was called
	assert.Equal(t, rs.StatusCode, http.StatusOK)

	defer rs.Body.Close()
	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	bytes.TrimSpace(body)
	assert.Equal(t, string(body), "OK")
}

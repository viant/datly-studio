package studioapi

import "testing"

func TestSecureSessionCookies(t *testing.T) {
	for _, tc := range []struct {
		origin string
		secure bool
	}{
		{"http://127.0.0.1:18183", false}, {"http://localhost:18183", false}, {"http://[::1]:18183", false},
		{"https://localhost:18183", true}, {"https://studio.example", true}, {"http://studio.example", true}, {"", true},
	} {
		if got := secureSessionCookies(tc.origin); got != tc.secure {
			t.Errorf("%s secure=%v want %v", tc.origin, got, tc.secure)
		}
	}
}

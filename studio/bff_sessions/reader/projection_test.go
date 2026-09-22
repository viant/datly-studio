package reader

import (
	"os"
	"strings"
	"testing"
)

func TestSessionReaderNeverProjectsEncryptedCredential(t *testing.T) {
	payload, err := os.ReadFile("sql/read.sql")
	if err != nil {
		t.Fatal(err)
	}
	query := strings.ToLower(string(payload))
	for _, secret := range []string{"payload_ciphertext", " token", "claims"} {
		if strings.Contains(query, secret) {
			t.Fatalf("session metadata reader projects secret column %q", secret)
		}
	}
}

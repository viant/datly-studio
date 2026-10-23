// Package resourceartifact defines the private identity of compiled dynamic
// readers stored in Studio's existing versioned resource-file table.
package resourceartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Namespace is reserved for server-owned compilation artifacts. It must not
// appear in authored resource snapshots, resource fingerprints or exports.
const Namespace = "__datly_compiled"

const MediaType = "application/vnd.viant.datly.runtime-contract+json"

type writerContextKey struct{}
type repairContextKey struct{}

// WithTrustedWrite is used only by Studio's in-process compiler after it has
// selected an immutable published version. Browser-authored resource calls do
// not receive this context value.
func WithTrustedWrite(ctx context.Context) context.Context {
	return context.WithValue(ctx, writerContextKey{}, true)
}

func TrustedWrite(ctx context.Context) bool {
	value, _ := ctx.Value(writerContextKey{}).(bool)
	return value
}

// WithTrustedRepair carries an expected SHA-256 of the exact previous stored
// bytes. The generated writer checks it against Previous inside its mutation
// transaction before replacing a structurally invalid artifact.
func WithTrustedRepair(ctx context.Context, previousSHA string) context.Context {
	return context.WithValue(WithTrustedWrite(ctx), repairContextKey{}, previousSHA)
}

func RepairDigest(ctx context.Context) string {
	value, _ := ctx.Value(repairContextKey{}).(string)
	return value
}

// Identity binds the original immutable source URI and every compilation input
// revision. The logical URI is never used as a filesystem path or DB key.
type Identity struct {
	SourceURI string
	InputSHA  string
	Key       string
	URI       string
	Path      string
}

func New(sourceURI, inputSHA string) (Identity, error) {
	if strings.TrimSpace(sourceURI) != sourceURI || sourceURI == "" || len(inputSHA) != 64 {
		return Identity{}, fmt.Errorf("compiled artifact source URI and SHA-256 are required")
	}
	if _, err := hex.DecodeString(inputSHA); err != nil || inputSHA != strings.ToLower(inputSHA) {
		return Identity{}, fmt.Errorf("compiled artifact input SHA-256 is invalid")
	}
	sourceHash := sha256.Sum256([]byte(sourceURI))
	keyHash := sha256.Sum256([]byte("datly-studio/compiled-v1\x00" + sourceURI + "\x00" + inputSHA))
	key := hex.EncodeToString(keyHash[:])
	return Identity{SourceURI: sourceURI, InputSHA: inputSHA, Key: key,
		URI:  "compiled://" + hex.EncodeToString(sourceHash[:]) + "/" + inputSHA,
		Path: "compiled/" + key + ".json"}, nil
}

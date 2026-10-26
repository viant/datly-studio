package studiohost

import (
	"context"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/scy"
)

// browserSessionKey keeps encrypted server-side OAuth sessions readable across
// restarts. The confidential client secret already comes from the host's SCY
// configuration; HKDF separates this at-rest key from OAuth authentication and
// from other applications using the same client. No token or key reaches the UI.
func browserSessionKey(ctx context.Context, override *scy.Resource, login *studioapi.Login, origin string) (string, error) {
	if override != nil {
		secret, err := scy.New().Load(ctx, override)
		if err != nil {
			return "", errors.New("Studio session secret could not be loaded")
		}
		return strings.TrimSpace(secret.String()), nil
	}
	if login == nil || login.ClientSecret == "" {
		return randomKey()
	}
	// JSON preserves unambiguous boundaries between independently scoped fields.
	info, err := json.Marshal([]string{"datly-studio/bff-session/v1", origin, login.ClientID, login.TokenURL})
	if err != nil {
		return "", err
	}
	key, err := hkdf.Key(sha256.New, []byte(login.ClientSecret), nil, string(info), 32)
	if err != nil {
		return "", errors.New("Studio OAuth session key derivation failed")
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

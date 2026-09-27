package bffauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/viant/datly-studio/sdk"
	leasewriter "github.com/viant/datly-studio/studio/bff_sessions/store_lease"
	"github.com/viant/scy/auth/jwt"
)

// SQLStore is a multi-instance session store backed by server-only Datly v1
// components against the Studio database. Cookie identifiers are hashed and
// the bearer/claims payload is encrypted before reaching those components.
type SQLStore struct {
	db         *sql.DB
	aead       cipher.AEAD
	mu         sync.Mutex
	components *sessionComponents
}

type storedPayload struct {
	Principal        sdk.Principal `json:"principal"`
	Token            string        `json:"token"`
	RefreshToken     string        `json:"refreshToken,omitempty"`
	IDTokenExpiresAt time.Time     `json:"idTokenExpiresAt,omitempty"`
	Claims           *jwt.Claims   `json:"claims"`
	ExpiresAt        time.Time     `json:"expiresAt"`
}

func NewSQLStore(db *sql.DB, key []byte) (*SQLStore, error) {
	if db == nil {
		return nil, errors.New("BFF session database is required")
	}
	if len(key) != 32 {
		return nil, errors.New("BFF session encryption key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SQLStore{db: db, aead: aead}, nil
}

func (s *SQLStore) Put(ctx context.Context, id string, value session) error {
	hash := sessionHash(id)
	existing, err := s.readDatlySessionRecord(ctx, hash)
	if err != nil {
		return err
	}
	ciphertext, err := s.encryptPayload(hash, value)
	if err != nil {
		return err
	}
	created := time.Now().UTC()
	if existing != nil && existing.CreatedAt != nil {
		created = *existing.CreatedAt
	}
	return s.writeDatlySession(ctx, hash, value.principal.Subject, ciphertext, value.expiresAt.Unix(), created, false)
}

func (s *SQLStore) encryptPayload(hash string, value session) ([]byte, error) {
	payload, err := json.Marshal(storedPayload{Principal: value.principal, Token: value.token, RefreshToken: value.refreshToken, IDTokenExpiresAt: value.idTokenExpiresAt, Claims: value.claims, ExpiresAt: value.expiresAt})
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, payload, []byte(hash)), nil
}

// AcquireRefreshLease is an atomic, cross-instance claim on one encrypted
// session. Only the winner may contact the provider's rotating token endpoint.
func (s *SQLStore) AcquireRefreshLease(ctx context.Context, id, owner string, now time.Time, ttl time.Duration) (bool, error) {
	if ttl <= 0 || ttl > 2*time.Minute {
		return false, errors.New("invalid refresh lease lifetime")
	}
	return s.leaseOperation(ctx, &leasewriter.Input{Operation: "acquire", SessionIDHash: sessionHash(id),
		LeaseOwner: owner, NowUnix: now.Unix(), LeaseUntilUnix: now.Add(ttl).Unix()})
}

// CompleteRefreshLease fences stale refresh workers and atomically stores the
// rotated encrypted token set while releasing the lease.
func (s *SQLStore) CompleteRefreshLease(ctx context.Context, id, owner string, now time.Time, value session) error {
	hash := sessionHash(id)
	ciphertext, err := s.encryptPayload(hash, value)
	if err != nil {
		return err
	}
	_, err = s.leaseOperation(ctx, &leasewriter.Input{Operation: "complete", SessionIDHash: hash,
		LeaseOwner: owner, NowUnix: now.Unix(), PayloadCiphertext: ciphertext})
	return err
}

func (s *SQLStore) ReleaseRefreshLease(ctx context.Context, id, owner string, now time.Time) error {
	_, err := s.leaseOperation(ctx, &leasewriter.Input{Operation: "release", SessionIDHash: sessionHash(id),
		LeaseOwner: owner, NowUnix: now.Unix()})
	return err
}

func (s *SQLStore) Get(ctx context.Context, id string) (session, bool, error) {
	hash := sessionHash(id)
	ciphertext, expiresUnix, found, err := s.readDatlySession(ctx, hash)
	if err != nil || !found {
		return session{}, found, err
	}
	if len(ciphertext) < s.aead.NonceSize() {
		return session{}, false, errors.New("invalid encrypted BFF session")
	}
	nonce, encrypted := ciphertext[:s.aead.NonceSize()], ciphertext[s.aead.NonceSize():]
	plain, err := s.aead.Open(nil, nonce, encrypted, []byte(hash))
	if err != nil {
		return session{}, false, fmt.Errorf("decrypt BFF session: %w", err)
	}
	var payload storedPayload
	if err = json.Unmarshal(plain, &payload); err != nil {
		return session{}, false, err
	}
	expires := time.Unix(expiresUnix, 0)
	return session{principal: payload.Principal, token: payload.Token, refreshToken: payload.RefreshToken, idTokenExpiresAt: payload.IDTokenExpiresAt, claims: payload.Claims, expiresAt: expires}, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) error {
	return s.deleteHashedSession(ctx, sessionHash(id))
}

func (s *SQLStore) deleteHashedSession(ctx context.Context, hash string) error {
	_, _, found, err := s.readDatlySession(ctx, hash)
	if err != nil || !found {
		return err
	}
	if err := s.writeDatlySession(ctx, hash, "", nil, 0, time.Time{}, true); err != nil {
		// Another instance may have deleted it after the first read. The
		// store contract is idempotent: verify the desired state instead of
		// interpreting a writer error string.
		_, _, stillPresent, readErr := s.readDatlySession(ctx, hash)
		if readErr == nil && !stillPresent {
			return nil
		}
		return err
	}
	return nil
}

func (s *SQLStore) DeleteExpired(ctx context.Context, now time.Time) error {
	return s.deleteExpiredDatly(ctx, now.Unix())
}

func sessionHash(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

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
	"time"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/scy/auth/jwt"
)

// SQLStore is a multi-instance session store backed by the Studio database.
// Cookie identifiers are hashed and the bearer/claims payload is encrypted.
type SQLStore struct {
	db   *sql.DB
	aead cipher.AEAD
}

type storedPayload struct {
	Principal sdk.Principal `json:"principal"`
	Token     string        `json:"token"`
	Claims    *jwt.Claims   `json:"claims"`
	ExpiresAt time.Time     `json:"expiresAt"`
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
	payload, err := json.Marshal(storedPayload{Principal: value.principal, Token: value.token, Claims: value.claims, ExpiresAt: value.expiresAt})
	if err != nil {
		return err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ciphertext := s.aead.Seal(nonce, nonce, payload, []byte(hash))
	_, err = s.db.ExecContext(ctx, `INSERT INTO bff_sessions(session_id_hash,subject_id,payload_ciphertext,expires_at_unix,created_at)
VALUES(?,?,?,?,?) ON CONFLICT(session_id_hash) DO UPDATE SET subject_id=excluded.subject_id,payload_ciphertext=excluded.payload_ciphertext,expires_at_unix=excluded.expires_at_unix`, hash, value.principal.Subject, ciphertext, value.expiresAt.Unix(), time.Now().UTC())
	return err
}

func (s *SQLStore) Get(ctx context.Context, id string) (session, bool, error) {
	hash := sessionHash(id)
	var ciphertext []byte
	var expiresUnix int64
	if err := s.db.QueryRowContext(ctx, `SELECT payload_ciphertext,expires_at_unix FROM bff_sessions WHERE session_id_hash=?`, hash).Scan(&ciphertext, &expiresUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session{}, false, nil
		}
		return session{}, false, err
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
	return session{principal: payload.Principal, token: payload.Token, claims: payload.Claims, expiresAt: expires}, true, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bff_sessions WHERE session_id_hash=?`, sessionHash(id))
	return err
}

func (s *SQLStore) DeleteExpired(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bff_sessions WHERE expires_at_unix<=?`, now.Unix())
	return err
}

func sessionHash(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

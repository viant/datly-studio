package oauth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk/access"
)

func TestGroupedAndLegacySignedEntityClaims(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := New(Config{Issuer: "issuer", Audience: "audience", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwt.Token) (any, error) { return public, nil }})
	if err != nil {
		t.Fatal(err)
	}
	base := jwt.RegisteredClaims{Issuer: "issuer", Subject: "alice", Audience: []string{"audience"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}
	canonical := Claims{RegisteredClaims: base, Tenant: "one", EntityGroups: access.EntityGroups{"project": {"9007199254740993", "102"}, "organization": {"north"}}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, canonical).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte(`"allowedEntities":{"organization":["north"],"project":["9007199254740993","102"]}`)) {
		t.Fatalf("claim was not serialized as map: %s", payload)
	}
	facts, err := provider.Resolve(WithBearer(context.Background(), token))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := facts.IDsForType("project"); err != nil || len(got) != 2 || got[0] != "9007199254740993" {
		t.Fatalf("canonical IDs: %v %v", got, err)
	}
	numericToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "alice", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "one", "allowedEntities": map[string]any{"project": []any{uint64(9007199254740993), uint64(18446744073709551615)}, "empty": []any{}}}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	numericFacts, err := provider.Resolve(WithBearer(context.Background(), numericToken))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := numericFacts.IDsForType("project"); err != nil || len(got) != 2 || got[0] != "9007199254740993" || got[1] != "18446744073709551615" {
		t.Fatalf("numeric precision: %v %v", got, err)
	}
	if got, err := numericFacts.IDsForType("empty"); err != nil || len(got) != 0 {
		t.Fatalf("empty entity map key: %v %v", got, err)
	}
	legacyGo := Claims{RegisteredClaims: base, Tenant: "one", AllowedEntities: []access.Entity{{Type: "project", ID: "101"}}}
	legacyToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, legacyGo).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	legacyPayload, _ := base64.RawURLEncoding.DecodeString(strings.Split(legacyToken, ".")[1])
	if !bytes.Contains(legacyPayload, []byte(`"allowedEntities":{"project":["101"]}`)) {
		t.Fatalf("legacy Go value did not serialize canonically: %s", legacyPayload)
	}
	legacyWire, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "alice", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "one", "allowedEntities": []map[string]any{{"type": "project", "id": "101"}}}).SignedString(private)
	if err != nil {
		t.Fatal(err)
	}
	facts, err = provider.Resolve(WithBearer(context.Background(), legacyWire))
	if err != nil || len(facts.Entities) != 1 || facts.EntityGroups["project"][0] != "101" {
		t.Fatalf("legacy wire: %+v %v", facts, err)
	}
	encoded, err := json.Marshal(facts)
	if err != nil || !bytes.Contains(encoded, []byte(`"allowedEntities":{"project":["101"]}`)) {
		t.Fatalf("legacy output not canonical: %s %v", encoded, err)
	}
}

func TestSignedEntityMapRejectsMalformedAuthority(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := New(Config{Issuer: "issuer", Audience: "audience", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwt.Token) (any, error) { return public, nil }})
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range []any{
		map[string]any{"project": []any{1, "1"}},
		map[string]any{"project": []any{float64(1.5)}},
		map[string]any{"project": nil},
		map[string]any{"": []any{1}},
		[]map[string]any{{"type": "project", "id": "101"}, {"type": "project", "id": "101"}},
	} {
		claims := jwt.MapClaims{"iss": "issuer", "aud": "audience", "sub": "alice", "exp": time.Now().Add(time.Minute).Unix(), "tenant": "one", "allowedEntities": claim}
		token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(private)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.Resolve(WithBearer(context.Background(), token)); err == nil {
			t.Fatalf("accepted malformed claim: %#v", claim)
		}
	}
	var claims Claims
	if err := json.Unmarshal([]byte(`{"iss":"issuer","allowedEntities":{"project":[1]},"allowedEntities":{"project":[2]}}`), &claims); err == nil {
		t.Fatal("duplicate authority claim accepted")
	}
	if err := json.Unmarshal([]byte(`{"iss":"issuer","allowedEntities":{"project":[1]},"AllowedEntities":{"project":[2]}}`), &claims); err == nil {
		t.Fatal("case-variant authority claim accepted")
	}
}

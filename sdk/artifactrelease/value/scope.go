package value

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	artifactrelease "github.com/viant/forge/backend/artifactrelease"
)

type Scope struct {
	Tenant, Account, Workspace, Namespace string
}

func (s Scope) Valid() bool {
	return (artifactrelease.Key{Tenant: s.Tenant, Account: s.Account, Workspace: s.Workspace, Namespace: s.Namespace, Name: "scope"}).Valid()
}

func (s Scope) ID() string {
	raw, _ := json.Marshal([]string{s.Tenant, s.Account, s.Workspace, s.Namespace})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s Scope) Owns(key artifactrelease.Key) bool {
	return key.Valid() && key.Tenant == s.Tenant && key.Account == s.Account && key.Workspace == s.Workspace && key.Namespace == s.Namespace
}

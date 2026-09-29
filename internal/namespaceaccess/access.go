// Package namespaceaccess enforces the boundary shared by Studio windows and
// namespace-specific runtime listeners. Namespace grants never bypass resource ACL.
package namespaceaccess

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/viant/authz"
	"strings"
	"time"
)

const Private = "private"
const Public = "public"

// ID is stable for the immutable owner/name namespace key.
func ID(owner, name string) string {
	sum := sha256.Sum256([]byte(owner + "\x00" + name))
	return hex.EncodeToString(sum[:])
}

type Policy struct {
	ID         string
	OwnerID    string
	Visibility string
	Roles      []string
}

func (p Policy) CanView(facts authz.Facts, now time.Time) bool {
	if p.ID == "" || p.OwnerID == "" {
		return false
	}
	if p.Visibility == Public {
		return true
	}
	if p.Visibility != "" && p.Visibility != Private {
		return false
	}
	if facts.Subject == "" || facts.Issuer == "" || !facts.ValidUntil.After(now) {
		return false
	}
	if facts.Subject == p.OwnerID {
		return true
	}
	for _, required := range p.Roles {
		if required == "" || strings.TrimSpace(required) != required {
			continue
		}
		for _, role := range facts.Roles {
			if role == required {
				return true
			}
		}
	}
	return false
}
func (p Policy) CanManage(facts authz.Facts, now time.Time) bool {
	return p.ID != "" && p.OwnerID != "" && facts.Subject == p.OwnerID && facts.Issuer != "" && facts.ValidUntil.After(now)
}

// Contains rejects cross-namespace requests even when the caller owns both.
func Contains(current, row string) bool { return current != "" && row != "" && current == row }

// Validate checks deployment metadata without interpreting role names.
func Validate(visibility string, roles []string, port *int) error {
	if visibility != "" && visibility != Private && visibility != Public {
		return fmt.Errorf("namespace visibility must be private or public")
	}
	seen := map[string]bool{}
	for _, role := range roles {
		if role == "" || strings.TrimSpace(role) != role || seen[role] {
			return fmt.Errorf("assigned roles must be distinct nonempty names")
		}
		seen[role] = true
	}
	if port != nil && (*port < 0 || *port > 65535) {
		return fmt.Errorf("MCP port must be 0 for automatic allocation or 1–65535")
	}
	return nil
}

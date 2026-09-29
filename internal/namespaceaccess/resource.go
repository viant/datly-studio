package namespaceaccess

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// ValidateResourceOwnership requires canonical ownership on new writes. An empty
// legacy value may be repaired by the trusted component-derived assignment.
func ValidateResourceOwnership(current, previous string) error {
	decoded, err := hex.DecodeString(current)
	if err != nil || len(decoded) != 32 || current != strings.ToLower(current) {
		return fmt.Errorf("resource requires canonical namespace ownership")
	}
	if previous != "" && previous != current {
		return fmt.Errorf("resource namespace ownership cannot change")
	}
	return nil
}

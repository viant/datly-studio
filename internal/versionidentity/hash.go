package versionidentity

import (
	"crypto/sha256"
	"fmt"
)

func Hash(reportID string, versionNo int, mode, authoredSQL, authoredDQL string, spec []byte) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%s:%s:%s", reportID, versionNo, mode, authoredSQL, authoredDQL, spec)))
	return fmt.Sprintf("%x", sum[:])
}

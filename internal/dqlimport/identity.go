package dqlimport

import (
	"crypto/sha256"
	"fmt"

	"github.com/viant/datly-studio/sdk"
)

// SpecHash binds a new version to every resource in its canonical filename
// order, not just the selected DQL entry.
func SpecHash(reportID string, versionNo int, source string, bundle *sdk.DQLBundle, files []string) string {
	bundleHash := sha256.New()
	for _, name := range files {
		fmt.Fprintf(bundleHash, "%d:%s:%d:", len(name), name, len(bundle.Files[name]))
		bundleHash.Write(bundle.Files[name])
	}
	spec := []byte(fmt.Sprintf(`{"bundleSha256":"%x"}`, bundleHash.Sum(nil)))
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%s:%s:%s", reportID, versionNo, "dql", "", source, spec)))
	return fmt.Sprintf("%x", sum[:])
}

func Namespace(reportID, ownerPackage string) string {
	digest := sha256.Sum256([]byte(reportID))
	return fmt.Sprintf("%s.imports.%x", ownerPackage, digest[:8])
}

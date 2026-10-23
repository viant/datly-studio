package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"sync"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/resourceartifact"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/typecatalog"
)

// Bump when Studio changes how it constructs transcribe.Source or consumes the
// resolved contract. Datly artifact format and published Datly module version
// are included separately below; local devel builds use executable SHA-256.
const studioReaderCompilerIdentity = "studio.dynamic-reader-compiler/2026-10-v1"

// compilerInputIdentity is calculated entirely from already admitted Studio
// publication/configuration rows and in-memory linked types. It never probes a
// business database. Changes to external schema require a new published
// source/connector revision; cached published versions do not auto-discover it.
func compilerInputIdentity(namespaceID, modulePath, rootDir string, definition definition, resourceHash, constantsHash string,
	connectors map[string]connectorDefinition, catalog *typecatalog.Catalog) (resourceartifact.Identity, string, error) {
	if definition.reportID == "" || definition.versionNo <= 0 || resourceHash == "" {
		return resourceartifact.Identity{}, "", fmt.Errorf("complete published reader identity is required")
	}
	owner := namespaceID
	if owner == "" {
		// Legacy unpartitioned Studio databases still need a deterministic
		// internal writer namespace. report_id remains globally unique there.
		owner = namespaceaccess.ID("studio-legacy-compiler", definition.reportID)
	}
	sourceURI := fmt.Sprintf("component://%s/%s@%d", owner, definition.reportID, definition.versionNo)
	catalogHash, err := linkedCatalogIdentity(catalog)
	if err != nil {
		return resourceartifact.Identity{}, "", err
	}
	type connectorIdentity struct {
		Name, Driver, DSNHash, SecretRefHash, OptionsHash string
		Etag                                              int64
	}
	names := make([]string, 0, len(connectors))
	for name := range connectors {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]connectorIdentity, 0, len(names))
	for _, name := range names {
		item := connectors[name]
		items = append(items, connectorIdentity{Name: name, Driver: item.driver,
			DSNHash: privateSHA(item.dsn), SecretRefHash: privateSHA(item.secretRef),
			OptionsHash: privateSHA(string(item.options)), Etag: item.etag})
	}
	buildIdentity, err := datlyCompilerBuildIdentity()
	if err != nil {
		return resourceartifact.Identity{}, "", err
	}
	raw, err := json.Marshal(struct {
		ArtifactVersion, Compiler, Build, ModulePath, RootDirHash, SourceURI, DQL, Scope, Name, Connector, Driver, ResourceHash, ConstantsHash, CatalogHash string
		Connectors                                                                                                                                          []connectorIdentity
	}{transcribe.RuntimeContractArtifactVersion, studioReaderCompilerIdentity, buildIdentity, modulePath, privateSHA(rootDir), sourceURI,
		definition.dql, definition.scope, definition.name, definition.connector, definition.driver,
		resourceHash, constantsHash, catalogHash, items})
	if err != nil {
		return resourceartifact.Identity{}, "", err
	}
	inputHash := sha256.Sum256(raw)
	identity, err := resourceartifact.New(sourceURI, hex.EncodeToString(inputHash[:]))
	return identity, owner, err
}

var localCompilerBinaryIdentity struct {
	sync.Once
	hash string
	err  error
}

func datlyCompilerBuildIdentity() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return localExecutableIdentity()
	}
	for _, dependency := range info.Deps {
		if dependency.Path != "github.com/viant/datly" {
			continue
		}
		if dependency.Replace == nil && dependency.Version != "" && dependency.Version != "(devel)" {
			return dependency.Version, nil
		}
		if dependency.Replace != nil && dependency.Replace.Version != "" && dependency.Replace.Version != "(devel)" {
			return dependency.Replace.Path + "@" + dependency.Replace.Version, nil
		}
		break
	}
	return localExecutableIdentity()
}

func localExecutableIdentity() (string, error) {
	// Local go.work builds have no immutable dependency version. Hash the
	// complete executable once, so edited compiler code cannot reuse bytes
	// from another local binary and no reader repeats the file read.
	localCompilerBinaryIdentity.Do(func() {
		path, err := os.Executable()
		if err != nil {
			localCompilerBinaryIdentity.err = err
			return
		}
		file, err := os.Open(path)
		if err != nil {
			localCompilerBinaryIdentity.err = err
			return
		}
		defer file.Close()
		h := sha256.New()
		if _, err = io.Copy(h, file); err != nil {
			localCompilerBinaryIdentity.err = err
			return
		}
		localCompilerBinaryIdentity.hash = hex.EncodeToString(h.Sum(nil))
	})
	if localCompilerBinaryIdentity.err != nil || localCompilerBinaryIdentity.hash == "" {
		return "", fmt.Errorf("local compiler executable identity is unavailable")
	}
	return "executable-sha256:" + localCompilerBinaryIdentity.hash, nil
}

func privateSHA(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func linkedCatalogIdentity(catalog *typecatalog.Catalog) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("linked type catalog is required")
	}
	h := sha256.New()
	for _, authority := range []typecatalog.Authority{typecatalog.PackageAuthority, typecatalog.TranscribeAuthority} {
		registry, err := catalog.Registry(authority)
		if err != nil {
			return "", err
		}
		keys := append([]string(nil), registry.Keys()...)
		sort.Strings(keys)
		for _, key := range keys {
			descriptor := registry.Lookup(key)
			if descriptor == nil || descriptor.Type == nil || descriptor.SynteticType != nil || descriptor.Definition != "" {
				return "", fmt.Errorf("catalog type %q lacks portable runtime authority", key)
			}
			origin, _, err := catalog.Origin(authority, key)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(h, "%d:%s%d:%s%d:%s", len(authority), authority, len(key), key, len(origin), origin)
			fmt.Fprint(h, transcribe.RuntimeTypeFingerprint(descriptor.Type))
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

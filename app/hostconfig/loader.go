package hostconfig

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/viant/afs"
	"github.com/viant/afs/storage"
	"github.com/viant/datly/bootstrap/connector"
	datly "github.com/viant/datly/standalone/config"
)

// Load reads one shared configuration and delegates native source resolution to
// Datly's loader. The captured native document retains the original URL, so
// relative connector/dependency/constant references keep Datly semantics. No
// temporary configuration files or secret material are written to disk.
func Load(ctx context.Context, location string) (*Config, error) {
	if ctx == nil {
		return nil, errors.New("host configuration context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(location) == "" {
		return nil, errors.New("host configuration URL is required")
	}
	if !strings.Contains(location, "://") {
		absolute, err := filepath.Abs(location)
		if err != nil {
			return nil, errors.New("invalid host configuration path")
		}
		location = absolute
	}
	fs := afs.New()
	data, err := fs.DownloadWithURL(ctx, location)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("host configuration could not be loaded")
	}
	return LoadDocument(ctx, location, data)
}

// LoadDocument resolves an already captured document at its original URL. Hosts
// can remove private extension fields without temporary files or a second read.
func LoadDocument(ctx context.Context, location string, data []byte) (*Config, error) {
	if ctx == nil {
		return nil, errors.New("host configuration context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(location) == "" {
		return nil, errors.New("host configuration URL is required")
	}
	if !strings.Contains(location, "://") {
		var err error
		location, err = filepath.Abs(location)
		if err != nil {
			return nil, err
		}
	}
	fs := afs.New()
	result, err := Decode(data)
	if err != nil {
		return nil, err
	}
	store, err := result.CatalogConnector()
	if err != nil {
		return nil, err
	}
	nativeConfig := result.Config
	nativeConfig.Connector = store.Name
	nativeConfig.Connectors = []connector.Config{store}
	nativeDocument, err := json.Marshal(&nativeConfig)
	if err != nil {
		return nil, errors.New("invalid native host configuration")
	}
	snapshot := &documentFS{Service: fs, location: location, data: nativeDocument}
	native, err := (datly.Loader{FS: snapshot}).Load(ctx, location)
	if err != nil {
		return nil, err
	}
	if len(native.Connectors) != 1 {
		return nil, errors.New("additional storage connectors are not supported; use singular Store")
	}
	native, err = native.ResolveConstants()
	if err != nil {
		return nil, err
	}
	resolvedStore := native.Connectors[0]
	result.Store = &resolvedStore
	native.Connector = ""
	native.Connectors = nil
	result.Config = *native
	if result.StaticRoot != "" && !filepath.IsAbs(result.StaticRoot) {
		result.StaticRoot = filepath.Join(native.BaseDir, result.StaticRoot)
	}
	return result, nil
}

// documentFS replaces only the already-read root document. Datly reads all
// dependency resources through the original filesystem implementation.
type documentFS struct {
	afs.Service
	location string
	data     []byte
}

func (s *documentFS) DownloadWithURL(ctx context.Context, location string, options ...storage.Option) ([]byte, error) {
	if location == s.location {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return append([]byte(nil), s.data...), nil
	}
	return s.Service.DownloadWithURL(ctx, location, options...)
}

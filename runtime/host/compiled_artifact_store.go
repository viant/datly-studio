package host

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/internal/resourceartifact"
	stored "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	write "github.com/viant/datly-studio/studio/report_resource_files/store_write"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	"github.com/viant/datly/tag"
)

var (
	errCompiledArtifactCorrupt  = errors.New("compiled artifact bytes are invalid")
	errCompiledArtifactConflict = errors.New("compiled artifact identity already contains different bytes")
	errCompiledArtifactTooLarge = errors.New("compiled artifact exceeds resource store's 1 MiB text limit")
)

// compiledArtifactStore uses the existing component_resource_files table and
// its generated internal reader/writer. It never changes a component source
// revision and never registers an HTTP/MCP route.
type compiledArtifactStore struct {
	mu      sync.RWMutex
	runtime *druntime.Runtime
	read    dexec.ComponentTarget
	write   dexec.ComponentTarget
}

func newCompiledArtifactStore(db *sql.DB) (*compiledArtifactStore, error) {
	if db == nil {
		return nil, fmt.Errorf("Studio resource database is required")
	}
	resources := resource.New()
	if err := resources.Register(stored.FileDatlyResourceNamespace, stored.FileDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(write.FileDatlyResourceNamespace, write.FileDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	reader, readTarget, err := readercomponent.Compile(reflect.TypeOf(stored.FileComponent{}), "store_snapshot",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	writer, writeTarget, err := compiledFileWriter(db, connector, resources)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{reader, writer}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &compiledArtifactStore{runtime: runtime, read: readTarget, write: writeTarget}, nil
}

func compiledFileWriter(db *sql.DB, connector *dsql.SQLComponent, resources *resource.Store) (*registry.RegisteredComponent, dexec.ComponentTarget, error) {
	holder := reflect.TypeOf(write.FileComponent{})
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("compiled resource writer contract is unavailable")
	}
	metadata, present, err := tag.ParseComponent(field.Tag)
	if err != nil || !present {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("compiled resource writer metadata: %w", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "store_write", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(write.Input{}), reflect.TypeOf(write.Output{}))
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(write.Input{}), OutputType: reflect.TypeOf(write.Output{}), Resources: resources})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	var providers []locator.Provider
	if len(artifact.ViewDependencies) > 0 {
		views, e := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
		if e != nil {
			return nil, dexec.ComponentTarget{}, e
		}
		providers = append(providers, views)
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(write.Input{}), reflect.TypeOf(write.Output{}), "patch")
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	registration := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: reflect.TypeOf(write.Output{}), Handler: handler,
		Providers: providers, DataSource: dml.Source{DB: db}}
	registration.Capabilities.Connector = connector
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	return registration, target, nil
}

func (s *compiledArtifactStore) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Shutdown(ctx)
}

func (s *compiledArtifactStore) Get(ctx context.Context, reportID string, versionNo int, identity resourceartifact.Identity) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("compiled artifact store is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.get(ctx, reportID, versionNo, identity)
}

func (s *compiledArtifactStore) get(ctx context.Context, reportID string, versionNo int, identity resourceartifact.Identity) ([]byte, error) {
	if s == nil || s.runtime == nil || reportID == "" || versionNo <= 0 || identity.Key == "" {
		return nil, fmt.Errorf("compiled artifact store identity is required")
	}
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo, ResourceId: identity.Key,
		Has: &stored.InputHas{ReportId: true, VersionNo: true, ResourceId: true}}
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.read, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil || len(output.Files) > 1 {
		return nil, errCompiledArtifactCorrupt
	}
	if len(output.Files) == 0 {
		return nil, sql.ErrNoRows
	}
	file := output.Files[0]
	if file == nil || file.ReportId != reportID || file.VersionNo != versionNo || file.ResourceId != identity.Key ||
		file.Namespace != resourceartifact.Namespace || file.ResourcePath != identity.Path || file.IsBinary {
		return nil, errCompiledArtifactCorrupt
	}
	if file.MediaType == nil || *file.MediaType != resourceartifact.MediaType {
		return nil, errCompiledArtifactCorrupt
	}
	hash := sha256.Sum256(file.Content)
	if file.ContentSha256 != hex.EncodeToString(hash[:]) || file.ContentSize != int64(len(file.Content)) {
		return append([]byte(nil), file.Content...), errCompiledArtifactCorrupt
	}
	return append([]byte(nil), file.Content...), nil
}

func (s *compiledArtifactStore) Put(ctx context.Context, namespaceID, reportID string, versionNo int, identity resourceartifact.Identity, payload []byte) error {
	return s.put(ctx, namespaceID, reportID, versionNo, identity, payload, "")
}

func (s *compiledArtifactStore) Repair(ctx context.Context, namespaceID, reportID string, versionNo int, identity resourceartifact.Identity, payload []byte, expectedPreviousSHA string) error {
	if len(expectedPreviousSHA) != 64 {
		return errCompiledArtifactConflict
	}
	return s.put(ctx, namespaceID, reportID, versionNo, identity, payload, expectedPreviousSHA)
}

func (s *compiledArtifactStore) put(ctx context.Context, namespaceID, reportID string, versionNo int, identity resourceartifact.Identity, payload []byte, expectedPreviousSHA string) error {
	if s == nil {
		return fmt.Errorf("compiled artifact store is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(payload) > 1<<20 {
		return errCompiledArtifactTooLarge
	}
	if s == nil || s.runtime == nil || namespaceID == "" || reportID == "" || versionNo <= 0 || identity.Key == "" {
		return fmt.Errorf("compiled artifact store identity is required")
	}
	prior, err := s.get(ctx, reportID, versionNo, identity)
	if err == nil {
		if string(prior) == string(payload) {
			return nil
		}
		if expectedPreviousSHA == "" {
			return errCompiledArtifactConflict
		}
	}
	if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, errCompiledArtifactCorrupt) && err != nil {
		return err
	}
	if expectedPreviousSHA != "" && errors.Is(err, sql.ErrNoRows) {
		return errCompiledArtifactConflict
	}
	if len(prior) > 0 {
		previous := sha256.Sum256(prior)
		if expectedPreviousSHA != hex.EncodeToString(previous[:]) {
			return errCompiledArtifactConflict
		}
	}
	digest := sha256.Sum256(payload)
	mediaType := resourceartifact.MediaType
	row := &write.StoredFile{NamespaceId: namespaceID, ReportId: reportID, VersionNo: versionNo, ResourceId: identity.Key,
		Namespace: resourceartifact.Namespace, ResourcePath: identity.Path, MediaType: &mediaType,
		Content: append([]byte(nil), payload...), ContentSize: int64(len(payload)), ContentSha256: hex.EncodeToString(digest[:]),
		CreatedAt: time.Now().UTC(), Has: &write.StoredFileHas{NamespaceId: true, ReportId: true, VersionNo: true,
			ResourceId: true, Namespace: true, ResourcePath: true, MediaType: true, Content: true,
			ContentSize: true, ContentSha256: true, IsBinary: true, CreatedAt: true, ShouldDelete: true}}
	input := &write.Input{}
	input.SetFiles([]*write.StoredFile{row})
	writeContext := resourceartifact.WithTrustedWrite(ctx)
	if expectedPreviousSHA != "" {
		writeContext = resourceartifact.WithTrustedRepair(ctx, expectedPreviousSHA)
	}
	_, err = s.runtime.InvokeComponent(writeContext, dexec.ComponentRequest{Target: s.write, Input: input})
	if err == nil {
		return nil
	}
	// Another compiler may have published the same immutable identity first.
	prior, readErr := s.get(ctx, reportID, versionNo, identity)
	if readErr == nil && string(prior) == string(payload) {
		return nil
	}
	if readErr == nil {
		return errCompiledArtifactConflict
	}
	return err
}

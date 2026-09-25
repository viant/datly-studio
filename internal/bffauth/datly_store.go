package bffauth

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	expiredreader "github.com/viant/datly-studio/studio/bff_sessions/store_expired"
	storedreader "github.com/viant/datly-studio/studio/bff_sessions/store_read"
	storedwriter "github.com/viant/datly-studio/studio/bff_sessions/store_write"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
)

type sessionComponents struct {
	runtime *druntime.Runtime
	read    dexec.ComponentTarget
	expired dexec.ComponentTarget
	write   dexec.ComponentTarget
}

// readDatlySession executes the generated, server-only Datly v1 reader. It
// never exposes the encrypted payload through the browser SDK or MCP routes.
func (s *SQLStore) readDatlySession(ctx context.Context, hash string) ([]byte, int64, bool, error) {
	row, err := s.readDatlySessionRecord(ctx, hash)
	if err != nil || row == nil {
		return nil, 0, false, err
	}
	return row.PayloadCiphertext, row.ExpiresAtUnix, true, nil
}

func (s *SQLStore) readDatlySessionRecord(ctx context.Context, hash string) (*storedreader.StoredSession, error) {
	components, err := s.loadComponents()
	if err != nil {
		return nil, err
	}
	value, err := components.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: components.read,
		Input: &storedreader.Input{SessionIdHash: hash, Has: &storedreader.InputHas{SessionIdHash: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedreader.Output)
	if !ok {
		return nil, fmt.Errorf("BFF session reader returned %T", value)
	}
	if len(output.Sessions) == 0 {
		return nil, nil
	}
	if len(output.Sessions) != 1 || output.Sessions[0] == nil || output.Sessions[0].SessionIdHash != hash {
		return nil, fmt.Errorf("BFF session reader returned an ambiguous identity")
	}
	return output.Sessions[0], nil
}

func (s *SQLStore) loadComponents() (*sessionComponents, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.components != nil {
		return s.components, nil
	}
	resources := resource.New()
	if err := resources.Register(storedreader.SessionDatlyResourceNamespace, storedreader.SessionDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(expiredreader.SessionDatlyResourceNamespace, expiredreader.SessionDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(storedwriter.SessionDatlyResourceNamespace, storedwriter.SessionDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: s.db}
	if err := connector.RegisterConnector("studio", s.db); err != nil {
		return nil, err
	}
	readerComponent, err := sessionComponent(reflect.TypeOf(storedreader.SessionComponent{}), "store_read", reflect.TypeOf(storedreader.Input{}), reflect.TypeOf(storedreader.Output{}))
	if err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: readerComponent,
		InputType: reflect.TypeOf(storedreader.Input{}), OutputType: reflect.TypeOf(storedreader.Output{}), Resources: resources})
	if err != nil {
		return nil, err
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		return nil, err
	}
	readerRegistration, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		return nil, err
	}
	expiredComponent, err := sessionComponent(reflect.TypeOf(expiredreader.SessionComponent{}), "store_expired", reflect.TypeOf(expiredreader.Input{}), reflect.TypeOf(expiredreader.Output{}))
	if err != nil {
		return nil, err
	}
	expiredArtifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: expiredComponent,
		InputType: reflect.TypeOf(expiredreader.Input{}), OutputType: reflect.TypeOf(expiredreader.Output{}), Resources: resources})
	if err != nil {
		return nil, err
	}
	expiredExecution, err := expiredArtifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		return nil, err
	}
	expiredRegistration, err := expiredArtifact.Registration(registry.RegisteredComponent{Reader: expiredExecution})
	if err != nil {
		return nil, err
	}
	writerComponent, err := sessionComponent(reflect.TypeOf(storedwriter.SessionComponent{}), "store_write", reflect.TypeOf(storedwriter.Input{}), reflect.TypeOf(storedwriter.Output{}))
	if err != nil {
		return nil, err
	}
	writerArtifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: writerComponent,
		InputType: reflect.TypeOf(storedwriter.Input{}), OutputType: reflect.TypeOf(storedwriter.Output{}), Resources: resources})
	if err != nil {
		return nil, err
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: writerArtifact.ViewDependencies, Input: writerArtifact.Input, SQL: connector})
	if err != nil {
		return nil, err
	}
	handler, err := writerhandler.New(writerComponent, reflect.TypeOf(storedwriter.Input{}), reflect.TypeOf(storedwriter.Output{}), "patch")
	if err != nil {
		return nil, err
	}
	writerRegistration := &registry.RegisteredComponent{Component: writerArtifact.Component, Input: writerArtifact.Input,
		Output: writerArtifact.Output, OutputType: reflect.TypeOf(storedwriter.Output{}), Handler: handler,
		Providers: []locator.Provider{views}, DataSource: dml.Source{DB: s.db}}
	writerRegistration.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{readerRegistration, expiredRegistration, writerRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	s.components = &sessionComponents{runtime: runtime, read: sessionTarget(readerComponent), expired: sessionTarget(expiredComponent), write: sessionTarget(writerComponent)}
	return s.components, nil
}

func sessionComponent(holder reflect.Type, packageName string, inputType, outputType reflect.Type) (*spec.Component, error) {
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, fmt.Errorf("BFF session %s has no component contract", packageName)
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return nil, fmt.Errorf("BFF session %s component metadata: %w", packageName, err)
	}
	if !present {
		return nil, fmt.Errorf("BFF session %s has no component metadata", packageName)
	}
	return (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: packageName, PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: inputType.Name(), OutputType: outputType.Name()}).Resolve(inputType, outputType)
}

func sessionTarget(component *spec.Component) dexec.ComponentTarget {
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	return target
}

func (s *SQLStore) writeDatlySession(ctx context.Context, hash, subject string, ciphertext []byte, expires int64, created time.Time, deleteRow bool) error {
	row := &storedwriter.StoredSession{SessionIdHash: hash, ShouldDelete: deleteRow,
		Has: &storedwriter.StoredSessionHas{SessionIdHash: true, ShouldDelete: deleteRow}}
	if !deleteRow {
		row.SubjectId, row.PayloadCiphertext, row.ExpiresAtUnix, row.CreatedAt = subject, ciphertext, expires, &created
		row.Has.SubjectId, row.Has.PayloadCiphertext, row.Has.ExpiresAtUnix, row.Has.CreatedAt = true, true, true, true
	}
	return s.writeDatlyRows(ctx, []*storedwriter.StoredSession{row})
}

func (s *SQLStore) writeDatlyRows(ctx context.Context, rows []*storedwriter.StoredSession) error {
	components, err := s.loadComponents()
	if err != nil {
		return err
	}
	_, err = components.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: components.write,
		Input: &storedwriter.Input{Sessions: rows, Has: &storedwriter.InputHas{Sessions: true}}})
	return err
}

// deleteExpiredDatly bounds each cleanup request so a large backlog cannot
// delay login indefinitely. Later logins (or a scheduled cleanup) continue it.
func (s *SQLStore) deleteExpiredDatly(ctx context.Context, cutoff int64) error {
	const batchSize, maxBatches = 256, 16
	for batch := 0; batch < maxBatches; batch++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		hashes, err := s.expiredSessionHashes(ctx, cutoff)
		if err != nil {
			return err
		}
		if len(hashes) == 0 {
			return nil
		}
		rows := make([]*storedwriter.StoredSession, 0, len(hashes))
		for _, hash := range hashes {
			rows = append(rows, &storedwriter.StoredSession{SessionIdHash: hash, ShouldDelete: true,
				Has: &storedwriter.StoredSessionHas{SessionIdHash: true, ShouldDelete: true}})
		}
		if err := s.writeDatlyRows(ctx, rows); err != nil {
			// A different BFF instance can delete one of the keys between the
			// bounded read and the managed writer transaction. Retry each key
			// through the idempotent postcondition check in that case.
			for _, hash := range hashes {
				if retryErr := s.deleteHashedSession(ctx, hash); retryErr != nil {
					return retryErr
				}
			}
		}
		if len(hashes) < batchSize {
			return nil
		}
	}
	return nil
}

func (s *SQLStore) expiredSessionHashes(ctx context.Context, cutoff int64) ([]string, error) {
	components, err := s.loadComponents()
	if err != nil {
		return nil, err
	}
	value, err := components.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: components.expired,
		Input: &expiredreader.Input{ExpiresBefore: cutoff, Has: &expiredreader.InputHas{ExpiresBefore: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*expiredreader.Output)
	if !ok || len(output.Sessions) > 256 {
		return nil, fmt.Errorf("BFF expired-session reader returned invalid result %T", value)
	}
	hashes := make([]string, 0, len(output.Sessions))
	for _, row := range output.Sessions {
		if row == nil || row.SessionIdHash == "" {
			return nil, fmt.Errorf("BFF expired-session reader returned an invalid key")
		}
		hashes = append(hashes, row.SessionIdHash)
	}
	return hashes, nil
}

func (s *SQLStore) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.components == nil {
		return nil
	}
	err := s.components.runtime.Shutdown(ctx)
	s.components = nil
	return err
}

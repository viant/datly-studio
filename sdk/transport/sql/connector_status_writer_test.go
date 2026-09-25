package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/connectors/store_status"
	_ "modernc.org/sqlite"
)

func TestConnectorStatusWriterUsesMatchedEtagAndCurrentProbe(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	now, expected := time.Now().UTC(), created.ETag
	err = transport.writeConnectorStatus(ctx, "activate", &stored.StoredConnector{Name: created.Name, Etag: &expected, UpdatedAt: &now,
		Has: &stored.StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true}})
	if !errors.Is(err, stored.ErrProbeRequired) {
		t.Fatalf("transcribed hook allowed activation without probe: %v", err)
	}
	disabled, err := client.Connectors().Disable(ctx, created.Name, created.ETag)
	if err != nil || disabled.Status != "disabled" || disabled.ETag != created.ETag+1 {
		t.Fatalf("disabled=%+v err=%v", disabled, err)
	}
	_, err = client.Connectors().Disable(ctx, created.Name, created.ETag)
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("stale status error=%v, want conflict", err)
	}
	_, err = client.Connectors().Activate(ctx, created.Name, disabled.ETag)
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorInvalidArgument {
		t.Fatalf("activation without probe error=%v, want invalid argument", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE connectors SET last_test_status='passed' WHERE name='source'`); err != nil {
		t.Fatal(err)
	}
	active, err := client.Connectors().Activate(ctx, created.Name, disabled.ETag)
	if err != nil || active.Status != "active" || active.ETag != disabled.ETag+1 {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if err := client.Connectors().Delete(ctx, active.Name, active.ETag); err != nil {
		t.Fatal(err)
	}
	_, err = client.Connectors().Disable(ctx, active.Name, active.ETag)
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("deleted connector status error=%v, want conflict", err)
	}
}

func TestConnectorProbeRejectsStaleConfiguration(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	transport.Probe = sdk.ConnectorProbeFunc(func(ctx context.Context, current *sdk.Connector) (*sdk.ConnectorTestResult, error) {
		dsn := "file:changed.db"
		if _, err := client.Connectors().Update(ctx, current.Name, sdk.UpdateConnectorInput{DSNTemplate: &dsn, ETag: current.ETag}); err != nil {
			return nil, err
		}
		return &sdk.ConnectorTestResult{Status: "passed"}, nil
	})
	_, err = client.Connectors().Test(ctx, created.Name)
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("probe of superseded configuration error=%v, want conflict", err)
	}
	current, err := client.Connectors().Get(ctx, created.Name)
	if err != nil || current.ETag != created.ETag+1 || current.LastTestStatus != "" || current.Status != "draft" {
		t.Fatalf("stale probe changed connector=%+v err=%v", current, err)
	}
	transport.Probe = sdk.ConnectorProbeFunc(func(context.Context, *sdk.Connector) (*sdk.ConnectorTestResult, error) {
		return &sdk.ConnectorTestResult{Status: "passed"}, nil
	})
	if result, err := client.Connectors().Test(ctx, created.Name); err != nil || result.Status != "passed" {
		t.Fatalf("current probe=%+v err=%v", result, err)
	}
	current, err = client.Connectors().Get(ctx, created.Name)
	if err != nil || current.LastTestStatus != "passed" || current.ETag != created.ETag+1 {
		t.Fatalf("current probe persistence=%+v err=%v", current, err)
	}
	if active, err := client.Connectors().Activate(ctx, created.Name, current.ETag); err != nil || active.Status != "active" {
		t.Fatalf("activation after current probe=%+v err=%v", active, err)
	}
}

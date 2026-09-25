package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

type blockedVersionValidator struct {
	started chan struct{}
	release chan struct{}
}

func (validator blockedVersionValidator) Validate(ctx context.Context, _ string, _ int) error {
	validator.started <- struct{}{}
	select {
	case <-validator.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestValidationCannotMarkEditedSourceValid(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	validator := blockedVersionValidator{started: make(chan struct{}, 1), release: make(chan struct{})}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}, Validator: validator}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "validation-race", Title: "Validation race", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithTimeout(owner, 10*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, validationErr := client.Versions().Validate(runCtx, report.ID, version.VersionNo)
		result <- validationErr
	}()
	select {
	case <-validator.started:
	case <-runCtx.Done():
		t.Fatal(runCtx.Err())
	}
	updated, editErr := client.Versions().Apply(owner, report.ID, version.VersionNo, sdk.EditCommand{
		Kind: "set_dql", ExpectedSourceRevision: version.SourceRevision,
		Payload: json.RawMessage(`{"authoredDql":"SELECT 2"}`),
	})
	close(validator.release)
	if editErr != nil || updated.Version.SourceRevision != version.SourceRevision+1 {
		t.Fatalf("concurrent edit=%+v err=%v", updated, editErr)
	}
	select {
	case validationErr := <-result:
		var sdkErr *sdk.Error
		if !errors.As(validationErr, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
			t.Fatalf("stale validation error=%v", validationErr)
		}
	case <-runCtx.Done():
		t.Fatal(runCtx.Err())
	}
	current, err := client.Versions().Get(owner, report.ID, version.VersionNo)
	if err != nil || current.SourceRevision != version.SourceRevision+1 ||
		current.AuthoredDQL != "SELECT 2" || current.CompileStatus != "pending" {
		t.Fatalf("edited source was overwritten by validation: version=%+v err=%v", current, err)
	}
}

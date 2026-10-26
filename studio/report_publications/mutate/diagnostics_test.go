package mutate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
	_ "modernc.org/sqlite"
)

type publicationDBProvider struct{ db *sql.DB }

func (p publicationDBProvider) Connector(_ context.Context, name string) (*sql.DB, error) {
	if name != "studio" {
		return nil, errors.New("unexpected connector")
	}
	return p.db, nil
}

type publicationFixtureBinder struct{ db *sql.DB }

func (b publicationFixtureBinder) Lookup(_ context.Context, key xhandler.ValueKey) (any, bool, error) {
	if key == rhandler.ConnectorCapabilityKey {
		return publicationDBProvider{b.db}, true, nil
	}
	return nil, false, nil
}
func (b publicationFixtureBinder) Bind(_ context.Context, target any) error {
	field := reflect.ValueOf(target).Elem().FieldByName("Authorization")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
		return errors.New("unexpected publication binding")
	}
	field.SetString("Bearer verified-credential")
	return nil
}

type publicationFixtureSession struct {
	xhandler.Session
	binder xhandler.Binder
}

func (s publicationFixtureSession) Binder() xhandler.Binder { return s.binder }

func publicationAPIFixture(t *testing.T) (context.Context, publicationFixtureSession, *VersionInput) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatalf("fixture schema: %v", err)
	}
	setup := &sqltransport.Transport{DB: db, Authorizer: &authorization.SDKAuthorizer{DB: db}}
	t.Cleanup(func() { setup.Close(context.Background()) })
	client, err := sdk.NewClient(setup)
	if err != nil {
		t.Fatalf("fixture SDK client: %v", err)
	}
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	if _, err = client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite", DSNTemplate: "file:fixture", OwnerID: "owner"}); err != nil {
		t.Fatalf("fixture connector: %v", err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatalf("fixture connector activation: %v", err)
	}
	report, err := client.Components().Create(ctx, sdk.CreateComponentInput{ID: "publication-fixture", Slug: "publication-fixture", Title: "Publication fixture", OwnerID: "owner", DefaultConnectorName: "main", ComponentScope: "reports", ComponentName: "reader"})
	if err != nil {
		t.Fatalf("fixture component: %v", err)
	}
	version, err := client.Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1", CreatedBy: "owner", ComponentSpec: json.RawMessage(`{"views":[]}`)})
	if err != nil {
		t.Fatalf("fixture version: %v", err)
	}
	validated, err := client.Versions().Validate(ctx, report.ID, version.VersionNo, version.SourceRevision)
	if err != nil || !validated.Valid {
		t.Fatalf("fixture validation failed: error=%v valid=%v diagnostics=%+v", err, validated != nil && validated.Valid, validated)
	}
	claims := new(jwt.Claims)
	claims.Subject = "owner"
	input := &VersionInput{Jwt: claims, ReportID: report.ID, VersionNo: version.VersionNo, Input: sdk.PublishInput{RequestedBy: "owner", ExpectedSourceRevision: validated.Version.SourceRevision}}
	return accessprovider.WithProvider(context.Background(), publicationIdentity{}), publicationFixtureSession{binder: publicationFixtureBinder{db}}, input
}

func TestPublicationMutateInProcessReloadDoesNotEchoPrivateCause(t *testing.T) {
	ctx, session, input := publicationAPIFixture(t)
	private := errors.New("SELECT secret_column FROM private_table /* DEFINITION_CANARY */ password=private-password")
	calls := 0
	ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { calls++; return fmt.Errorf("reader discovery: %w", private) }, nil, nil))
	output := &Output{}
	err := NewPublish().Exec(ctx, session, input, output)
	var wireError *xresponse.Error
	if !errors.As(err, &wireError) || wireError.Code != 503 || calls != 1 {
		t.Fatalf("mutation failure classification/calls: %v/%d", err, calls)
	}
	if !errors.Is(err, private) {
		t.Fatal("actual mutation lost internal typed cause")
	}
	wire, marshalErr := json.Marshal(wireError.Payload)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, canary := range []string{"secret_column", "private_table", "DEFINITION_CANARY", "private-password"} {
		if strings.Contains(string(wire), canary) || strings.Contains(err.Error(), canary) {
			t.Fatal("actual mutate API exposed private reload cause")
		}
	}
	payload, ok := wireError.Payload.(map[string]any)
	if !ok {
		t.Fatal("missing structured mutation error")
	}
	diagnostics, ok := payload["diagnostics"].([]sdk.Diagnostic)
	if !ok || len(diagnostics) != 1 || diagnostics[0].Code != "publication_runtime_reload" || diagnostics[0].Stage != "runtime_reload" || diagnostics[0].Field != "runtime" || diagnostics[0].Hint == "" {
		t.Fatalf("not actionable: %+v", payload)
	}
	if output.Response != nil {
		t.Fatal("failed mutation returned a successful publication")
	}
}

func TestPublicationMutateSuccessRemainsCompact(t *testing.T) {
	ctx, session, input := publicationAPIFixture(t)
	calls := 0
	ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { calls++; return nil }, nil, nil))
	output := &Output{}
	if err := NewPublish().Exec(ctx, session, input, output); err != nil {
		t.Fatal(err)
	}
	if output.Response == nil || output.Response.Status != "active" || calls != 1 {
		t.Fatalf("publication/calls=%+v/%d", output.Response, calls)
	}
	raw, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{}
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"definition", "componentSpec", "authoredDql", "authoredSql", "generatedDql", "typeManifest", "resourceManifest", "Response"} {
		if _, ok := body[field]; ok {
			t.Fatalf("success echoed %s", field)
		}
	}
	for _, field := range []string{"reportId", "activeVersionNo", "desiredGeneration", "status", "specHash"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("compact receipt omitted %s", field)
		}
	}
}

func TestPublicationMutateAuthCancelAndRevisionClassificationsUnchanged(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		var out Output
		err := NewPublish().Exec(context.Background(), nil, &VersionInput{}, &out)
		var response *xresponse.Error
		if !errors.As(err, &response) || response.Code != 403 {
			t.Fatalf("auth classification: %v", err)
		}
	})
	t.Run("canceled before mutation", func(t *testing.T) {
		ctx, session, input := publicationAPIFixture(t)
		ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { return nil }, nil, nil))
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		err := NewPublish().Exec(ctx, session, input, &Output{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel classification: %v", err)
		}
	})
	t.Run("source revision conflict", func(t *testing.T) {
		ctx, session, input := publicationAPIFixture(t)
		input.Input.ExpectedSourceRevision++
		calls := 0
		ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { calls++; return nil }, nil, nil))
		err := NewPublish().Exec(ctx, session, input, &Output{})
		var response *xresponse.Error
		if !errors.As(err, &response) || response.Code != 409 || calls != 0 {
			t.Fatalf("CAS classification: %v/%d", err, calls)
		}
		payload := response.Payload.(map[string]any)
		if payload["expectedSourceRevision"] != input.Input.ExpectedSourceRevision || payload["currentSourceRevision"] != input.Input.ExpectedSourceRevision-1 {
			t.Fatal("revision fields changed")
		}
	})
	t.Run("canceled during reload", func(t *testing.T) {
		ctx, session, input := publicationAPIFixture(t)
		ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { return context.Canceled }, nil, nil))
		err := NewPublish().Exec(ctx, session, input, &Output{})
		var response *xresponse.Error
		if !errors.As(err, &response) || response.Code != 503 || !errors.Is(err, context.Canceled) {
			t.Fatalf("reload cancellation classification: %v", err)
		}
	})
}

func TestPublicationMapErrorKeepsCodesAndPrivateTypedCause(t *testing.T) {
	t.Run("projected internal diagnostics", assertPublicationMapErrorPreservesProjectedInternalDiagnostics)
	private := errors.New("SELECT private_column FROM private_table password=private-password")
	for _, tc := range []struct {
		name   string
		code   sdk.ErrorCode
		status int
	}{
		{"invalid", sdk.ErrorInvalidArgument, 400}, {"unauthorized", sdk.ErrorUnauthorized, 401}, {"forbidden", sdk.ErrorForbidden, 403}, {"missing", sdk.ErrorNotFound, 404}, {"conflict", sdk.ErrorConflict, 409}, {"unavailable", sdk.ErrorUnavailable, 503}, {"internal", sdk.ErrorInternal, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &sdk.Error{Code: tc.code, Message: "known public error", Field: "sourceRevision", ExpectedSourceRevision: 7, CurrentSourceRevision: 8, Cause: private}
			if tc.code == sdk.ErrorInternal {
				source.Message = private.Error()
			}
			mapped := mapError(source)
			var response *xresponse.Error
			if !errors.As(mapped, &response) || response.Code != tc.status || !errors.Is(mapped, private) {
				t.Fatalf("classification/cause changed: %v", mapped)
			}
			payload := response.Payload.(map[string]any)
			if payload["code"] != tc.code || payload["field"] != "sourceRevision" || payload["expectedSourceRevision"] != int64(7) || payload["currentSourceRevision"] != int64(8) {
				t.Fatal("existing error fields changed")
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "private_column") || strings.Contains(mapped.Error(), "private-password") {
				t.Fatal("private typed cause leaked")
			}
		})
	}
	mapped := mapError(private)
	var response *xresponse.Error
	if !errors.As(mapped, &response) || response.Code != 500 || !errors.Is(mapped, private) || strings.Contains(mapped.Error(), "private_column") {
		t.Fatal("unexpected error leaked or lost cause")
	}
}

func TestPublicationMutateForbiddenActorAndDeadlineStillFail(t *testing.T) {
	t.Run("other actor", func(t *testing.T) {
		ctx, session, input := publicationAPIFixture(t)
		input.Jwt.Subject = "intruder"
		input.Input.RequestedBy = "intruder"
		calls := 0
		ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { calls++; return nil }, nil, nil))
		err := NewPublish().Exec(ctx, session, input, &Output{})
		var response *xresponse.Error
		if !errors.As(err, &response) || response.Code != 403 || calls != 0 {
			t.Fatalf("forbidden actor changed: %v/%d", err, calls)
		}
	})
	t.Run("expired context", func(t *testing.T) {
		ctx, session, input := publicationAPIFixture(t)
		ctx = runtimeadmin.WithClient(ctx, runtimeadmin.NewInProcess(func(context.Context, int64) error { return nil }, nil, nil))
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer cancel()
		err := NewPublish().Exec(ctx, session, input, &Output{})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("deadline classification: %v", err)
		}
	})
}

func assertPublicationMapErrorPreservesProjectedInternalDiagnostics(t *testing.T) {
	private := errors.New("SELECT private_column FROM private_table password=private-password")
	public := sdk.Diagnostic{Severity: "error", Stage: "contract", Code: "DQL-TYPE", Field: "componentSpec.views[0]", Message: "Unsupported scalar type.", Hint: "Use a supported scalar type.", Line: 12, Column: 7}
	source := &sdk.Error{Code: sdk.ErrorInternal, Message: private.Error(), Field: public.Field, ExpectedSourceRevision: 7, CurrentSourceRevision: 8, Diagnostics: []sdk.Diagnostic{public}, Cause: private}
	mapped := mapError(source)
	var response *xresponse.Error
	if !errors.As(mapped, &response) || response.Code != 500 || !errors.Is(mapped, private) {
		t.Fatal("internal classification or cause changed")
	}
	payload := response.Payload.(map[string]any)
	ds, ok := payload["diagnostics"].([]sdk.Diagnostic)
	if !ok || len(ds) != 1 || ds[0] != public || payload["message"] != public.Message || payload["field"] != public.Field || payload["expectedSourceRevision"] != int64(7) || payload["currentSourceRevision"] != int64(8) {
		t.Fatalf("projected feedback lost: %+v", payload)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private_column") || strings.Contains(mapped.Error(), "private-password") {
		t.Fatal("projected internal error exposed private SDK message")
	}
	ds[0].Hint = "caller mutation"
	if source.Diagnostics[0] != public || source.Message != private.Error() {
		t.Fatal("mapError mutated trusted source error")
	}
}

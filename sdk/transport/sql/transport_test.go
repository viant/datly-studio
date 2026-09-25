package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/transcribe"
	_ "modernc.org/sqlite"
)

func TestTransportUsesCanonicalConnectorAndReportTables(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "alice"})
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Now: func() time.Time { return time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) }, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}

	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{
		Name: "main", Driver: "sqlite", DSNTemplate: "file:main", OwnerID: "alice",
		Options: json.RawMessage(`{"busy_timeout_ms":1000}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if connector.Name != "main" || connector.Status != "draft" || connector.DSNTemplate != "" || !connector.DSNConfigured {
		t.Fatalf("connector = %+v", connector)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}

	report, err := client.Reports().Create(ctx, sdk.CreateReportInput{
		ID: "r-main", Slug: "main", Title: "Main", OwnerID: "alice",
		DefaultConnectorName: "main", ComponentScope: "reports", ComponentName: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ID != "r-main" || report.DefaultConnectorName != "main" {
		t.Fatalf("report = %+v", report)
	}

	version, err := client.Versions().Create(ctx, report.ID, sdk.CreateVersionInput{
		AuthoringMode: "dql", AuthoredDQL: "SELECT 1", CreatedBy: "alice",
		ComponentSpec: json.RawMessage(`{"views":[]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if version.ReportID != report.ID || version.VersionNo != 1 || version.AuthoringMode != "dql" {
		t.Fatalf("version = %+v", version)
	}
	edit, err := client.Versions().Apply(ctx, report.ID, 1, sdk.EditCommand{
		Kind: "set_dql", ExpectedSourceRevision: version.SourceRevision,
		Payload: json.RawMessage(`{"authoredDql":"SELECT 2"}`),
	})
	if err != nil || edit.Version.GeneratedDQL != "SELECT 2" || edit.Version.SourceRevision != 2 {
		t.Fatalf("edit = %+v, %v", edit, err)
	}

	validation, err := client.Versions().Validate(ctx, report.ID, 1)
	if err != nil || !validation.Valid || validation.Version.CompileStatus != "valid" {
		t.Fatalf("validation = %+v, %v", validation, err)
	}
	transport.Validator = validationStub{err: errors.New("required connector is unavailable")}
	failedValidation, err := client.Versions().Validate(ctx, report.ID, 1)
	if err != nil || failedValidation.Valid || failedValidation.Version.CompileStatus != "invalid" || len(failedValidation.Diagnostics) != 1 || failedValidation.Diagnostics[0].Code != "runtime_contract" {
		t.Fatalf("failed validation = %+v, %v", failedValidation, err)
	}
	transport.Validator = validationStub{err: fmt.Errorf("compile: %w", &transcribe.CompileError{Diagnostics: []*transcribe.Diagnostic{{Code: "DQL-TYPE", Severity: transcribe.SeverityError, Message: "unknown linked type", Hint: "Import the package that owns the type.", Span: transcribe.Span{Start: transcribe.Position{Line: 12, Char: 7}}}}})}
	structuredValidation, err := client.Versions().Validate(ctx, report.ID, 1)
	if err != nil || structuredValidation.Valid || len(structuredValidation.Diagnostics) != 1 || structuredValidation.Diagnostics[0].Code != "DQL-TYPE" || structuredValidation.Diagnostics[0].Hint == "" || structuredValidation.Diagnostics[0].Line != 12 || structuredValidation.Diagnostics[0].Column != 7 {
		t.Fatalf("structured validation = %+v, %v", structuredValidation, err)
	}
	transport.Validator = nil
	validation, err = client.Versions().Validate(ctx, report.ID, 1)
	if err != nil || !validation.Valid {
		t.Fatalf("revalidation = %+v, %v", validation, err)
	}
	export, err := client.Versions().ExportDQL(ctx, report.ID, 1)
	if err != nil || export.DQL != "SELECT 2" || !export.Complete {
		t.Fatalf("export = %+v, %v", export, err)
	}
	descriptor, err := client.Versions().Descriptor(ctx, report.ID, 1)
	if err != nil || string(descriptor.Component) != `{"views":[]}` {
		t.Fatalf("descriptor = %+v, %v", descriptor, err)
	}
	beforePublish, err := client.Reports().Get(ctx, report.ID)
	if err != nil {
		t.Fatal(err)
	}

	var activatedGeneration int64
	transport.Activator = RuntimeActivatorFunc(func(_ context.Context, generation int64) error { activatedGeneration = generation; return nil })
	publication, err := client.Publications().Publish(ctx, report.ID, 1, sdk.PublishInput{RequestedBy: "alice", Reason: "initial reader release"})
	if err != nil {
		t.Fatal(err)
	}
	if publication.ReportID != report.ID || publication.ActiveVersionNo != 1 || publication.Status != "active" || publication.ActiveGeneration == nil {
		t.Fatalf("publication = %+v", publication)
	}
	afterPublish, err := client.Reports().Get(ctx, report.ID)
	if err != nil || afterPublish.Status != "active" || afterPublish.ETag != beforePublish.ETag+1 {
		t.Fatalf("report after publish = %+v, before = %+v, err = %v", afterPublish, beforePublish, err)
	}
	if activatedGeneration != *publication.ActiveGeneration {
		t.Fatalf("activated generation=%d publication=%+v", activatedGeneration, publication)
	}
	loadedPublication, err := client.Publications().Get(ctx, report.ID)
	if err != nil || loadedPublication.Status != "active" || loadedPublication.ActiveGeneration == nil {
		t.Fatalf("loaded publication=%+v err=%v", loadedPublication, err)
	}
	if _, err = db.Exec(`INSERT INTO report_mcp_exposures(report_id,version_no,exposure_id,route_id,route_method,route_path,kind,name,description,mime_type,enabled,ordinal) VALUES(?,1,'tool','route','GET','/main','tool','alice.main.read','Read main','application/json',TRUE,0)`, report.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal) VALUES(?,1,'docs','alice.docs','guide','skill://alice-guide/',0)`, report.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_skill_roots(report_id,version_no,skill_id,folder_id,skill_root,ordinal) VALUES(?,1,'guide','docs','.',0)`, report.ID); err != nil {
		t.Fatal(err)
	}
	transport.RuntimeProbe = RuntimeHostProbeFunc(func(context.Context) (*sdk.RuntimeHost, error) {
		return &sdk.RuntimeHost{Status: "ready", Revision: *publication.ActiveGeneration}, nil
	})
	runtimeStatus, err := client.Runtime().Status(ctx)
	if err != nil || runtimeStatus.Status != "active" || runtimeStatus.ActiveGeneration != *publication.ActiveGeneration || runtimeStatus.Host == nil || runtimeStatus.Host.Status != "ready" || runtimeStatus.Host.Revision != *publication.ActiveGeneration || len(runtimeStatus.Readers) != 1 || len(runtimeStatus.Readers[0].MCPExposures) != 1 || len(runtimeStatus.Readers[0].MCPResources) != 1 || runtimeStatus.Readers[0].MCPResources[0].URIPrefix != "skill://alice-guide/" || len(runtimeStatus.Readers[0].Skills) != 1 || runtimeStatus.Readers[0].Skills[0].SkillRoot != "." {
		t.Fatalf("runtime status = %+v, %v", runtimeStatus, err)
	}
	aliceStatus, err := client.Runtime().Status(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"}))
	if err != nil || len(aliceStatus.Readers) != 1 || aliceStatus.Readers[0].Title != "Main" {
		t.Fatalf("alice runtime status = %+v, %v", aliceStatus, err)
	}
	bobStatus, err := client.Runtime().Status(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "bob"}))
	if err != nil || len(bobStatus.Readers) != 0 {
		t.Fatalf("bob runtime status = %+v, %v", bobStatus, err)
	}
	other, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "other", Driver: "sqlite", DSNTemplate: "file:other", OwnerID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, other.Name); err != nil {
		t.Fatal(err)
	}
	unversioned, err := client.Reports().Create(ctx, sdk.CreateReportInput{Slug: "unversioned", Title: "Unversioned", OwnerID: "alice", DefaultConnectorName: "main"})
	if err != nil {
		t.Fatal(err)
	}
	otherName := other.Name
	unversioned, err = client.Reports().Update(ctx, unversioned.ID, sdk.UpdateReportInput{DefaultConnectorName: &otherName, ETag: unversioned.ETag})
	if err != nil || unversioned.DefaultConnectorName != other.Name {
		t.Fatalf("unversioned connector=%q err=%v", unversioned.DefaultConnectorName, err)
	}
	if _, err = db.Exec(`UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id=?`, unversioned.ID); err != nil {
		t.Fatal(err)
	}
	currentReport, err := client.Reports().Get(ctx, report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Reports().Update(ctx, report.ID, sdk.UpdateReportInput{DefaultConnectorName: &otherName, ETag: currentReport.ETag}); err == nil {
		t.Fatal("versioned report connector was repointed")
	} else {
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
			t.Fatalf("connector repoint error=%v", err)
		}
	}
	unchanged, err := client.Reports().Get(ctx, report.ID)
	if err != nil || unchanged.DefaultConnectorName != "main" {
		t.Fatalf("report connector=%q err=%v", unchanged.DefaultConnectorName, err)
	}
	versionTwo, err := client.Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", CreatedBy: "alice", AuthoredDQL: "SELECT 3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().Validate(ctx, report.ID, versionTwo.VersionNo); err != nil {
		t.Fatal(err)
	}
	transport.Activator = RuntimeActivatorFunc(func(_ context.Context, generation int64) error {
		var activeVersion, desiredVersion int
		var activeGeneration int64
		var status string
		if queryErr := db.QueryRow(`SELECT active_version_no,desired_version_no,active_generation,publication_status FROM report_publications WHERE report_id=?`, report.ID).Scan(&activeVersion, &desiredVersion, &activeGeneration, &status); queryErr != nil {
			return queryErr
		}
		if activeVersion != 1 || desiredVersion != versionTwo.VersionNo || activeGeneration != *publication.ActiveGeneration || status != "pending" {
			return fmt.Errorf("staged publication replaced active state: active=%d desired=%d generation=%d status=%s", activeVersion, desiredVersion, activeGeneration, status)
		}
		activatedGeneration = generation
		return nil
	})
	if _, err = client.Publications().Rollback(ctx, report.ID, versionTwo.VersionNo, sdk.PublishInput{RequestedBy: "alice", Reason: "restore selected release"}); err != nil {
		t.Fatal(err)
	}
	var versionOneState, versionTwoState string
	if err = db.QueryRow(`SELECT state FROM report_versions WHERE report_id=? AND version_no=1`, report.ID).Scan(&versionOneState); err != nil || versionOneState != "superseded" {
		t.Fatalf("version one state=%q err=%v", versionOneState, err)
	}
	if err = db.QueryRow(`SELECT state FROM report_versions WHERE report_id=? AND version_no=?`, report.ID, versionTwo.VersionNo).Scan(&versionTwoState); err != nil || versionTwoState != "published" {
		t.Fatalf("version two state=%q err=%v", versionTwoState, err)
	}
	transport.Activator = RuntimeActivatorFunc(func(_ context.Context, generation int64) error {
		var activeGeneration int64
		var status string
		if queryErr := db.QueryRow(`SELECT active_generation,publication_status FROM report_publications WHERE report_id=?`, report.ID).Scan(&activeGeneration, &status); queryErr != nil {
			return queryErr
		}
		if activeGeneration != activatedGeneration || status != "unpublishing" {
			return fmt.Errorf("staged unpublish removed active state: generation=%d status=%s", activeGeneration, status)
		}
		activatedGeneration = generation
		return nil
	})
	unpublished, err := client.Publications().Unpublish(ctx, report.ID, sdk.UnpublishInput{RequestedBy: "alice", Reason: "retire reader"})
	if err != nil || unpublished.Status != "unpublished" {
		t.Fatalf("unpublish = %+v, %v", unpublished, err)
	}
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return errors.New("reload failed password=topsecret") })
	if _, err = client.Publications().Publish(ctx, report.ID, 1, sdk.PublishInput{RequestedBy: "alice", Reason: "password=also-secret"}); err == nil || !strings.Contains(err.Error(), "reload failed") {
		t.Fatalf("failed publication error=%v", err)
	}
	failedEvents, err := client.Publications().ListEvents(ctx, report.ID, sdk.ListPublicationEventsInput{Status: "failed"})
	if err != nil || len(failedEvents.Items) != 1 || failedEvents.Items[0].Operation != "publish" || failedEvents.Items[0].RequestedBy != "alice" || failedEvents.Items[0].GenerationNo == nil || strings.Contains(failedEvents.Items[0].FailureMessage, "topsecret") || strings.Contains(failedEvents.Items[0].Reason, "also-secret") {
		t.Fatalf("failed publication events=%+v err=%v", failedEvents, err)
	}
	for _, assertion := range []struct {
		operation string
		reason    string
	}{
		{operation: "publish", reason: "initial reader release"},
		{operation: "rollback", reason: "restore selected release"},
		{operation: "unpublish", reason: "retire reader"},
	} {
		items, listErr := client.Publications().ListEvents(ctx, report.ID, sdk.ListPublicationEventsInput{Operation: assertion.operation, Status: "succeeded"})
		if listErr != nil || len(items.Items) != 1 || items.Items[0].Reason != assertion.reason || items.Items[0].OccurredAt.IsZero() {
			t.Fatalf("%s events=%+v err=%v", assertion.operation, items, listErr)
		}
	}
	var generationStatus, publicationStatus string
	if err = db.QueryRow(`SELECT status FROM runtime_generations ORDER BY generation_no DESC LIMIT 1`).Scan(&generationStatus); err != nil || generationStatus != "failed" {
		t.Fatalf("failed generation status=%q err=%v", generationStatus, err)
	}
	if err = db.QueryRow(`SELECT publication_status FROM report_publications WHERE report_id=?`, report.ID).Scan(&publicationStatus); err != nil || publicationStatus != "failed" {
		t.Fatalf("restored publication status=%q err=%v", publicationStatus, err)
	}
	var failedGeneration int64
	if err = db.QueryRow(`SELECT generation_no FROM runtime_generations ORDER BY generation_no DESC LIMIT 1`).Scan(&failedGeneration); err != nil {
		t.Fatal(err)
	}
	if err = transport.restoreFailedPublication(ctx, report.ID, failedGeneration, publicationState{}, false, errors.New("retry")); err == nil {
		t.Fatal("already failed generation accepted a second restore")
	}
	if err = db.QueryRow(`SELECT publication_status FROM report_publications WHERE report_id=?`, report.ID).Scan(&publicationStatus); err != nil || publicationStatus != "failed" {
		t.Fatalf("second restore changed publication status=%q err=%v", publicationStatus, err)
	}
	transport.Preview = previewStub{}
	preview, err := client.Preview().Execute(ctx, report.ID, 1, sdk.PreviewInput{Input: json.RawMessage(`{"limit":5}`), Limit: 5})
	if err != nil || string(preview.Data) != `{"rows":[{"value":2}]}` {
		t.Fatalf("preview = %+v, %v", preview, err)
	}

	page, err := client.Reports().List(ctx, sdk.ListReportsInput{OwnerID: "alice", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "r-main" {
		t.Fatalf("reports = %+v", page.Items)
	}

	status, err := client.Connectors().Disable(ctx, "main", connector.ETag)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "disabled" || status.ETag != 2 {
		t.Fatalf("disabled connector = %+v", status)
	}
	if err := client.Connectors().Delete(ctx, "main", status.ETag); err == nil {
		t.Fatal("delete referenced connector unexpectedly succeeded")
	}
}

type previewStub struct{}

type validationStub struct{ err error }

type warmupStub struct {
	wait   <-chan struct{}
	result *sdk.WarmupResult
	err    error
}

func (v validationStub) Validate(context.Context, string, int) error { return v.err }

func (previewStub) Execute(context.Context, string, int, sdk.PreviewInput) (*sdk.PreviewResult, error) {
	return &sdk.PreviewResult{Data: json.RawMessage(`{"rows":[{"value":2}]}`)}, nil
}

func (w warmupStub) Warmup(ctx context.Context, _ string, _ int) (*sdk.WarmupResult, error) {
	if w.wait != nil {
		select {
		case <-w.wait:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return w.result, w.err
}

func TestWarmupRunsPersistAndDeduplicateServerOwnedEvidence(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,?,?);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','draft','main','example.com/reader','reader',1,?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',1,'validated','dql','SELECT 1','{}','1','hash','{}','valid','v1','v1',3,'owner',?);`, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}, Warmup: warmupStub{wait: release, result: &sdk.WarmupResult{Status: "completed", PlannedCases: 4, CompletedCases: 4, Entries: 7, Duration: time.Millisecond, Target: sdk.WarmupTarget{View: "reader", CacheName: "reader-cache", CacheProvider: "afs", ConnectorName: "main", IndexColumn: "ID"}}}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.Versions().Warmup(ctx, "reader", 1)
	if err != nil || first.Status != "accepted" || first.RunID == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if first.CreatedAt == nil || first.UpdatedAt == nil || first.CreatedBy == nil || *first.CreatedBy != "owner" ||
		first.UpdatedBy == nil || *first.UpdatedBy != "owner" {
		t.Fatalf("warmup creation audit=%+v", first)
	}
	duplicate, err := client.Versions().Warmup(ctx, "reader", 1)
	if err != nil || duplicate.RunID != first.RunID {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	var completed *sdk.WarmupRun
	for time.Now().Before(deadline) {
		completed, err = client.Versions().WarmupRun(ctx, "reader", first.RunID)
		if err == nil && completed.Status == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || completed == nil || completed.Status != "completed" || completed.PlannedCases != 4 || completed.CompletedCases != 4 || completed.Entries != 7 || completed.Target.CacheName != "reader-cache" {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}
	if completed.UpdatedAt == nil || !completed.UpdatedAt.After(*first.UpdatedAt) || completed.UpdatedBy == nil || *completed.UpdatedBy != "owner" {
		t.Fatalf("warmup completion audit=%+v", completed)
	}
	page, err := client.Versions().ListWarmupRuns(ctx, "reader", 1, sdk.ListWarmupRunsInput{Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].RunID != first.RunID {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	transport.Warmup = warmupStub{result: &sdk.WarmupResult{Status: "running", PlannedCases: 4, CompletedCases: 1, Entries: 2, Target: sdk.WarmupTarget{View: "reader", CacheName: "reader-cache"}}, err: &sdk.Error{Code: sdk.ErrorInternal, Message: "bounded warmup case failed"}}
	partialStart, err := client.Versions().Warmup(ctx, "reader", 1)
	if err != nil || partialStart.RunID == first.RunID {
		t.Fatalf("partial start=%+v err=%v", partialStart, err)
	}
	var partial *sdk.WarmupRun
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		partial, err = client.Versions().WarmupRun(ctx, "reader", partialStart.RunID)
		if err == nil && partial.Status == "partial" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || partial == nil || partial.Status != "partial" || partial.CompletedCases != 1 || len(partial.Diagnostics) != 1 || partial.Diagnostics[0].Message != "bounded warmup case failed" {
		t.Fatalf("partial=%+v err=%v", partial, err)
	}
}

func TestTransportReturnsTypedNotFound(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Reports().Get(context.Background(), "missing")
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
		t.Fatalf("error = %v, want typed not_found", err)
	}
}

type allowAuthorizer struct{}

func (allowAuthorizer) Authorize(context.Context, AuthorizationRequest) error { return nil }

func TestTransportRejectsMissingAuthorizer(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Connectors().List(context.Background(), sdk.ListConnectorsInput{})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorForbidden {
		t.Fatalf("error = %v, want authorization forbidden", err)
	}
}

func TestRuntimeCatalogProjectsReaderCubeAndComposeComponents(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite", OwnerID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(ctx, sdk.CreateReportInput{Slug: "spend", Title: "Spend", OwnerID: "owner", DefaultConnectorName: "main"})
	if err != nil {
		t.Fatal(err)
	}
	dql := `#package('example.com/spend/reader')
#setting($_ = $connector('main'))
#setting($_ = $route('/spend','GET'))
#setting($_ = $cube())
#setting($_ = $cubeCompose(true,true,4,50,12000))
#setting($_ = $mcp('owner.spend.read','Read spend'))
#define($_ = $Rows<[]*Spend>(output/view))
SELECT spend.*, groupable(spend), tag(spend.account_id,'groupable:"true"'), CAST(spend.total AS float64)
FROM (SELECT 1 AS account_id, SUM(2) AS total GROUP BY 1) spend`
	version, err := client.Versions().Create(ctx, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: dql, CreatedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().Validate(ctx, report.ID, version.VersionNo); err != nil {
		t.Fatal(err)
	}
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return nil })
	publication, err := client.Publications().Publish(ctx, report.ID, version.VersionNo, sdk.PublishInput{RequestedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_mcp_exposures(report_id,version_no,exposure_id,route_id,route_method,route_path,kind,name,description,mime_type,enabled,ordinal) VALUES(?,?,'reader','route','GET','/spend','tool','owner.spend.read','Read spend','application/json',TRUE,0)`, report.ID, version.VersionNo); err != nil {
		t.Fatal(err)
	}
	status, err := client.Runtime().Status(ctx)
	if err != nil || len(status.Readers) != 1 || status.ActiveGeneration != *publication.ActiveGeneration {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	tools := status.Readers[0].MCPExposures
	if len(tools) != 3 || tools[0].Name != "owner.spend.read" || tools[1].Name != "owner.spend.readCube" || tools[1].Path != "/spend/cube" || !tools[1].Enabled || tools[2].Name != "owner.spend.readCubeCompose" || tools[2].Path != "/spend/cube/compose" || !tools[2].Enabled {
		t.Fatalf("tools=%+v", tools)
	}
}

func TestTransportConnectorCatalogFiltersSearchesAndPaginates(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for _, values := range [][]any{
		{"alpha", "sqlite", "Primary catalog", "catalog-owner", "active", now.Add(2 * time.Minute)},
		{"beta", "mysql", "Warehouse reporting", "catalog-owner", "draft", now.Add(time.Minute)},
		{"gamma", "sqlite", "Archive", "another-owner", "disabled", now},
	} {
		if _, err = db.Exec(`INSERT INTO connectors(name,driver,description,owner_id,status,etag,created_at,updated_at) VALUES(?,?,?,?,?,1,?,?)`, values[0], values[1], values[2], values[3], values[4], values[5], values[5]); err != nil {
			t.Fatal(err)
		}
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}

	page, err := client.Connectors().List(ctx, sdk.ListConnectorsInput{Query: "warehouse", Status: "draft", Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "beta" {
		t.Fatalf("description/status search = %+v, err=%v", page, err)
	}
	page, err = client.Connectors().List(ctx, sdk.ListConnectorsInput{Query: "sqlite", Limit: 1, Offset: 1})
	if err != nil || page.Limit != 1 || page.Offset != 1 || len(page.Items) != 1 || page.Items[0].Name != "gamma" {
		t.Fatalf("driver pagination = %+v, err=%v", page, err)
	}
	page, err = client.Connectors().List(ctx, sdk.ListConnectorsInput{Query: "another-owner", Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "gamma" {
		t.Fatalf("owner search = %+v, err=%v", page, err)
	}
}

func TestTransportDerivesConnectorOwnerFromPrincipal(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner-a"})
	connector, err := client.Connectors().Create(principal, sdk.CreateConnectorInput{Name: "owned", Driver: "sqlite"})
	if err != nil || connector.OwnerID != "owner-a" {
		t.Fatalf("connector=%+v err=%v", connector, err)
	}
	if _, err = client.Connectors().Activate(principal, connector.Name, connector.ETag); err == nil {
		t.Fatal("activation before passing a probe unexpectedly succeeded")
	}
}

func TestTransportRequiresNewProbeAfterConnectionConfigurationChange(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner-a"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite", DSNTemplate: "file:one.db"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active',last_test_status='passed' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	connector, err = client.Connectors().Get(owner, "main")
	if err != nil {
		t.Fatal(err)
	}
	description := "catalog source"
	updated, err := client.Connectors().Update(owner, connector.Name, sdk.UpdateConnectorInput{Description: &description, ETag: connector.ETag})
	if err != nil || updated.Status != "active" || updated.LastTestStatus != "passed" {
		t.Fatalf("description update=%+v err=%v", updated, err)
	}
	dsn := "file:two.db"
	updated, err = client.Connectors().Update(owner, updated.Name, sdk.UpdateConnectorInput{DSNTemplate: &dsn, ETag: updated.ETag})
	if err != nil || updated.Status != "draft" || updated.LastTestStatus != "" || updated.LastTestedAt != nil {
		t.Fatalf("connection update=%+v err=%v", updated, err)
	}
	_, err = client.Connectors().Update(owner, updated.Name, sdk.UpdateConnectorInput{Description: &description, ETag: connector.ETag})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("stale connector configuration update error=%v", err)
	}
}

func TestTransportDerivesReportIdentityAndOwnerFromPrincipal(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner-a"})
	connector, err := client.Connectors().Create(principal, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	input := sdk.CreateReportInput{Slug: "vendor-catalog", Title: "Vendor Catalog", DefaultConnectorName: connector.Name}
	if _, err = client.Reports().Create(principal, input); err == nil {
		t.Fatal("report creation with an inactive connector unexpectedly succeeded")
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(principal, input)
	if err != nil {
		t.Fatal(err)
	}
	if report.ID == "" || report.Namespace != "general" || report.OwnerID != "owner-a" || report.OwnerPackage == "" || !strings.Contains(report.ComponentScope, "/dynamic/"+report.OwnerPackage+"/"+report.ID) || report.ComponentName != "reader" {
		t.Fatalf("report=%+v", report)
	}
	_, err = client.Reports().Create(principal, input)
	var duplicate *sdk.Error
	if !errors.As(err, &duplicate) || duplicate.Code != sdk.ErrorConflict {
		t.Fatalf("duplicate report slug error=%v", err)
	}
	namespace := "inventory.forecasting"
	if _, err = client.Namespaces().Create(principal, sdk.CreateNamespaceInput{Name: namespace, Title: "Inventory Forecasting"}); err != nil {
		t.Fatal(err)
	}
	updated, err := client.Reports().Update(principal, report.ID, sdk.UpdateReportInput{Namespace: &namespace, ETag: report.ETag})
	if err != nil || updated.Namespace != namespace {
		t.Fatalf("namespace update=%+v err=%v", updated, err)
	}
	_, err = client.Reports().Update(principal, report.ID, sdk.UpdateReportInput{Title: &namespace, ETag: report.ETag})
	var stale *sdk.Error
	if !errors.As(err, &stale) || stale.Code != sdk.ErrorConflict {
		t.Fatalf("stale report update error=%v", err)
	}
}

func TestTransportReusesDatlyReaderBuilderForVersionedDQL(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner-a"})
	connector, err := client.Connectors().Create(principal, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Connectors().Create(principal, sdk.CreateConnectorInput{Name: "lookup", Driver: "sqlite"}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='lookup'`); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(principal, sdk.CreateReportInput{Slug: "reader-builder", Title: "Reader Builder", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(principal, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", CreatedBy: "owner-a", AuthoredDQL: `#package('example.com/legacy/reader')
#setting($_ = $route('/records','GET'))
SELECT records.* FROM (SELECT 1 AS id) records`})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := client.Versions().Inspect(principal, report.ID, version.VersionNo)
	if err != nil || len(inspection.Structure) == 0 {
		t.Fatalf("inspection=%+v err=%v", inspection, err)
	}
	var initialStructure readerbuilder.Structure
	if err = json.Unmarshal(inspection.Structure, &initialStructure); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(initialStructure.AvailableConnectors) != "[lookup main]" {
		t.Fatalf("available connectors=%v", initialStructure.AvailableConnectors)
	}
	packageOperation, _ := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationSetPackage, Package: &readerbuilder.PackageMutation{Path: "browser.supplied/is/ignored"}})
	packageResult, err := client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: version.SourceRevision, Operation: packageOperation})
	if err != nil || !packageResult.Applied || !strings.Contains(packageResult.Inspection.DQL, `#package("`+report.ComponentScope+`/reader")`) {
		t.Fatalf("package result=%+v err=%v", packageResult, err)
	}
	version = packageResult.Inspection.Version
	connectorOperation, _ := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationSetSetting, Setting: &readerbuilder.SettingMutation{Name: "connector", Args: []string{"'lookup'"}}})
	connectorResult, err := client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: version.SourceRevision, Operation: connectorOperation})
	if err != nil || !connectorResult.Applied || !strings.Contains(connectorResult.Inspection.DQL, `$connector('lookup')`) {
		t.Fatalf("connector result=%+v err=%v", connectorResult, err)
	}
	updatedReport, err := client.Reports().Get(principal, report.ID)
	if err != nil || updatedReport.DefaultConnectorName != "lookup" || updatedReport.ETag != report.ETag+2 {
		t.Fatalf("connector report=%+v err=%v", updatedReport, err)
	}
	version = connectorResult.Inspection.Version
	operation, err := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationAddField, Field: &readerbuilder.Field{Name: "Limit", Type: "int", SourceKind: "query", SourceName: "limit"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: version.SourceRevision, Operation: operation})
	if err != nil || !result.Applied || result.Inspection.Version.SourceRevision != version.SourceRevision+1 || !strings.Contains(result.Inspection.DQL, "$Limit<int>") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	afterField, err := client.Reports().Get(principal, report.ID)
	if err != nil || afterField.ETag != updatedReport.ETag {
		t.Fatalf("same-connector Reader Builder edit changed report etag: %+v err=%v", afterField, err)
	}
	operation, err = json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationAddView, View: &readerbuilder.ViewMutation{
		Name: "totals", Kind: "derived", Parent: "records", SQL: "SELECT COUNT(*) AS count FROM ($View.Records.NonWindowSQL) parent",
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: result.Inspection.Version.SourceRevision, Operation: operation})
	if err != nil || !result.Applied || !strings.Contains(result.Inspection.DQL, "output/derived") {
		t.Fatalf("derived result=%+v err=%v", result, err)
	}
	var structure readerbuilder.Structure
	if err = json.Unmarshal(result.Inspection.Structure, &structure); err != nil {
		t.Fatal(err)
	}
	if structure.Component == nil || structure.Component.RootView == nil || len(structure.Component.RootView.Relations) != 1 || structure.Component.RootView.Relations[0].Kind != "derived" {
		t.Fatalf("derived structure=%+v", structure.Component)
	}

	wrongExposure, _ := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationSetSetting, Setting: &readerbuilder.SettingMutation{Name: "mcp", Args: []string{"'records.read'"}}})
	if _, err = client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: result.Inspection.Version.SourceRevision, Operation: wrongExposure}); err == nil || !strings.Contains(err.Error(), "owner prefix") {
		t.Fatalf("unscoped MCP tool error=%v", err)
	}
	toolName := report.OwnerPackage + ".records.read"
	exposure, _ := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationSetSetting, Setting: &readerbuilder.SettingMutation{Name: "mcp", Args: []string{"'" + toolName + "'"}}})
	result, err = client.Versions().ApplyReaderCommand(principal, report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: result.Inspection.Version.SourceRevision, Operation: exposure})
	if err != nil || !result.Applied {
		t.Fatalf("owner-scoped MCP exposure=%+v err=%v", result, err)
	}
	if _, err = db.Exec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'reader-builder:1','active',1,'{}','owner-a',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES(?,1,1,1,1,'active','reader-builder:1','hash','owner-a',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, report.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().Create(principal, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", CreatedBy: "owner-a", AuthoredDQL: `#package('example.com/latest')
#setting($_ = $route('/latest','GET'))
SELECT records.* FROM (SELECT 1 AS id) records`}); err != nil {
		t.Fatal(err)
	}
	second, err := client.Reports().Create(principal, sdk.CreateReportInput{Slug: "reader-builder-two", Title: "Reader Builder Two", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	secondVersion, err := client.Versions().Create(principal, second.ID, sdk.CreateVersionInput{AuthoringMode: "dql", CreatedBy: "owner-a", AuthoredDQL: `#package('example.com/second')
#setting($_ = $route('/second','GET'))
SELECT records.* FROM (SELECT 1 AS id) records`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().ApplyReaderCommand(principal, second.ID, secondVersion.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: secondVersion.SourceRevision, Operation: exposure}); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("duplicate MCP tool error=%v", err)
	}
}

func TestTransportScopesCatalogReadsToPrincipalOwnerOrACL(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	transport := &Transport{DB: db, Now: func() time.Time { return now }, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	for _, connector := range []sdk.CreateConnectorInput{
		{Name: "alice-db", Driver: "sqlite", OwnerID: "alice"},
		{Name: "shared-db", Driver: "sqlite", OwnerID: "bob"},
		{Name: "private-db", Driver: "sqlite", OwnerID: "bob"},
	} {
		if _, err = client.Connectors().Create(ctx, connector); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active'`); err != nil {
		t.Fatal(err)
	}
	for _, report := range []sdk.CreateReportInput{
		{ID: "alice-report", Slug: "alice", Title: "Alice", OwnerID: "alice", DefaultConnectorName: "alice-db", ComponentScope: "reports", ComponentName: "alice"},
		{ID: "shared-report", Slug: "shared", Title: "Shared", OwnerID: "bob", DefaultConnectorName: "shared-db", ComponentScope: "reports", ComponentName: "shared"},
		{ID: "private-report", Slug: "private", Title: "Private", OwnerID: "bob", DefaultConnectorName: "private-db", ComponentScope: "reports", ComponentName: "private"},
	} {
		if _, err = client.Reports().Create(ctx, report); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO report_acl(report_id,subject_type,subject_id,can_view) VALUES ('shared-report','user','alice',TRUE)`); err != nil {
		t.Fatal(err)
	}
	scoped := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"})
	reports, err := client.Reports().List(scoped, sdk.ListReportsInput{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got := reportIDs(reports.Items); fmt.Sprint(got) != "[alice-report shared-report]" {
		t.Fatalf("reports=%v", got)
	}
	filtered, err := client.Reports().List(scoped, sdk.ListReportsInput{Query: "SHARED", OwnerID: "bob", ConnectorName: "shared-db", Limit: 1})
	if err != nil || fmt.Sprint(reportIDs(filtered.Items)) != "[shared-report]" {
		t.Fatalf("filtered report page=%+v err=%v", filtered, err)
	}
	second, err := client.Reports().List(scoped, sdk.ListReportsInput{Limit: 1, Offset: 1})
	if err != nil || fmt.Sprint(reportIDs(second.Items)) != "[shared-report]" {
		t.Fatalf("second report page=%+v err=%v", second, err)
	}
	all, err := client.Reports().List(ctx, sdk.ListReportsInput{Limit: 10})
	if err != nil || len(all.Items) != 3 {
		t.Fatalf("trusted in-process catalog=%+v err=%v", all, err)
	}
	connectors, err := client.Connectors().List(scoped, sdk.ListConnectorsInput{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got := connectorNames(connectors.Items); fmt.Sprint(got) != "[alice-db shared-db]" {
		t.Fatalf("connectors=%v", got)
	}
	if _, err = client.Reports().Get(scoped, "private-report"); !isNotFound(err) {
		t.Fatalf("private report error=%v", err)
	}
	if _, err = client.Connectors().Get(scoped, "private-db"); !isNotFound(err) {
		t.Fatalf("private connector error=%v", err)
	}
}

func TestTransportStoresVersionedResourcesAndSkillsThroughSDK(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(principal, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(principal, sdk.CreateReportInput{Slug: "resources", Title: "Resources", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(principal, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", CreatedBy: "owner", AuthoredDQL: `#package('example.com/resources')
#setting($_ = $route('/resources','GET'))
SELECT 1`})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Resources().UpsertFile(principal, sdk.ResourceFile{ReportID: report.ID, VersionNo: version.VersionNo,
		Namespace: report.OwnerPackage + ".docs", ResourcePath: "guide/SKILL.md", Content: "missing revision"})
	var missingRevision *sdk.Error
	if !errors.As(err, &missingRevision) || missingRevision.Code != sdk.ErrorInvalidArgument {
		t.Fatalf("resource mutation without revision error=%v", err)
	}
	if _, err := db.Exec(`UPDATE report_versions SET state='validated',compile_status='valid',validated_at=CURRENT_TIMESTAMP WHERE report_id=? AND version_no=?`, report.ID, version.VersionNo); err != nil {
		t.Fatal(err)
	}
	file, err := client.Resources().UpsertFile(principal, sdk.ResourceFile{ReportID: report.ID, VersionNo: version.VersionNo, Namespace: report.OwnerPackage + ".docs", ResourcePath: "guide/SKILL.md", Content: "---\nname: guide\ndescription: Test guide\n---\nUse the guide.", ExpectedSourceRevision: version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Files) != 1 || file.Files[0].ContentSHA256 == "" || file.Version.SourceRevision != version.SourceRevision+1 {
		t.Fatalf("file snapshot=%+v", file)
	}
	var touchedState, touchedCompile, touchedDiagnostics string
	var touchedValidation sql.NullTime
	if err := db.QueryRow(`SELECT state,compile_status,compile_diagnostics_json,validated_at FROM report_versions WHERE report_id=? AND version_no=?`, report.ID, version.VersionNo).
		Scan(&touchedState, &touchedCompile, &touchedDiagnostics, &touchedValidation); err != nil ||
		touchedState != "draft" || touchedCompile != "pending" || touchedDiagnostics != "[]" || touchedValidation.Valid {
		t.Fatalf("touched version state=%q compile=%q diagnostics=%q validated=%v err=%v", touchedState, touchedCompile, touchedDiagnostics, touchedValidation, err)
	}
	folder, err := client.Resources().UpsertFolder(principal, sdk.ResourceFolder{ReportID: report.ID, VersionNo: version.VersionNo, Namespace: report.OwnerPackage + ".docs", RootPath: "guide", URIPrefix: "skill://owner-guide/", ExpectedSourceRevision: file.Version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	if len(folder.Folders) != 1 {
		t.Fatalf("folder snapshot=%+v", folder)
	}
	snapshot, err := client.Resources().UpsertSkill(principal, sdk.SkillRoot{ReportID: report.ID, VersionNo: version.VersionNo, FolderID: folder.Folders[0].FolderID, SkillRoot: ".", ExpectedSourceRevision: folder.Version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Skills) != 1 || snapshot.Skills[0].SkillRoot != "." || snapshot.Version.CompileStatus != "pending" {
		t.Fatalf("skill snapshot=%+v", snapshot)
	}

	staleRevision := file.Version.SourceRevision
	assertRevisionConflict := func(err error) {
		t.Helper()
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
			t.Fatalf("expected revision conflict, got %v", err)
		}
		if sdkErr.ExpectedSourceRevision != staleRevision || sdkErr.CurrentSourceRevision != snapshot.Version.SourceRevision {
			t.Fatalf("conflict metadata=%+v", sdkErr)
		}
	}
	_, err = client.Resources().UpsertFile(principal, sdk.ResourceFile{ReportID: report.ID, VersionNo: version.VersionNo, ResourceID: file.Files[0].ResourceID, Namespace: report.OwnerPackage + ".docs", ResourcePath: "guide/SKILL.md", Content: "stale", ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	_, err = client.Resources().UpsertFolder(principal, sdk.ResourceFolder{ReportID: report.ID, VersionNo: version.VersionNo, FolderID: folder.Folders[0].FolderID, Namespace: report.OwnerPackage + ".docs", RootPath: "changed", URIPrefix: "skill://changed/", ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	_, err = client.Resources().UpsertSkill(principal, sdk.SkillRoot{ReportID: report.ID, VersionNo: version.VersionNo, SkillID: snapshot.Skills[0].SkillID, FolderID: folder.Folders[0].FolderID, SkillRoot: ".", ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	_, err = client.Resources().DeleteFileWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, ResourceID: file.Files[0].ResourceID, ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	_, err = client.Resources().DeleteFolderWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, FolderID: folder.Folders[0].FolderID, ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	_, err = client.Resources().DeleteSkillWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, SkillID: snapshot.Skills[0].SkillID, ExpectedSourceRevision: staleRevision})
	assertRevisionConflict(err)
	afterConflicts, err := client.Resources().Get(principal, report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if afterConflicts.Version.SourceRevision != snapshot.Version.SourceRevision || afterConflicts.Files[0].Content != file.Files[0].Content || len(afterConflicts.Folders) != 1 || len(afterConflicts.Skills) != 1 {
		t.Fatalf("stale mutations changed resources: %+v", afterConflicts)
	}
	type raceResult struct {
		snapshot *sdk.ResourceSnapshot
		err      error
	}
	start := make(chan struct{})
	results := make(chan raceResult, 2)
	for _, id := range []string{"rf-race-a", "rf-race-b"} {
		go func(resourceID string) {
			<-start
			value, raceErr := client.Resources().UpsertFile(principal, sdk.ResourceFile{
				ReportID: report.ID, VersionNo: version.VersionNo, ResourceID: resourceID,
				Namespace: report.OwnerPackage + ".docs", ResourcePath: "guide/" + resourceID + ".md",
				Content: resourceID, ExpectedSourceRevision: snapshot.Version.SourceRevision,
			})
			results <- raceResult{snapshot: value, err: raceErr}
		}(id)
	}
	close(start)
	successes, conflicts := 0, 0
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			continue
		}
		var sdkErr *sdk.Error
		if errors.As(result.err, &sdkErr) && sdkErr.Code == sdk.ErrorConflict && sdkErr.ExpectedSourceRevision == snapshot.Version.SourceRevision && sdkErr.CurrentSourceRevision == snapshot.Version.SourceRevision+1 {
			conflicts++
			continue
		}
		t.Fatalf("unexpected competing mutation error: %v", result.err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("competing results successes=%d conflicts=%d", successes, conflicts)
	}
	afterRace, err := client.Resources().Get(principal, report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if afterRace.Version.SourceRevision != snapshot.Version.SourceRevision+1 || len(afterRace.Files) != 2 {
		t.Fatalf("competing mutation was not atomic: %+v", afterRace)
	}
	afterSkillDelete, err := client.Resources().DeleteSkillWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, SkillID: snapshot.Skills[0].SkillID, ExpectedSourceRevision: afterRace.Version.SourceRevision})
	if err != nil || len(afterSkillDelete.Skills) != 0 || afterSkillDelete.Version.SourceRevision != afterRace.Version.SourceRevision+1 {
		t.Fatalf("skill delete snapshot=%+v err=%v", afterSkillDelete, err)
	}
	afterFolderDelete, err := client.Resources().DeleteFolderWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, FolderID: folder.Folders[0].FolderID, ExpectedSourceRevision: afterSkillDelete.Version.SourceRevision})
	if err != nil || len(afterFolderDelete.Folders) != 0 || afterFolderDelete.Version.SourceRevision != afterSkillDelete.Version.SourceRevision+1 {
		t.Fatalf("folder delete snapshot=%+v err=%v", afterFolderDelete, err)
	}
	afterFileDelete, err := client.Resources().DeleteFileWithRevision(principal, sdk.ResourceDeleteInput{ReportID: report.ID, VersionNo: version.VersionNo, ResourceID: file.Files[0].ResourceID, ExpectedSourceRevision: afterFolderDelete.Version.SourceRevision})
	if err != nil || len(afterFileDelete.Files) != 1 || afterFileDelete.Version.SourceRevision != afterFolderDelete.Version.SourceRevision+1 {
		t.Fatalf("file delete snapshot=%+v err=%v", afterFileDelete, err)
	}
	if _, err := db.Exec(`UPDATE report_versions SET state='published' WHERE report_id=? AND version_no=?`, report.ID, version.VersionNo); err != nil {
		t.Fatal(err)
	}
	_, err = client.Resources().UpsertFile(principal, sdk.ResourceFile{ReportID: report.ID, VersionNo: version.VersionNo,
		Namespace: report.OwnerPackage + ".docs", ResourcePath: "guide/late.md", Content: "late",
		ExpectedSourceRevision: afterFileDelete.Version.SourceRevision})
	var immutable *sdk.Error
	if !errors.As(err, &immutable) || immutable.Code != sdk.ErrorConflict {
		t.Fatalf("published version resource mutation error=%v", err)
	}
}

func TestTransportAdministersReportACLThroughSDK(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "acl", Title: "ACL", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "viewer", CanView: true, CanRun: true})
	if err != nil || !entry.CanView || entry.CanEdit || entry.ETag != 1 {
		t.Fatalf("ACL entry=%+v err=%v", entry, err)
	}
	updated, err := client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "viewer", CanView: true, CanRun: true, CanEdit: true, ETag: entry.ETag})
	if err != nil || !updated.CanEdit || updated.ETag != 2 {
		t.Fatalf("updated ACL entry=%+v err=%v", updated, err)
	}
	assertACLETagConflict := func(err error, expected, current int64) {
		t.Helper()
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict || sdkErr.ExpectedETag != expected || sdkErr.CurrentETag != current {
			t.Fatalf("ACL conflict=%+v err=%v, want expected=%d current=%d", sdkErr, err, expected, current)
		}
	}
	_, err = client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "viewer", CanView: false, ETag: entry.ETag})
	assertACLETagConflict(err, 1, 2)
	err = client.ACL().Delete(owner, report.ID, "user", "viewer", entry.ETag)
	assertACLETagConflict(err, 1, 2)

	// Both writers start from the same list snapshot. The conditional etag
	// predicate allows one update and returns structured conflict metadata to
	// the other; neither can silently overwrite the winner.
	start := make(chan struct{})
	type mutationResult struct {
		entry *sdk.ReportACL
		err   error
	}
	results := make(chan mutationResult, 2)
	var writers sync.WaitGroup
	for _, canPublish := range []bool{true, false} {
		writers.Add(1)
		go func(canPublish bool) {
			defer writers.Done()
			<-start
			value, mutationErr := client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "viewer", CanView: true, CanRun: true, CanEdit: true, CanPublish: canPublish, ETag: updated.ETag})
			results <- mutationResult{entry: value, err: mutationErr}
		}(canPublish)
	}
	close(start)
	writers.Wait()
	close(results)
	successes, conflicts := 0, 0
	var winner *sdk.ReportACL
	for result := range results {
		if result.err == nil {
			successes++
			winner = result.entry
			continue
		}
		var sdkErr *sdk.Error
		if errors.As(result.err, &sdkErr) && sdkErr.Code == sdk.ErrorConflict && sdkErr.ExpectedETag == updated.ETag && sdkErr.CurrentETag == updated.ETag+1 {
			conflicts++
			continue
		}
		t.Fatalf("unexpected competing ACL update error=%v", result.err)
	}
	if successes != 1 || conflicts != 1 || winner == nil || winner.ETag != 3 {
		t.Fatalf("competing ACL results successes=%d conflicts=%d winner=%+v", successes, conflicts, winner)
	}
	items, err := client.ACL().List(owner, report.ID)
	if err != nil || len(items) != 1 || items[0].SubjectID != "viewer" || items[0].ETag != winner.ETag {
		t.Fatalf("ACL list=%+v err=%v", items, err)
	}
	nonOwner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "viewer"})
	if _, err = client.ACL().Upsert(nonOwner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "viewer", CanView: true}); err == nil || !strings.Contains(err.Error(), "owner") {
		t.Fatalf("non-owner ACL change error=%v", err)
	}
	if _, err = client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "role", SubjectID: "analyst", CanView: true}); err == nil || !strings.Contains(err.Error(), "verified JWT sub") {
		t.Fatalf("role ACL update=%v, want JWT sub subject validation error", err)
	}
	if _, err = client.ACL().Upsert(owner, sdk.ReportACL{ReportID: report.ID, SubjectType: "user", SubjectID: "unsafe", CanUseDQL: true}); err == nil || !strings.Contains(err.Error(), "view capability") {
		t.Fatalf("inconsistent ACL update=%v", err)
	}
	if err = client.ACL().Delete(owner, report.ID, "user", "viewer", winner.ETag); err != nil {
		t.Fatal(err)
	}
}

func TestTransportGovernsNamespacesThroughSDK(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	namespace, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "finance.ops", Title: "Finance Operations"})
	if err != nil || namespace.OwnerID != "owner" || namespace.Status != "active" {
		t.Fatalf("namespace=%+v err=%v", namespace, err)
	}
	page, err := client.Namespaces().List(owner, sdk.ListNamespacesInput{Query: "finance", Status: "active"})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != namespace.Name {
		t.Fatalf("namespace page=%+v err=%v", page, err)
	}
	description := "Production finance readers"
	updated, err := client.Namespaces().Update(owner, namespace.Name, sdk.UpdateNamespaceInput{Description: &description, ETag: namespace.ETag})
	if err != nil || updated.Description != description || updated.ETag != namespace.ETag+1 {
		t.Fatalf("updated namespace=%+v err=%v", updated, err)
	}
	if _, err = client.Reports().Create(owner, sdk.CreateReportInput{Slug: "ledger", Title: "Ledger", Namespace: namespace.Name, DefaultConnectorName: connector.Name}); err != nil {
		t.Fatal(err)
	}
	if err = client.Namespaces().Delete(owner, namespace.Name, updated.ETag); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("delete referenced namespace error=%v", err)
	}
	empty, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "scratch", Title: "Scratch"})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Namespaces().Delete(owner, empty.Name, empty.ETag); err != nil {
		t.Fatal(err)
	}
}

func TestReaderInspectionRedactsAdvancedDQLByCapability(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='main'`); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "secure", Title: "Secure", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	source := `#package('example.com/secure')
#setting($_ = $route('/secure','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*, type(rows,'Row') FROM (SELECT 1 AS id) rows`
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql) VALUES(?, 'user', 'viewer', TRUE, TRUE, FALSE, FALSE, FALSE)`, report.ID); err != nil {
		t.Fatal(err)
	}
	viewer := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "viewer"})
	inspection, err := client.Versions().Inspect(viewer, report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.DQL != "" || inspection.Version.AuthoredDQL != "" || inspection.Capabilities.CanUseDQL || !inspection.Capabilities.CanRun || inspection.Capabilities.CanEdit {
		t.Fatalf("redacted inspection=%+v version=%+v", inspection, inspection.Version)
	}
	if _, err = db.Exec(`UPDATE report_acl SET can_use_dql=TRUE WHERE report_id=? AND subject_id='viewer'`, report.ID); err != nil {
		t.Fatal(err)
	}
	inspection, err = client.Versions().Inspect(viewer, report.ID, version.VersionNo)
	if err != nil || inspection.DQL == "" || !inspection.Capabilities.CanUseDQL {
		t.Fatalf("DQL-enabled inspection=%+v err=%v", inspection, err)
	}
}

func TestPublicationCompensatesRuntimeWhenActivationPersistenceFails(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,?,?);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','active','main','example.com/reader','reader',1,?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,published_at) VALUES('reader',1,'published','dql','SELECT 1','{}','1','one','{}','valid','v1','v1',1,'owner',?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,validated_at) VALUES('reader',2,'validated','dql','SELECT 2','{}','1','two','{}','valid','v1','v1',2,'owner',?,?);
INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'one','active',1,'{}','owner',?,?);
INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES('reader',1,1,1,1,'active','one','one','owner',?,?);
CREATE TRIGGER reject_publication_activation BEFORE UPDATE OF publication_status ON report_publications
WHEN OLD.publication_status='pending' AND NEW.publication_status='active' AND NEW.failure_json IS NULL
BEGIN SELECT RAISE(ABORT,'activation persistence failed'); END;`,
		now, now, now, now, now, now, now, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	var reloads []int64
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}, Activator: RuntimeActivatorFunc(func(_ context.Context, generation int64) error {
		reloads = append(reloads, generation)
		return nil
	})}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	if _, err = client.Publications().Publish(owner, "reader", 2, sdk.PublishInput{ExpectedSourceRevision: 2}); err == nil || !strings.Contains(err.Error(), "activation persistence failed") {
		t.Fatalf("publication error=%v", err)
	}
	if fmt.Sprint(reloads) != "[2 1]" {
		t.Fatalf("runtime reloads=%v, want candidate then compensation", reloads)
	}
	var activeVersion int
	var activeGeneration int64
	var status string
	var failure sql.NullString
	if err = db.QueryRow(`SELECT active_version_no,active_generation,publication_status,failure_json FROM report_publications WHERE report_id='reader'`).Scan(&activeVersion, &activeGeneration, &status, &failure); err != nil {
		t.Fatal(err)
	}
	if activeVersion != 1 || activeGeneration != 1 || status != "active" || !failure.Valid {
		t.Fatalf("restored publication active=%d generation=%d status=%s failure=%v", activeVersion, activeGeneration, status, failure)
	}
	var generationStatus string
	if err = db.QueryRow(`SELECT status FROM runtime_generations WHERE generation_no=2`).Scan(&generationStatus); err != nil || generationStatus != "failed" {
		t.Fatalf("candidate generation status=%s err=%v", generationStatus, err)
	}
}

func TestPublicationRecoversExpiredBuildingGeneration(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-stagedGenerationLease - time.Minute)
	_, err = db.Exec(`
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,?,?);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','general','reader','Reader','owner','active','main','example.com/reader','reader',1,?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,published_at) VALUES('reader',1,'published','dql','SELECT 1','{}','1','one','{}','valid','v1','v1',1,'owner',?,?);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,validated_at) VALUES('reader',2,'validated','dql','SELECT 2','{}','1','two','{}','valid','v1','v1',2,'owner',?,?);
INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'one','active',1,'{}','owner',?,?);
INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(2,'stale','building',0,'{}','owner',?);
INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at) VALUES('reader',1,2,2,1,'pending','stale','two','owner',?,?);`,
		now, now, now, now, now, now, now, now, now, now, now, stale, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE runtime_generations SET requested_at=? WHERE generation_no=2`, stale); err != nil {
		t.Fatal(err)
	}
	var reloaded int64
	transport := &Transport{DB: db, Now: func() time.Time { return now }, Authorizer: allowAuthorizer{}, Activator: RuntimeActivatorFunc(func(_ context.Context, generation int64) error { reloaded = generation; return nil })}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := client.Publications().Publish(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"}), "reader", 2, sdk.PublishInput{ExpectedSourceRevision: 2})
	if err != nil || publication.ActiveVersionNo != 2 || publication.ActiveGeneration == nil || reloaded != *publication.ActiveGeneration || reloaded != 3 {
		t.Fatalf("publication=%+v reload=%d err=%v", publication, reloaded, err)
	}
	var staleStatus string
	var diagnostics sql.NullString
	if err = db.QueryRow(`SELECT status,diagnostics_json FROM runtime_generations WHERE generation_no=2`).Scan(&staleStatus, &diagnostics); err != nil || staleStatus != "failed" || !diagnostics.Valid {
		t.Fatalf("stale status=%q diagnostics=%v err=%v", staleStatus, diagnostics, err)
	}
}

func reportIDs(items []*sdk.Report) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

func connectorNames(items []*sdk.Connector) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Name)
	}
	return result
}

func isNotFound(err error) bool {
	var sdkErr *sdk.Error
	return errors.As(err, &sdkErr) && sdkErr.Code == sdk.ErrorNotFound
}

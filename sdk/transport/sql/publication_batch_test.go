package sqltransport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/httptransport"
)

func TestPublicationBatchStagesSiblingsBeforeOneReload(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	reloads := 0
	transport.Activator = RuntimeActivatorFunc(func(ctx context.Context, generation int64) error {
		reloads++
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_publications WHERE publication_status='pending' AND desired_generation=? AND desired_version_no=1", generation).Scan(&count); err != nil {
			return err
		}
		if count != 3 {
			return fmt.Errorf("candidate has %d of 3 required siblings", count)
		}
		return nil
	})
	request := map[string]any{"reportId": "first", "versionNo": 1, "input": map[string]any{
		"expectedSourceRevision": 1,
		"additionalVersions": []map[string]any{
			{"reportId": "second", "versionNo": 1, "expectedSourceRevision": 1},
			{"reportId": "third", "versionNo": 1, "expectedSourceRevision": 1},
		},
	}}
	var result sdk.Publication
	require.NoError(t, transport.Invoke(ctx, sdk.OperationPublicationPublish, request, &result))
	require.Equal(t, 1, reloads)
	for _, id := range []string{"first", "second", "third"} {
		published, getErr := client.Publications().Get(ctx, id)
		require.NoError(t, getErr)
		require.Equal(t, "active", published.Status)
		require.Equal(t, result.ActiveGeneration, published.ActiveGeneration)
	}
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT report_count FROM runtime_generations WHERE generation_no=?", *result.ActiveGeneration).Scan(&count))
	require.Equal(t, 3, count)
}

func newPublicationBatchFixture(t *testing.T) (context.Context, *sql.DB, *Transport, sdk.Client) {
	t.Helper()
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "batch.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	require.NoError(t, err)
	_, err = client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE connectors SET status='active' WHERE name='main'")
	require.NoError(t, err)
	for _, id := range []string{"first", "second", "third"} {
		_, err = client.Components().Create(ctx, sdk.CreateComponentInput{ID: id, Slug: id, Title: id, DefaultConnectorName: "main"})
		require.NoError(t, err)
		v, createErr := client.Versions().Create(ctx, id, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
		require.NoError(t, createErr)
		_, err = client.Versions().Validate(ctx, id, v.VersionNo, v.SourceRevision)
		require.NoError(t, err)
	}

	return ctx, db, transport, client
}

func batchPublicationInput(version int) sdk.PublishInput {
	return sdk.PublishInput{ExpectedSourceRevision: 1, AdditionalVersions: []sdk.PublishVersion{
		{ReportID: "second", VersionNo: version, ExpectedSourceRevision: 1},
		{ReportID: "third", VersionNo: version, ExpectedSourceRevision: 1},
	}}
}

func TestPublicationBatchReloadFailureRestoresEveryMember(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return nil })
	old, err := client.Publications().Publish(ctx, "first", 1, batchPublicationInput(1))
	require.NoError(t, err)
	for _, id := range []string{"first", "second", "third"} {
		v, err := client.Versions().Create(ctx, id, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 2"})
		require.NoError(t, err)
		_, err = client.Versions().Validate(ctx, id, v.VersionNo, v.SourceRevision)
		require.NoError(t, err)
	}
	failure := errors.New("synthetic component compilation failure")
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return failure })
	_, err = client.Publications().Publish(ctx, "first", 2, batchPublicationInput(2))
	require.ErrorIs(t, err, failure)
	for _, id := range []string{"first", "second", "third"} {
		p, err := client.Publications().Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, p.ActiveVersionNo)
		require.Equal(t, old.ActiveGeneration, p.ActiveGeneration)
		require.Equal(t, "active", p.Status)
		v, err := client.Versions().Get(ctx, id, 2)
		require.NoError(t, err)
		require.Equal(t, "draft", v.State)
	}
	var pending int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_publications WHERE publication_status='pending'").Scan(&pending))
	require.Zero(t, pending)
}

func TestPublicationBatchMidActivationFailureRollsBackEveryWrite(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return nil })
	old, err := client.Publications().Publish(ctx, "first", 1, batchPublicationInput(1))
	require.NoError(t, err)
	for _, id := range []string{"first", "second", "third"} {
		v, err := client.Versions().Create(ctx, id, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 2"})
		require.NoError(t, err)
		_, err = client.Versions().Validate(ctx, id, v.VersionNo, v.SourceRevision)
		require.NoError(t, err)
	}
	_, err = db.ExecContext(ctx, `CREATE TRIGGER fail_second_activation BEFORE UPDATE ON component_publications
	WHEN NEW.report_id='second' AND NEW.active_version_no=2
	BEGIN SELECT RAISE(ABORT, 'fixture persistence failure'); END`)
	require.NoError(t, err)
	var reloads []int64
	transport.Activator = RuntimeActivatorFunc(func(_ context.Context, generation int64) error { reloads = append(reloads, generation); return nil })
	_, err = client.Publications().Publish(ctx, "first", 2, batchPublicationInput(2))
	require.Error(t, err)
	require.Len(t, reloads, 2)
	require.Equal(t, *old.ActiveGeneration, reloads[1])
	for _, id := range []string{"first", "second", "third"} {
		p, err := client.Publications().Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, 1, p.ActiveVersionNo)
		require.Equal(t, old.ActiveGeneration, p.ActiveGeneration)
		v, err := client.Versions().Get(ctx, id, 1)
		require.NoError(t, err)
		require.Equal(t, "published", v.State)
		v, err = client.Versions().Get(ctx, id, 2)
		require.NoError(t, err)
		require.Equal(t, "draft", v.State)
	}
}

func TestPublicationBatchRejectsInvalidMemberBeforeStaging(t *testing.T) {
	for _, scenario := range []string{"denied", "stale", "duplicate", "invalid", "namespace"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, db, transport, client := newPublicationBatchFixture(t)
			reloads := 0
			transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { reloads++; return nil })
			input := batchPublicationInput(1)
			switch scenario {
			case "denied":
				transport.Authorizer = AuthorizerFunc(func(_ context.Context, r AuthorizationRequest) error {
					if r.Operation == sdk.OperationPublicationPublish && r.ReportID == "third" {
						return &sdk.Error{Code: sdk.ErrorForbidden, Message: "denied"}
					}
					return nil
				})
			case "stale":
				input.AdditionalVersions[1].ExpectedSourceRevision = 2
			case "duplicate":
				input.AdditionalVersions[1].ReportID = "first"
			case "invalid":
				input.AdditionalVersions[1].VersionNo = 0
			case "namespace":
				_, err := db.ExecContext(ctx, "UPDATE components SET namespace='another' WHERE id='third'")
				require.NoError(t, err)
			}
			_, err := client.Publications().Publish(ctx, "first", 1, input)
			require.Error(t, err)
			require.Zero(t, reloads)
			var count int
			require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM runtime_generations").Scan(&count))
			require.Zero(t, count)
		})
	}
}

func TestPublicationBatchRejectsRollbackExtras(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { t.Fatal("rollback must not stage a batch"); return nil })
	_, err := client.Publications().Rollback(ctx, "first", 1, batchPublicationInput(1))
	var typed *sdk.Error
	require.ErrorAs(t, err, &typed)
	require.Equal(t, sdk.ErrorInvalidArgument, typed.Code)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM runtime_generations").Scan(&count))
	require.Zero(t, count)
}

func TestPublicationBatchActivationConflictRestoresInitialMembers(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	var reloads []int64
	transport.Activator = RuntimeActivatorFunc(func(ctx context.Context, generation int64) error {
		reloads = append(reloads, generation)
		if generation > 0 {
			_, err := db.ExecContext(ctx, "UPDATE component_versions SET source_revision=source_revision+1 WHERE report_id='third' AND version_no=1")
			return err
		}
		return nil
	})
	_, err := client.Publications().Publish(ctx, "first", 1, batchPublicationInput(1))
	require.Error(t, err)
	require.Len(t, reloads, 2)
	require.Zero(t, reloads[1])
	var active, pending int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_publications WHERE active_generation IS NOT NULL").Scan(&active))
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_publications WHERE publication_status='pending'").Scan(&pending))
	require.Zero(t, active)
	require.Zero(t, pending)
}

func TestPublicationBatchCancellationRestoresAllStagedMembers(t *testing.T) {
	ctx, db, transport, client := newPublicationBatchFixture(t)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	transport.Activator = RuntimeActivatorFunc(func(ctx context.Context, generation int64) error { cancel(); return ctx.Err() })
	_, err := client.Publications().Publish(ctx, "first", 1, batchPublicationInput(1))
	require.ErrorIs(t, err, context.Canceled)
	var pending int
	require.NoError(t, db.QueryRowContext(context.WithoutCancel(ctx), "SELECT COUNT(*) FROM component_publications WHERE publication_status='pending'").Scan(&pending))
	require.Zero(t, pending)
}

func TestPublicationBatchHTTPBoundary(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(fmt.Sprint(denied), func(t *testing.T) {
			ctx, db, transport, _ := newPublicationBatchFixture(t)
			transport.Activator = RuntimeActivatorFunc(func(context.Context, int64) error { return nil })
			if denied {
				transport.Authorizer = AuthorizerFunc(func(_ context.Context, r AuthorizationRequest) error {
					if r.Permission == "publish" && r.ReportID == "second" {
						return &sdk.Error{Code: sdk.ErrorForbidden, Message: "publication denied"}
					}
					return nil
				})
			}
			gateway := httptransport.Gateway{Transport: transport, Config: httptransport.Config{Mode: httptransport.Authenticated, Authenticator: httptransport.AuthenticatorFunc(func(context.Context, *http.Request) (context.Context, error) { return ctx, nil })}}
			body, err := json.Marshal(map[string]any{"reportId": "first", "versionNo": 1, "input": batchPublicationInput(1)})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, httptransport.PathPrefix+sdk.OperationPublicationPublish, bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			gateway.ServeHTTP(response, request)
			if denied {
				require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
				var count int
				require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM runtime_generations").Scan(&count))
				require.Zero(t, count)
			} else {
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				var result sdk.Publication
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				require.Equal(t, "active", result.Status)
				var count int
				require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM component_publications WHERE active_generation=?", *result.ActiveGeneration).Scan(&count))
				require.Equal(t, 3, count)
			}
		})
	}
}

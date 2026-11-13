package connectivity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	bq "github.com/viant/bigquery"
	"github.com/viant/datly-studio/sdk"
	"google.golang.org/api/googleapi"
	apioption "google.golang.org/api/option"
)

func TestSchemaSearchServiceScopeReuseRefreshAndErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/projects/source/datasets/data/tables":
			require.Equal(t, "25", r.URL.Query().Get("maxResults"))
			if r.URL.Query().Get("pageToken") == "next" {
				_, _ = w.Write([]byte(`{"tables":[{"tableReference":{"projectId":"source","datasetId":"data","tableId":"items"}}]}`))
			} else {
				_, _ = w.Write([]byte(`{"tables":[{"tableReference":{"projectId":"source","datasetId":"data","tableId":"records"}}],"nextPageToken":"next"}`))
			}
		case "/projects/source/datasets/data/tables/records", "/projects/source/datasets/data/tables/items":
			_, _ = w.Write([]byte(`{"schema":{"fields":[{"name":"id","type":"INTEGER","description":"Record identity"}]}}`))
		case "/projects/source/datasets/data/tables/denied":
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"metadata denied"}}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	bq.SetOptions(apioption.WithEndpoint(server.URL+"/"), apioption.WithHTTPClient(server.Client()))
	defer bq.SetOptions()
	service := NewSchemaSearch(SQLCatalog{}, sdk.SchemaSearchOptions{})
	defer service.Close()
	connector := &sdk.Connector{Name: "warehouse", Driver: "bigquery", DSNTemplate: "bigquery://billing/default?credJSON=%7B%7D"}
	principal := sdk.Principal{Subject: "reader"}
	input := sdk.SchemaSearchInput{Catalog: "source", Schema: "data", DescriptionTerms: []string{"identity"}}
	ctx := context.Background()
	first, err := service.Search(ctx, connector, principal, input)
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	require.False(t, first.MetadataComplete)
	require.Equal(t, "next", first.NextMetadataCursor)
	require.EqualValues(t, 2, calls.Load())
	warm, err := service.Search(ctx, connector, principal, input)
	require.NoError(t, err)
	require.Equal(t, first.SnapshotID, warm.SnapshotID)
	require.EqualValues(t, 2, calls.Load())
	input.MetadataCursor = "next"
	next, err := service.Search(ctx, connector, principal, input)
	require.NoError(t, err)
	require.Equal(t, "items", next.Items[0].Table.Name)
	require.False(t, next.MetadataComplete, "terminal partial page is not a complete dataset index")
	require.Empty(t, next.NextMetadataCursor)
	require.NotEqual(t, first.SnapshotID, next.SnapshotID)
	input.MetadataCursor = ""
	isolated, err := service.Search(ctx, connector, sdk.Principal{Subject: "other"}, input)
	require.NoError(t, err)
	require.NotEqual(t, first.SnapshotID, isolated.SnapshotID)
	input.Refresh = true
	refreshed, err := service.Search(ctx, connector, principal, input)
	require.NoError(t, err)
	require.NotEqual(t, first.SnapshotID, refreshed.SnapshotID)
	require.EqualValues(t, 8, calls.Load())
	input.Refresh = false
	input.Table = "records"
	exact, err := service.Search(ctx, connector, principal, input)
	require.NoError(t, err)
	require.Len(t, exact.Items, 1)
	require.True(t, exact.MetadataComplete)
	require.EqualValues(t, 9, calls.Load(), "exact table must skip listing")
	input.Table = "denied"
	for i := 0; i < 2; i++ {
		_, err = service.Search(ctx, connector, principal, input)
		var cause *googleapi.Error
		require.ErrorAs(t, err, &cause)
		require.Equal(t, 403, cause.Code)
	}
	require.EqualValues(t, 11, calls.Load(), "failures must not become cached empty results")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = service.Search(canceled, connector, principal, input)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 11, calls.Load())
}

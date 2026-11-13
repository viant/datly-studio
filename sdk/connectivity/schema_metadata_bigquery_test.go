package connectivity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	bq "github.com/viant/bigquery"
	"github.com/viant/datly-studio/internal/schemasearch"
	"github.com/viant/datly-studio/sdk"
	"google.golang.org/api/googleapi"
	apioption "google.golang.org/api/option"
)

func TestBigQueryMetadataSnapshotLoadsOnceAndUsesOnlyAPIs(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "GET", r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/projects/source/datasets/data/tables":
			require.Equal(t, "2", r.URL.Query().Get("maxResults"))
			_, _ = w.Write([]byte(`{"tables":[{"tableReference":{"projectId":"source","datasetId":"data","tableId":"records"}},{"tableReference":{"projectId":"source","datasetId":"data","tableId":"items"}}],"nextPageToken":"next"}`))
		case "/projects/source/datasets/data/tables/records", "/projects/source/datasets/data/tables/items":
			_, _ = w.Write([]byte(`{"schema":{"fields":[{"name":"id","type":"INTEGER","description":"Record identity"}]}}`))
		case "/projects/source/datasets/data/tables/denied":
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"metadata denied"}}`))
		default:
			t.Errorf("unexpected metadata request %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	bq.SetOptions(apioption.WithEndpoint(server.URL+"/"), apioption.WithoutAuthentication())
	defer bq.SetOptions()
	connector := &sdk.Connector{Name: "warehouse", Driver: "bigquery", DSNTemplate: "bigquery://billing/default"}
	input := SchemaMetadataInput{Catalog: "source", Schema: "data", Limit: 2}
	cache := schemasearch.NewCache(schemasearch.CacheOptions{})
	defer cache.Close()
	key := schemasearch.Scope{CallerIdentity: "fixture", ConnectorIdentity: "warehouse", CredentialIdentity: "fixture", Catalog: "source", Schema: "data"}
	load := func(ctx context.Context) (*schemasearch.Metadata, error) {
		p, e := (SQLCatalog{}).ColumnMetadataPage(ctx, connector, input)
		if e != nil {
			return nil, e
		}
		return &schemasearch.Metadata{Tables: p.Tables, Complete: p.Complete, Continuation: p.NextPageToken}, nil
	}
	first, err := cache.Get(context.Background(), key, false, load)
	require.NoError(t, err)
	require.False(t, first.Complete)
	require.Equal(t, "next", first.Continuation)
	require.EqualValues(t, 3, calls.Load())
	warm, err := cache.Get(context.Background(), key, false, load)
	require.NoError(t, err)
	require.Same(t, first, warm)
	require.EqualValues(t, 3, calls.Load(), "warm search must not repeat source metadata calls")
	found, err := warm.Index.Search(context.Background(), schemasearch.Query{DescriptionTerms: []string{"identity"}})
	require.NoError(t, err)
	require.Len(t, found.Items, 2)
	_, err = cache.Get(context.Background(), key, true, load)
	require.NoError(t, err)
	require.EqualValues(t, 6, calls.Load())
	_, err = (SQLCatalog{}).ColumnMetadataPage(context.Background(), connector, SchemaMetadataInput{Catalog: "source", Schema: "data", Table: "denied"})
	var cause *googleapi.Error
	require.ErrorAs(t, err, &cause)
	require.Equal(t, 403, cause.Code)
	require.True(t, strings.Contains(err.Error(), "denied"))
}

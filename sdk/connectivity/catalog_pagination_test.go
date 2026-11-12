package connectivity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	bq "github.com/viant/bigquery"
	"github.com/viant/datly-studio/sdk"
	"google.golang.org/api/googleapi"
	apioption "google.golang.org/api/option"
)

func TestCatalogTablePages(t *testing.T) {
	calls, status := 0, 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/projects/source/datasets/data/tables", r.URL.Path)
		require.Equal(t, "7", r.URL.Query().Get("maxResults"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != 200 {
			_, _ = w.Write([]byte(`{"error":{"message":"denied"}}`))
			return
		}
		if r.URL.Query().Get("pageToken") == "next" {
			_, _ = w.Write([]byte(`{"tables":[{"tableReference":{"projectId":"source","datasetId":"data","tableId":"matching"},"type":"VIEW"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"tables":[{"tableReference":{"projectId":"source","datasetId":"data","tableId":"other"},"type":"TABLE"}],"nextPageToken":"next"}`))
	}))
	defer server.Close()
	bq.SetOptions(apioption.WithEndpoint(server.URL+"/"), apioption.WithoutAuthentication())
	defer bq.SetOptions()
	connector := &sdk.Connector{Name: "source", Driver: "bigquery", DSNTemplate: "bigquery://billing/default"}
	input := sdk.TableCatalogInput{Catalog: "source", Schema: "data", Limit: 7, Query: "match"}
	catalog := SQLCatalog{}
	page, err := catalog.Tables(context.Background(), connector, input)
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Equal(t, "next", page.NextPageToken)
	require.Equal(t, 1, calls)
	input.PageToken = page.NextPageToken
	page, err = catalog.Tables(context.Background(), connector, input)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "matching", page.Items[0].Name)
	require.Equal(t, "VIEW", page.Items[0].Type)
	require.Empty(t, page.NextPageToken)
	require.Equal(t, 2, calls)
	status = 403
	_, err = catalog.Tables(context.Background(), connector, input)
	var cause *googleapi.Error
	require.ErrorAs(t, err, &cause)
	require.Equal(t, 403, cause.Code)
	before := calls
	input.Offset = 7
	_, err = catalog.Tables(context.Background(), connector, input)
	require.ErrorContains(t, err, "pageToken")
	_, err = catalog.Tables(context.Background(), connector, sdk.TableCatalogInput{})
	require.ErrorContains(t, err, "explicit catalog and schema")
	require.Equal(t, before, calls)
}

func TestCatalogKnownTableNeedsNoQueryJob(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/projects/source/datasets/data/tables/records", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema":{"fields":[{"name":"id","type":"INTEGER","mode":"REQUIRED"},{"name":"label","type":"STRING"}]}}`))
	}))
	defer server.Close()
	bq.SetOptions(apioption.WithEndpoint(server.URL+"/"), apioption.WithoutAuthentication())
	defer bq.SetOptions()
	connector := &sdk.Connector{Name: "source", Driver: "bigquery", DSNTemplate: "bigquery://billing/default"}
	result, err := (SQLCatalog{}).Table(context.Background(), connector, sdk.TableDetailInput{Catalog: "source", Schema: "data", Table: "records"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, result.Columns, 2)
	require.Equal(t, "INT64", result.Columns[0].Type)
	require.Equal(t, "catalog", result.MetadataSource)
	require.False(t, result.ConstraintsKnown)
	require.False(t, result.Columns[0].Nullable)
}

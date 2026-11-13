package connectivity

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/sqlx/metadata/sink"
	"os"
	"testing"
	"time"
)

func TestColumnMetadataPageKeepsQualifiedTablesAndUnknownConstraints(t *testing.T) {
	page := columnMetadataPage([]sink.Column{
		{Catalog: "source", Schema: "one", Table: "records", Name: "id", Type: "INT", Nullable: "NO", Comments: "Record identity"},
		{Catalog: "source", Schema: "one", Table: "records", Name: "label", Type: "TEXT", Nullable: "YES"},
		{Catalog: "source", Schema: "two", Table: "records", Name: "id", Type: "INT"},
	}, "continue")
	require.Len(t, page.Tables, 2)
	require.Len(t, page.Tables[0].Columns, 2)
	require.Equal(t, "Record identity", page.Tables[0].Columns[0].Description)
	require.False(t, page.Tables[0].ConstraintsKnown)
	require.False(t, page.Complete)
	require.Equal(t, "continue", page.NextPageToken)
	require.False(t, *page.Tables[1].Columns[0].NullableKnown)
	require.True(t, columnMetadataPage(nil, "").Complete)
}

func TestColumnMetadataPageNativeMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_METADATA_MYSQL_DSN")
	if dsn == "" {
		t.Skip("isolated metadata fixture not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()
	name := fmt.Sprintf("studio_metadata_%d", time.Now().UnixNano())
	_, err = db.ExecContext(ctx, "CREATE DATABASE `"+name+"`")
	require.NoError(t, err)
	defer db.Exec("DROP DATABASE `" + name + "`")
	_, err = db.ExecContext(ctx, "CREATE TABLE `"+name+"`.records(id INT, label VARCHAR(20) COMMENT 'Human label')")
	require.NoError(t, err)
	connector := &sdk.Connector{Name: "fixture", Driver: "mysql", DSNTemplate: dsn}
	page, err := (SQLCatalog{}).ColumnMetadataPage(ctx, connector, SchemaMetadataInput{Schema: name, Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Tables, 1)
	require.Len(t, page.Tables[0].Columns, 1)
	require.False(t, page.Complete)
	require.NotEmpty(t, page.NextPageToken)
	next, err := (SQLCatalog{}).ColumnMetadataPage(ctx, connector, SchemaMetadataInput{Schema: name, Limit: 1, PageToken: page.NextPageToken})
	require.NoError(t, err)
	require.Len(t, next.Tables, 1)
	require.Equal(t, "label", next.Tables[0].Columns[0].Name)
	require.Equal(t, "Human label", next.Tables[0].Columns[0].Description)
	require.True(t, next.Complete)
}

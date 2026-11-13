package host

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/sdk"
)

func TestSchemaSearchIsHostOwned(t *testing.T) {
	_, err := SchemaSearchFromContext(context.Background())
	require.Error(t, err, "unbound requests must not allocate a temporary cache")
	first, err := NewAuthoring(AuthoringConfig{})
	require.NoError(t, err)
	defer first.Close()
	second, err := NewAuthoring(AuthoringConfig{})
	require.NoError(t, err)
	defer second.Close()
	var seen []sdk.SchemaSearcher
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searcher, err := SchemaSearchFromContext(r.Context())
		require.NoError(t, err)
		seen = append(seen, searcher)
	})
	for i := 0; i < 2; i++ {
		first.Middleware(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}
	second.Middleware(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	require.Same(t, seen[0], seen[1])
	require.NotSame(t, seen[0], seen[2])
}

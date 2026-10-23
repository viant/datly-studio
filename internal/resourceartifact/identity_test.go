package resourceartifact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentitySeparatesSourceAndInputWithoutExposingSource(t *testing.T) {
	sha := strings.Repeat("a", 64)
	first, err := New("component://namespace/report@1", sha)
	require.NoError(t, err)
	replay, err := New("component://namespace/report@1", sha)
	require.NoError(t, err)
	require.Equal(t, first, replay)
	require.NotContains(t, first.URI, "report")
	require.True(t, strings.HasPrefix(first.URI, "compiled://"))
	require.Equal(t, "compiled/"+first.Key+".json", first.Path)
	changed, err := New("component://namespace/report@2", sha)
	require.NoError(t, err)
	require.NotEqual(t, first.Key, changed.Key)
	_, err = New("component://namespace/report@1", "bad")
	require.Error(t, err)
}

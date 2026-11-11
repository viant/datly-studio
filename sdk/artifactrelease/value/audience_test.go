package value

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	contract "github.com/viant/forge/backend/artifactrelease"
	"testing"
)

func TestAudienceJSONRetainsPrivateArraysAndRequiresExplicitPublic(t *testing.T) {
	channel := contract.Channel{Key: contract.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "shared", Name: "main"}, Policies: []authz.NamedReference{{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}}}
	legacy, _ := json.Marshal(channel.Policies)
	raw, err := EncodeAudience(channel)
	require.NoError(t, err)
	require.Equal(t, legacy, raw)
	decoded := contract.Channel{Key: channel.Key}
	require.NoError(t, DecodeAudience(raw, &decoded))
	require.Equal(t, channel, decoded)
	channel.Public = true
	channel.Policies = nil
	raw, err = EncodeAudience(channel)
	require.NoError(t, err)
	require.JSONEq(t, `{"public":true}`, string(raw))
	require.NoError(t, DecodeAudience(raw, &decoded))
	require.Equal(t, channel, decoded)
	for _, raw := range []string{`[]`, `null`, `{}`, `{"public":false}`, `{"public":true,"policies":[]}`, `{"public":true,"other":1}`} {
		require.Error(t, DecodeAudience([]byte(raw), &decoded), raw)
	}
}

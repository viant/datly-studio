package host

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	identity "github.com/viant/forge/backend/resource"
	"testing"
)

func TestStandaloneDatasourceConfigRequiresProviderIdentity(t *testing.T) {
	component := ComponentReference{Kind: "linked", ID: "rows", Revision: "immutable", Method: "GET", Route: "/rows"}
	policy := authz.Policy{Mode: "public"}
	family := authz.ResourceFamily{Kind: "datasource", ID: "team/rows", Tenant: "team"}
	config := &Config{Access: &ResourceAccessConfig{Tenant: "team"}, Forge: &ForgeConfig{DataSources: []ForgeDataSource{{URI: "datasource://team/rows", DefinitionPath: "rows.json", Component: component}}, Policies: []authz.Document{{Resource: authz.Resource{Kind: family.Kind, ID: family.ID, Tenant: family.Tenant, Version: identity.WorkingCandidate}, Revision: 1, Policies: map[string]authz.Policy{"discover": policy, "describe": policy, "execute": policy}}}, Selections: []authz.SelectionDocument{{Resource: family, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}}
	require.ErrorContains(t, validateForgeConfig(config), "require ProviderIdentity")
	config.Forge.ProviderIdentity = "configured-producer"
	require.NoError(t, validateForgeConfig(config))
	// A window-only deployment retains its preexisting optional identity contract.
	config.Forge.ProviderIdentity = ""
	config.Forge.DataSources = nil
	config.Forge.Windows = []ForgeWindow{{URI: "window://team/overview", DefinitionPath: "overview.json", Components: map[string]ComponentReference{"rows": component}}}
	config.Forge.Policies[0].Resource.Kind = "window"
	config.Forge.Policies[0].Resource.ID = "team/overview"
	config.Forge.Selections[0].Resource.Kind = "window"
	config.Forge.Selections[0].Resource.ID = "team/overview"
	require.NoError(t, validateForgeConfig(config))
}

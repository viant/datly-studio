package preview

import (
	"context"
	"github.com/viant/bindly/resource"
	"strings"
	"testing"
	"testing/fstest"
)

func TestViewDQLUsesVersionResourcesAndSelectedConnector(t *testing.T) {
	store, err := resource.New().WithDefault(fstest.MapFS{"sql/root.sql": {Data: []byte("SELECT ID FROM events")}, "sql/channel.sql": {Data: []byte("SELECT ID,NAME FROM channels")}})
	if err != nil {
		t.Fatal(err)
	}
	definition := &definition{Scope: "example.com/fixture", Name: "reader", Connector: "main", Connectors: []connectorDefinition{{Name: "main"}, {Name: "lookup"}}, Resources: store, DQL: `#package('example.com/fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/fixture','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT root.*,channel.*,type(root,'Row'),type(channel,'Channel'),use_connector(channel,'lookup')
FROM (${embed:sql/root.sql}) root
JOIN (${embed:sql/channel.sql}) channel ON root.ID=channel.ID AND 1=1`}
	source, err := testViewDQL(context.Background(), definition, "channel", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "$connector('lookup')") || !strings.Contains(source, "${embed:sql/channel.sql}") || strings.Contains(source, "JOIN") {
		t.Fatalf("incorrect view test source: %s", source)
	}
	connectors := requiredConnectors(context.Background(), definition, source, []string{"main", "lookup"})
	if len(connectors) != 2 || connectors[0] != "lookup" || connectors[1] != "main" {
		t.Fatalf("embedded view connectors=%v", connectors)
	}
}

func TestViewDQLReturnsErrorForMissingComponentAuthority(t *testing.T) {
	_, err := testViewDQL(context.Background(), &definition{Scope: "example.com/fixture", Name: "reader", Connector: "main", DQL: "SELECT * FROM (${embed:missing.sql}) root"}, "root", nil)
	if err == nil {
		t.Fatal("missing component authority was accepted")
	}
}

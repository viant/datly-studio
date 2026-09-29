package reader

import (
	"context"
	"encoding/json"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"reflect"
	"testing"
)

func TestNamespacePageCarriesSettings(t *testing.T) {
	roles := `["analyst"]`
	input := &NamespaceQueryInput{Limit: 25, Offset: 10}
	ctx := context.WithValue(context.Background(), reflect.TypeFor[*NamespaceQueryInput](), input)
	output := &NamespaceQueryOutput{Items: []*NamespaceRecord{{OwnerId: "alice", Name: "forecasting", AllowedRolesJson: &roles}}}
	if err := output.Finalize(ctx); err != nil {
		t.Fatal(err)
	}
	row := output.Items[0]
	if row.NamespaceId != namespaceaccess.ID("alice", "forecasting") || row.Visibility != "private" || !reflect.DeepEqual(row.AllowedRoles, []string{"analyst"}) || output.PageLimit != 25 || output.PageOffset != 10 {
		t.Fatalf("incomplete namespace page: %+v", output)
	}
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Items []struct {
			Roles []string `json:"allowedRoles"`
		} `json:"items"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil || len(wire.Items) != 1 || !reflect.DeepEqual(wire.Items[0].Roles, []string{"analyst"}) {
		t.Fatalf("namespace roles missing from wire: %s err=%v", payload, err)
	}
	roles = `invalid`
	if err := output.Finalize(ctx); err == nil {
		t.Fatal("malformed assigned roles accepted")
	}
}

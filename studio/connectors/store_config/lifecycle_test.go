package store_config

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestConnectorConfigRulesPreserveOrInvalidateProbe(t *testing.T) {
	now := time.Now().UTC()
	etag := int64(4)
	passed := "passed"
	previous := &StoredConnector{Name: "source", Driver: "sqlite", Status: "active", Etag: &etag,
		OptionsJson: []byte(`{"pool":1}`), LastTestStatus: &passed, LastTestedAt: &now}
	state := xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredConnector, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredConnector {
		version := etag
		value := &StoredConnector{}
		value.SetName("source")
		value.SetDriver("sqlite")
		value.SetDsnTemplate(nil)
		value.SetSecretRef(nil)
		value.SetDescription(nil)
		value.SetOptionsJson([]byte(`{"pool":1}`))
		value.SetEtag(&version)
		value.SetUpdatedAt(&now)
		return value
	}
	rules := &ConnectorConfigRules{}
	descriptionOnly := row()
	description := "edited"
	descriptionOnly.SetDescription(&description)
	if err := rules.Init(context.Background(), descriptionOnly, state); err != nil {
		t.Fatal(err)
	}
	if descriptionOnly.Has.Status || descriptionOnly.Has.LastTestStatus || *descriptionOnly.Etag != etag+1 {
		t.Fatalf("description edit should preserve status and probe: %+v", descriptionOnly)
	}
	connectionEdit := row()
	connectionEdit.SetDriver("mysql")
	if err := rules.Init(context.Background(), connectionEdit, state); err != nil {
		t.Fatal(err)
	}
	if connectionEdit.Status != "draft" || !connectionEdit.Has.Status ||
		!connectionEdit.Has.LastTestStatus || !connectionEdit.Has.LastTestErrorCode || !connectionEdit.Has.LastTestedAt ||
		connectionEdit.LastTestStatus != nil || connectionEdit.LastTestedAt != nil {
		t.Fatalf("connection edit must invalidate probe: %+v", connectionEdit)
	}
	stale := row()
	*stale.Etag--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale config error = %v", err)
	}
	previous.DeletedAt = &now
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("deleted config error = %v", err)
	}
}

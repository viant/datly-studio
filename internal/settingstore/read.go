package settingstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/studio/settings"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/namespace_settings/store_read"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// Reader resolves exact draft/version metadata via a server-only
// Datly v1 reader. A Studio API host may reuse one instance for all previews.
type Reader struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func NewReader(db *sql.DB) (*Reader, error) {
	resources := resource.New()
	if err := resources.Register(stored.SettingsDatlyResourceNamespace, stored.SettingsDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.SettingsComponent{}), "store_read",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &Reader{runtime: runtime, target: target}, nil
}

// Read returns independent policy values, scoped by the complete namespace identity.
func Read(ctx context.Context, db *sql.DB, owner, namespace string) (*settings.Namespace, error) {
	reader, err := NewReader(db)
	if err != nil {
		return nil, err
	}
	defer reader.runtime.Shutdown(ctx)
	input := &stored.Input{}
	input.SetOwnerId(owner)
	input.SetNamespace(namespace)
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("namespace settings reader returned %T", value)
	}
	result := &settings.Namespace{}
	for _, row := range output.Settings {
		if row == nil || row.OwnerId != owner || row.Namespace != namespace {
			return nil, fmt.Errorf("settings ownership mismatch")
		}
		decoded, err := settings.Decode(row.SettingKey, row.ValueJson)
		if err != nil {
			return nil, err
		}
		switch v := decoded.(type) {
		case *settings.Security:
			result.Security = v
		case *settings.ResourceLimits:
			result.ResourceLimits = v
		}
	}
	return result, nil
}

// Values preserves only explicitly supplied setting keys.
func Values(value *settings.Namespace) (map[string]json.RawMessage, error) {
	result := map[string]json.RawMessage{}
	if value == nil {
		return result, nil
	}
	for key, v := range map[string]any{settings.SecurityKey: value.Security, settings.ResourceLimitsKey: value.ResourceLimits} {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if string(data) == "null" {
			continue
		}
		if _, err = settings.Decode(key, data); err != nil {
			return nil, err
		}
		result[key] = data
	}
	return result, nil
}

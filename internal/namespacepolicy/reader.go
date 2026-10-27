package namespacepolicy

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/authz"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/namespaceroles"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/resource_policy/store_namespace"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type Reader struct {
	runtime *druntime.Runtime
	target  exec.ComponentTarget
}

func NewReader(db *sql.DB) (*Reader, error) {
	resources := resource.New()
	if err := resources.Register(stored.NamespaceDatlyResourceNamespace, stored.NamespaceDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeFor[stored.NamespaceComponent](), "store_namespace", reflect.TypeFor[stored.Input](), reflect.TypeFor[stored.Output](), resources, connector)
	if err != nil {
		return nil, err
	}
	registration.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &Reader{runtime: runtime, target: target}, nil
}

func (r *Reader) read(ctx context.Context, namespaceID, componentID string) (*stored.Namespace, error) {
	input := &stored.Input{}
	input.SetNamespaceId(namespaceID)
	input.SetComponentId(componentID)
	value, err := r.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: r.target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("namespace policy reader returned %T", value)
	}
	if len(output.Namespaces) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Namespaces) != 1 || output.Namespaces[0] == nil {
		return nil, fmt.Errorf("ambiguous namespace policy scope")
	}
	row := output.Namespaces[0]
	if row.NamespaceId == "" || namespaceID != "" && row.NamespaceId != namespaceID {
		return nil, authz.ErrDenied
	}
	return row, nil
}

func (r *Reader) Namespace(ctx context.Context, componentID string) (string, error) {
	row, err := r.read(ctx, "", componentID)
	if err != nil {
		return "", err
	}
	// An inactive namespace must not fall through to public component defaults.
	if row.Status != "active" {
		return "", authz.ErrDenied
	}
	return row.NamespaceId, nil
}

func (r *Reader) CanManage(ctx context.Context, facts authz.Facts, resource authz.Resource) (bool, error) {
	if resource.Kind != "namespace" || resource.Version != "1" || resource.Tenant != facts.Tenant || facts.Subject == "" {
		return false, nil
	}
	row, err := r.read(ctx, resource.ID, "")
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if namespacevisibility.HasAdministratorRole(facts) {
		return row.Status == "active", nil
	}
	grants, err := namespaceroles.Decode(row.RolesJson)
	if err != nil {
		return false, err
	}
	return row.Status == "active" && (row.OwnerId == facts.Subject || namespaceroles.Matches(grants.Manage, facts.Roles)), nil
}

func (r *Reader) Close(ctx context.Context) error { return r.runtime.Shutdown(ctx) }

package connectorcatalog

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	connectorget "github.com/viant/datly-studio/studio/connectors/get"
	access "github.com/viant/datly-studio/studio/connectors/store_access"
	stored "github.com/viant/datly-studio/studio/connectors/store_catalog"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
)

// Authorized loads server-only connection material only after the exact
// public connector read authorizes the verified principal. The private
// catalog is independently scoped to that principal.
func Authorized(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, name string) (*sdk.Connector, error) {
	return resolve(ctx, session, claims, auth, name, false)
}

// ForProbe additionally requires connector edit permission and permits draft
// connectors so a successful test can precede activation.
func ForProbe(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, name string) (*sdk.Connector, error) {
	return resolve(ctx, session, claims, auth, name, true)
}

func resolve(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, name string, probe bool) (*sdk.Connector, error) {
	if claims == nil || auth == nil || auth.Auth == nil || claims.Subject == "" || auth.Auth.Subject != claims.Subject {
		return nil, publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, publisherguard.PublicError(400, "connector name is required")
	}
	if session == nil || session.Binder() == nil {
		return nil, fmt.Errorf("connector catalog session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, fmt.Errorf("Datly component invoker is unavailable")
	}
	if probe {
		guard := &access.Input{}
		guard.SetName(name)
		guard.SetSubject(claims.Subject)
		guard.SetPermission("edit")
		value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[access.ConnectorComponent](), "connector", "GET", "/_studio/connector-store/access"), Input: guard})
		if err != nil {
			return nil, err
		}
		allowed, ok := value.(*access.Output)
		if !ok || allowed == nil || len(allowed.Connectors) != 1 || allowed.Connectors[0] == nil || allowed.Connectors[0].Name != name {
			return nil, publisherguard.PublicError(403, "Studio authorization denied")
		}
	}
	access := &connectorget.ConnectorGetInput{}
	access.SetJwt(claims)
	access.SetAuth(auth)
	access.SetName(name)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[connectorget.ConnectorComponent](), "connector", "POST", "/v1/studio/sdk/connectors.get"), Input: access})
	if err != nil {
		return nil, err
	}
	visible, ok := value.(*connectorget.ConnectorGetOutput)
	if !ok || visible == nil || visible.Item == nil || visible.Item.Name != name {
		return nil, fmt.Errorf("connector access reader returned %T without matching row", value)
	}
	if !probe && visible.Item.Status != "active" {
		return nil, publisherguard.PublicError(400, "connector must be active before browsing schema")
	}
	read := &stored.Input{}
	read.SetName(name)
	read.SetQuery("")
	if probe {
		read.SetStatus("")
	} else {
		read.SetStatus("active")
	}
	read.SetOwnerId("")
	read.SetDriver("")
	read.SetSubject(claims.Subject)
	read.SetScoped(true)
	read.SetPageLimit(2)
	read.SetPageOffset(0)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ConnectorComponent](), "connector", "GET", "/_studio/connector-store/catalog"), Input: read})
	if err != nil {
		return nil, err
	}
	page, ok := value.(*stored.Output)
	if !ok || page == nil || len(page.Connectors) != 1 || page.Connectors[0] == nil {
		return nil, fmt.Errorf("connector private catalog returned %T without one matching row", value)
	}
	row := page.Connectors[0]
	if row.Name != name || row.OwnerId != visible.Item.OwnerId || row.Driver != visible.Item.Driver || row.Status != visible.Item.Status {
		return nil, fmt.Errorf("connector private catalog identity changed")
	}
	connector := &sdk.Connector{Name: row.Name, Driver: row.Driver, OwnerID: row.OwnerId, Status: row.Status,
		ETag: row.Etag, Options: append([]byte(nil), row.OptionsJson...)}
	if row.DsnTemplate != nil {
		connector.DSNTemplate = *row.DsnTemplate
	}
	if row.SecretRef != nil {
		connector.SecretRef = *row.SecretRef
	}
	return connector, nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

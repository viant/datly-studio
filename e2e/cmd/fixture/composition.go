package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	xhandler "github.com/viant/xdatly/handler"
)

type compositionInput struct{}
type compositionOutput struct {
	Count int `json:"count"`
}
type compositionHandler struct{ target dexec.ComponentTarget }

func (h compositionHandler) Exec(ctx context.Context, session xhandler.Session, _ *compositionInput, output *compositionOutput) error {
	value, found, err := session.Binder().Lookup(ctx, dexec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("native component invoker missing")
	}
	result, err := value.(dexec.ComponentInvoker).InvokeComponent(ctx, dexec.ComponentRequest{Target: h.target})
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	var rows struct {
		Rows []json.RawMessage `json:"rows"`
	}
	if err = json.Unmarshal(encoded, &rows); err != nil {
		return err
	}
	output.Count = len(rows.Rows)
	return nil
}

// Resolve only the namespace persisted by this synthetic case. No namespace ID
// exists when the fixture starts; Endly creates it through authoring MCP.
func compositionResolver(db *sql.DB) runtimehost.NamespaceBuildResolver {
	return func(ctx context.Context, id string) (runtimehost.NamespaceBuildExtension, error) {
		var name string
		err := db.QueryRowContext(ctx, "SELECT name FROM namespaces WHERE namespace_id=? AND status='active'", id).Scan(&name)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if name != "composed" {
			return nil, nil
		}
		return func(_ context.Context, build *application.Build) error {
			var target dexec.ComponentTarget
			for _, item := range build.Components {
				for _, route := range item.Component.Routes {
					if route.Path == "/composed" {
						target = dexec.ComponentTarget{Component: item.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
					}
				}
			}
			// Empty namespace startup has no reader yet. The next publication builds
			// the handler together with its reader in the same immutable generation.
			if target.Component.Name == "" {
				return nil
			}
			component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/e2e/composition", Name: "Count"}, Routes: []*spec.Route{{Method: http.MethodGet, Path: "/composed/count", MCP: []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "composition.count", Description: "Count the published synthetic customer rows"}}}}}
			artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[compositionInput](), OutputType: reflect.TypeFor[compositionOutput]()})
			if err != nil {
				return err
			}
			registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[compositionInput, compositionOutput](compositionHandler{target: target})})
			if err != nil {
				return err
			}
			build.Components = append(build.Components, registered)
			// Linked components have no authored source revision. Authorize the
			// wrapper through its exact reader policy, while keeping every
			// ordinary reader on the SDK's source metadata and policy path.
			linked := dexec.ComponentTarget{Component: component.Key, Route: spec.RouteRef{Method: http.MethodGet, Path: "/composed/count"}}
			oldTool, oldCatalog, oldMetadata := build.MCP.AuthorizeTool, build.MCP.AuthorizeCatalogTool, build.MCP.ToolMetadata
			build.MCP.AuthorizeTool = func(ctx context.Context, candidate dexec.ComponentTarget) error {
				if candidate == linked {
					candidate = target
				}
				return oldTool(ctx, candidate)
			}
			build.MCP.AuthorizeCatalogTool = func(ctx context.Context, candidate dexec.ComponentTarget, action string) error {
				if candidate == linked {
					candidate = target
				}
				return oldCatalog(ctx, candidate, action)
			}
			build.MCP.ToolMetadata = func(ctx context.Context, candidate dexec.ComponentTarget) (map[string]interface{}, error) {
				if candidate == linked {
					return nil, nil
				}
				return oldMetadata(ctx, candidate)
			}
			return nil
		}, nil
	}
}

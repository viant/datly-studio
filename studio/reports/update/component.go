package update

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespacevalidation"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	connectors "github.com/viant/datly-studio/studio/connectors/get"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	versions "github.com/viant/datly-studio/studio/report_versions/store_head"
	reportcreate "github.com/viant/datly-studio/studio/reports/create"
	guard "github.com/viant/datly-studio/studio/reports/edit_guard"
	reports "github.com/viant/datly-studio/studio/reports/get"
	stored "github.com/viant/datly-studio/studio/reports/store_config"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Options struct {
	Namespace            *string `json:"namespace,omitempty"`
	Slug                 *string `json:"slug,omitempty"`
	Title                *string `json:"title,omitempty"`
	Description          *string `json:"description,omitempty"`
	Status               *string `json:"status,omitempty"`
	DefaultConnectorName *string `json:"defaultConnectorName,omitempty"`
	ComponentScope       *string `json:"componentScope,omitempty"`
	ComponentName        *string `json:"componentName,omitempty"`
	CurrentDraftVersion  *int    `json:"currentDraftVersion,omitempty"`
	ETag                 int64   `json:"etag"`
}

type Input struct {
	Jwt   *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Id    string             `parameter:"Id,kind=body,in=id,dataType=string,required=true" json:"id"`
	Input Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

type Output = reportcreate.Output

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"report,path=/v1/studio/sdk/components.update,method=POST,connector=studio,handler=NewReportUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.components.update\",\"description\":\"Update an authorized Datly Studio report at an expected revision\"}]" caseFormat:"lc"`
}

var ReportDatly = new(Component)
var ReportDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReportUpdate" {
		return custom.Factory(NewReportUpdate)
	}
	return nil
}

type updateHandler struct{}

func NewReportUpdate() xhandler.Contract[Input, Output] { return &updateHandler{} }

func (*updateHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return publicError(403, "verified Studio principal is required")
	}
	if input.Id == "" || input.Input.ETag <= 0 {
		return publicError(400, "id and positive etag are required")
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("report update handler session and output are required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	access := &guard.Input{}
	access.SetJwt(input.Jwt)
	access.SetAuth(input.Auth)
	access.SetReportId(input.Id)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[guard.ReportComponent](), "report", "POST", "/_studio/reports/edit-guard"), Input: access})
	if err != nil {
		return err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil {
		return fmt.Errorf("report edit guard returned %T", value)
	}
	if allowed.Item == nil || allowed.Item.Id != input.Id {
		return publicError(403, "Studio authorization denied")
	}
	read := &reports.ReportGetInput{}
	read.SetJwt(input.Jwt)
	read.SetAuth(input.Auth)
	read.SetId(input.Id)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[reports.ReportComponent](), "report", "POST", "/v1/studio/sdk/components.get"), Input: read})
	if err != nil {
		return err
	}
	result, ok := value.(*reports.ReportGetOutput)
	if !ok || result == nil {
		return fmt.Errorf("report reader returned %T", value)
	}
	if result.Item == nil || result.Item.Id != input.Id || result.Item.OwnerId == "" {
		return publicError(404, "report not found")
	}
	current := *result.Item
	if current.Etag != input.Input.ETag {
		return publicError(409, "report etag does not match")
	}
	if input.Input.ComponentScope != nil || input.Input.ComponentName != nil {
		return publicError(403, "component package identity is server managed")
	}
	if input.Input.Slug != nil {
		current.Slug = *input.Input.Slug
	}
	if input.Input.Namespace != nil {
		if !namespacevalidation.ValidName(*input.Input.Namespace) {
			return publicError(400, "namespace must use lowercase letters, numbers, underscores, and optional dot-separated segments")
		}
		current.Namespace = *input.Input.Namespace
	}
	if input.Input.Title != nil {
		current.Title = *input.Input.Title
	}
	if input.Input.Description != nil {
		current.Description = input.Input.Description
	}
	if input.Input.Status != nil {
		current.Status = *input.Input.Status
	}
	if input.Input.DefaultConnectorName != nil {
		requested := strings.TrimSpace(*input.Input.DefaultConnectorName)
		if requested == "" {
			return publicError(400, "default connector name is required")
		}
		if requested != current.DefaultConnectorName {
			connectorInput := &connectors.ConnectorGetInput{}
			connectorInput.SetJwt(input.Jwt)
			connectorInput.SetAuth(input.Auth)
			connectorInput.SetName(requested)
			value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[connectors.ConnectorComponent](), "connector", "POST", "/v1/studio/sdk/connectors.get"), Input: connectorInput})
			if err != nil {
				return err
			}
			connector, ok := value.(*connectors.ConnectorGetOutput)
			if !ok || connector == nil {
				return fmt.Errorf("connector reader returned %T", value)
			}
			if connector.Item == nil || connector.Item.Name != requested {
				return publicError(404, "connector not found")
			}
			if connector.Item.Status != "active" {
				return publicError(400, "default connector must be active")
			}
			headInput := &versions.Input{}
			headInput.SetReportId(input.Id)
			value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[versions.HeadComponent](), "head", "GET", "/_studio/report-version-store/head"), Input: headInput})
			if err != nil {
				return err
			}
			head, ok := value.(*versions.Output)
			if !ok || head == nil || len(head.Heads) != 1 || head.Heads[0] == nil {
				return fmt.Errorf("version head returned %T without one total", value)
			}
			if head.Heads[0].MaxVersionNo > 0 {
				return publicError(409, "default connector is part of the versioned reader contract; change it through Reader Builder and create a validated version")
			}
			current.DefaultConnectorName = requested
		}
	}
	if input.Input.CurrentDraftVersion != nil {
		current.CurrentDraftVersion = input.Input.CurrentDraftVersion
	}
	if err := requireActiveNamespace(ctx, invoker, current.OwnerId, current.Namespace); err != nil {
		return err
	}
	now, etag := time.Now().UTC(), input.Input.ETag
	row := &stored.StoredReport{Id: input.Id, Namespace: current.Namespace, Slug: current.Slug,
		Title: current.Title, Description: current.Description, OwnerId: current.OwnerId,
		Status: current.Status, DefaultConnectorName: current.DefaultConnectorName,
		ComponentScope: current.ComponentScope, ComponentName: current.ComponentName,
		CurrentDraftVersion: current.CurrentDraftVersion, Etag: &etag, UpdatedAt: &now,
		Has: &stored.StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true,
			Description: true, OwnerId: true, Status: true, DefaultConnectorName: true,
			ComponentScope: true, ComponentName: true, CurrentDraftVersion: true,
			Etag: true, UpdatedAt: true}}
	write := &stored.Input{}
	write.SetReports([]*stored.StoredReport{row})
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ReportComponent](), "report", "PATCH", "/_studio/report-store/config"), Input: write})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publicError(409, "report etag does not match")
		}
		return err
	}
	updated, ok := value.(*stored.Output)
	if !ok || updated == nil || len(updated.Data) != 1 || updated.Data[0] == nil ||
		updated.Data[0].Id != input.Id || updated.Data[0].Etag == nil {
		return fmt.Errorf("report config writer returned %T without one updated row", value)
	}
	*output = Output{Id: input.Id, Namespace: current.Namespace, Slug: current.Slug,
		Title: current.Title, OwnerId: current.OwnerId,
		OwnerPackage: sdk.OwnerPackageSegment(current.OwnerId), Status: current.Status,
		DefaultConnectorName: current.DefaultConnectorName,
		ComponentScope:       current.ComponentScope, ComponentName: current.ComponentName,
		CurrentDraftVersion: current.CurrentDraftVersion, Etag: *updated.Data[0].Etag,
		CreatedAt: current.CreatedAt, UpdatedAt: now}
	if current.Description != nil {
		output.Description = *current.Description
	}
	return nil
}

func requireActiveNamespace(ctx context.Context, invoker exec.ComponentInvoker, owner, name string) error {
	read := &namespaces.Input{}
	read.SetOwnerId(owner)
	read.SetName(name)
	read.SetQuery("")
	read.SetStatus("")
	read.SetSubject("")
	read.SetScoped(false)
	read.SetPageLimit(2)
	read.SetPageOffset(0)
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: read})
	if err != nil {
		return err
	}
	result, ok := value.(*namespaces.Output)
	if !ok || result == nil {
		return fmt.Errorf("owned namespace reader returned %T", value)
	}
	if len(result.Namespaces) == 0 {
		return publicError(404, "namespace not found")
	}
	if len(result.Namespaces) != 1 || result.Namespaces[0] == nil ||
		result.Namespaces[0].OwnerId != owner || result.Namespaces[0].Name != name {
		return fmt.Errorf("owned namespace reader returned ambiguous or mismatched rows")
	}
	if result.Namespaces[0].Status != "active" {
		return publicError(400, "namespace must be active")
	}
	return nil
}

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func publicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}

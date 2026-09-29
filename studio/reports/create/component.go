package create

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespacevalidation"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	connectors "github.com/viant/datly-studio/studio/connectors/get"
	namespacecreate "github.com/viant/datly-studio/studio/namespaces/create"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	stored "github.com/viant/datly-studio/studio/reports/store_insert"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

type Input struct {
	NamespaceId          *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt                  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth                 *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Id                   string             `parameter:"Id,kind=body,in=id,dataType=string" json:"id,omitempty"`
	Slug                 string             `parameter:"Slug,kind=body,in=slug,dataType=string,required=true" json:"slug"`
	Namespace            string             `parameter:"Namespace,kind=body,in=namespace,dataType=string" json:"namespace,omitempty"`
	Title                string             `parameter:"Title,kind=body,in=title,dataType=string,required=true" json:"title"`
	Description          string             `parameter:"Description,kind=body,in=description,dataType=string" json:"description,omitempty"`
	OwnerId              string             `parameter:"OwnerId,kind=body,in=ownerId,dataType=string" json:"ownerId,omitempty"`
	DefaultConnectorName string             `parameter:"DefaultConnectorName,kind=body,in=defaultConnectorName,dataType=string,required=true" json:"defaultConnectorName"`
	ComponentScope       string             `parameter:"ComponentScope,kind=body,in=componentScope,dataType=string" json:"componentScope,omitempty"`
	ComponentName        string             `parameter:"ComponentName,kind=body,in=componentName,dataType=string" json:"componentName,omitempty"`
}

type Output struct {
	Id                   string    `parameter:"Id,kind=output,in=body,dataType=string" json:"id"`
	Namespace            string    `parameter:"Namespace,kind=output,in=body,dataType=string" json:"namespace"`
	Slug                 string    `parameter:"Slug,kind=output,in=body,dataType=string" json:"slug"`
	Title                string    `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description          string    `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	OwnerId              string    `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	OwnerPackage         string    `parameter:"OwnerPackage,kind=output,in=body,dataType=string" json:"ownerPackage"`
	Status               string    `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	DefaultConnectorName string    `parameter:"DefaultConnectorName,kind=output,in=body,dataType=string" json:"defaultConnectorName"`
	ComponentScope       string    `parameter:"ComponentScope,kind=output,in=body,dataType=string" json:"componentScope"`
	ComponentName        string    `parameter:"ComponentName,kind=output,in=body,dataType=string" json:"componentName"`
	CurrentDraftVersion  *int      `parameter:"CurrentDraftVersion,kind=output,in=body,dataType=*int" json:"currentDraftVersion,omitempty"`
	Etag                 int64     `parameter:"Etag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt            time.Time `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt            time.Time `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"report,path=/v1/studio/sdk/components.create,method=POST,connector=studio,handler=NewReportCreate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.components.create\",\"description\":\"Create a Datly Studio report under the verified owner\"}]" caseFormat:"lc"`
}

var ReportDatly = new(Component)
var ReportDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewReportCreate" {
		return custom.Factory(NewReportCreate)
	}
	return nil
}

type createHandler struct{}

func NewReportCreate() xhandler.Contract[Input, Output] { return &createHandler{} }

func (*createHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || input.Jwt == nil || input.Auth == nil || input.Auth.Auth == nil ||
		input.Jwt.Subject == "" || input.Auth.Auth.Subject != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("verified Studio owner is required")}
	}
	if session == nil || session.Binder() == nil || output == nil {
		return fmt.Errorf("report create handler session and output are required")
	}
	input.Slug, input.Title, input.DefaultConnectorName = strings.TrimSpace(input.Slug), strings.TrimSpace(input.Title), strings.TrimSpace(input.DefaultConnectorName)
	if !validSlug(input.Slug) || input.Title == "" || input.DefaultConnectorName == "" {
		return &xresponse.Error{Code: 400, Cause: errors.New("slug, title, and defaultConnectorName are required; slug must use lowercase letters, numbers, and dashes")}
	}
	namespace := strings.TrimSpace(input.Namespace)
	if namespace == "" {
		namespace = "general"
	}
	if !namespacevalidation.ValidName(namespace) {
		return &xresponse.Error{Code: 400, Cause: errors.New("namespace must use lowercase letters, numbers, underscores, and optional dot-separated segments")}
	}
	if owner := strings.TrimSpace(input.OwnerId); owner != "" && owner != input.Jwt.Subject {
		return &xresponse.Error{Code: 403, Cause: errors.New("cannot create another principal's report")}
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("Datly component invoker is unavailable")
	}
	if input.NamespaceId != nil {
		if err := namespaceaccess.ValidateResourceOwnership(*input.NamespaceId, ""); err != nil {
			return &xresponse.Error{Code: 400, Cause: errors.New("valid namespace selection is required")}
		}
		selected := &namespaces.Input{}
		selected.NamespaceId = *input.NamespaceId
		selected.SetOwnerId(input.Jwt.Subject)
		selected.SetStatus("active")
		selected.SetScoped(false)
		selected.SetPageLimit(2)
		selected.SetPageOffset(0)
		selected.Has.NamespaceId = true
		value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: selected})
		if err != nil {
			return err
		}
		page, ok := value.(*namespaces.Output)
		if !ok || page == nil || len(page.Namespaces) != 1 || page.Namespaces[0] == nil || page.Namespaces[0].NamespaceId != *input.NamespaceId || page.Namespaces[0].OwnerId != input.Jwt.Subject {
			return &xresponse.Error{Code: 403, Cause: errors.New("selected namespace is unavailable")}
		}
		if input.Namespace != "" && input.Namespace != page.Namespaces[0].Name {
			return &xresponse.Error{Code: 403, Cause: errors.New("component creation is outside the selected namespace")}
		}
		namespace = page.Namespaces[0].Name
	}
	connectorInput := &connectors.ConnectorGetInput{}
	connectorInput.SetJwt(input.Jwt)
	connectorInput.SetAuth(input.Auth)
	connectorInput.SetName(input.DefaultConnectorName)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[connectors.ConnectorComponent](), "connector", "POST", "/v1/studio/sdk/connectors.get"), Input: connectorInput})
	if err != nil {
		return err
	}
	connector, ok := value.(*connectors.ConnectorGetOutput)
	if !ok || connector == nil {
		return fmt.Errorf("connector reader returned %T", value)
	}
	if connector.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("connector not found")}
	}
	if connector.Item.Name != input.DefaultConnectorName || connector.Item.OwnerId == "" {
		return fmt.Errorf("connector reader returned mismatched identity")
	}
	if connector.Item.Status != "active" {
		return &xresponse.Error{Code: 400, Cause: errors.New("default connector must be active")}
	}
	if err := requireNamespace(ctx, invoker, input, namespace); err != nil {
		return err
	}
	id := strings.TrimSpace(input.Id)
	if id == "" {
		data := make([]byte, 10)
		if _, err := rand.Read(data); err != nil {
			return err
		}
		id = "r" + hex.EncodeToString(data)
	}
	now := time.Now().UTC()
	row := &stored.StoredReport{Id: id, Namespace: namespace, Slug: input.Slug, Title: input.Title,
		OwnerId: input.Jwt.Subject, Status: "draft", DefaultConnectorName: input.DefaultConnectorName,
		ComponentScope: "github.com/viant/datly-studio/dynamic/" + sdk.OwnerPackageSegment(input.Jwt.Subject) + "/" + id,
		ComponentName:  "reader", Etag: 1, CreatedAt: now, UpdatedAt: now,
		Has: &stored.StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true,
			Description: true, OwnerId: true, Status: true, DefaultConnectorName: true,
			ComponentScope: true, ComponentName: true, Etag: true, CreatedAt: true, UpdatedAt: true}}
	if description := strings.TrimSpace(input.Description); description != "" {
		row.Description = &description
	}
	row.SetNamespaceId(namespaceaccess.ID(row.OwnerId, row.Namespace))
	write := &stored.Input{}
	write.SetReports([]*stored.StoredReport{row})
	written, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[stored.ReportComponent](), "report", "POST", "/_studio/report-store/insert"), Input: write})
	if err != nil {
		return err
	}
	mutation, ok := written.(*stored.Output)
	if !ok || mutation == nil || len(mutation.Data) != 1 {
		return fmt.Errorf("report insert returned %T without one inserted row: %+v", written, written)
	}
	report := mutation.Data[0]
	if report == nil || report.Id != id || report.OwnerId != input.Jwt.Subject {
		return fmt.Errorf("report insert returned a mismatched row")
	}
	*output = Output{Id: report.Id, Namespace: report.Namespace, Slug: report.Slug,
		Title: report.Title, OwnerId: report.OwnerId,
		OwnerPackage: sdk.OwnerPackageSegment(report.OwnerId), Status: report.Status,
		DefaultConnectorName: report.DefaultConnectorName, ComponentScope: report.ComponentScope,
		ComponentName: report.ComponentName,
		Etag:          report.Etag, CreatedAt: report.CreatedAt, UpdatedAt: report.UpdatedAt}
	if report.Description != nil {
		output.Description = *report.Description
	}
	return nil
}

func requireNamespace(ctx context.Context, invoker exec.ComponentInvoker, input *Input, name string) error {
	read := func() (bool, error) {
		request := &namespaces.Input{}
		request.SetOwnerId(input.Jwt.Subject)
		request.SetName(name)
		request.SetQuery("")
		request.SetStatus("")
		request.SetSubject("")
		request.SetScoped(false)
		request.SetPageLimit(2)
		request.SetPageOffset(0)
		value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespaces.NamespaceComponent](), "namespace", "GET", "/_studio/namespace-store/read"), Input: request})
		if err != nil {
			return false, err
		}
		result, ok := value.(*namespaces.Output)
		if !ok || result == nil {
			return false, fmt.Errorf("owned namespace reader returned %T", value)
		}
		if len(result.Namespaces) == 0 {
			return false, nil
		}
		if len(result.Namespaces) != 1 || result.Namespaces[0] == nil || result.Namespaces[0].OwnerId != input.Jwt.Subject || result.Namespaces[0].Name != name {
			return false, fmt.Errorf("owned namespace reader returned ambiguous or mismatched rows")
		}
		if result.Namespaces[0].Status != "active" {
			return false, &xresponse.Error{Code: 400, Cause: errors.New("namespace must be active")}
		}
		return true, nil
	}
	found, err := read()
	if err != nil || found {
		return err
	}
	if name != "general" {
		return &xresponse.Error{Code: 404, Cause: errors.New("namespace not found")}
	}
	created := &namespacecreate.NamespaceCreateInput{}
	created.SetJwt(input.Jwt)
	created.SetAuth(input.Auth)
	created.SetNamespace(&namespacecreate.NamespaceRecord{Name: "general", Title: "General", Description: pointer("Default namespace")})
	createdValue, createErr := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target(reflect.TypeFor[namespacecreate.NamespaceComponent](), "namespace", "POST", "/v1/studio/sdk/namespaces.create"), Input: created})
	if createErr == nil {
		result, ok := createdValue.(*namespacecreate.NamespaceCreateOutput)
		if !ok || result == nil || result.Data == nil || result.Data.OwnerId != input.Jwt.Subject ||
			result.Data.Name != "general" || result.Data.Status != "active" {
			return fmt.Errorf("default namespace writer returned %T with mismatched identity", createdValue)
		}
		return nil
	}
	// A concurrent creator may have inserted the same default namespace. The
	// authoritative owned read distinguishes that race from a failed insert.
	found, err = read()
	if err == nil && found {
		return nil
	}
	return createErr
}

func pointer(value string) *string { return &value }

func target(holder reflect.Type, name, method, path string) exec.ComponentTarget {
	return exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: path}}
}

func validSlug(value string) bool {
	if value == "" || len(value) > 200 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, character := range value {
		if character != '-' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

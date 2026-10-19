package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

const resourceFilterKey = "io.viant/resources/filter"

type ResourcesListInput struct {
	Jwt     *jwt.Claims                       `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output                `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Request schema.ListResourcesRequestParams `parameter:"Request,kind=body,in=,dataType=schema.ListResourcesRequestParams,required=false" anonymous:"true"`
}
type ResourcesListOutput struct {
	Response *schema.ListResourcesResult `parameter:"Response,kind=output,in=body,dataType=*schema.ListResourcesResult" json:"-"`
}

func (*ResourcesListOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[schema.ListResourcesResult]()
}
func (o *ResourcesListOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.Response) }

type ResourcesListComponent struct {
	Contract xdatly.Component[ResourcesListInput, ResourcesListOutput] `component:"resources_list,path=/v1/studio/catalog/resources/list,method=POST,connector=studio,handler=NewResourcesList" mcp:"[{\"kind\":\"tool\",\"name\":\"resources/list\",\"description\":\"List authorized resource headers with native paging and scoped metadata filters\"}]" caseFormat:"lc"`
}

var ResourcesListDatlyLinkedType = reflect.TypeFor[ResourcesListComponent]()

func (ResourcesListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewResourcesList" {
		return custom.Factory(NewResourcesList)
	}
	return nil
}
func NewResourcesList() xhandler.Contract[ResourcesListInput, ResourcesListOutput] {
	return &handler[ResourcesListInput, ResourcesListOutput]{run: runResourcesList}
}

type resourceFilter struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Query     string `json:"query"`
	Limit     int    `json:"limit"`
}
type resourceCursor struct {
	Fingerprint string `json:"f"`
	Management  int    `json:"m"`
	Offset      int    `json:"o"`
}

func runResourcesList(ctx context.Context, session xhandler.Session, input *ResourcesListInput, output *ResourcesListOutput) error {
	if input == nil || output == nil {
		return fail(400, "resource listing input/output required")
	}
	admin, definitions, err := management(input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	filter := resourceFilter{}
	metaBytes, _ := json.Marshal(input.Request.Meta)
	meta := map[string]any{}
	_ = json.Unmarshal(metaBytes, &meta)
	if raw, ok := meta[resourceFilterKey]; ok {
		bytes, e := json.Marshal(raw)
		if e != nil {
			return fail(400, "invalid resource filter")
		}
		decoder := json.NewDecoder(strings.NewReader(string(bytes)))
		decoder.DisallowUnknownFields()
		if string(bytes) == "null" || decoder.Decode(&filter) != nil {
			return fail(400, "invalid resource filter")
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 32
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return fail(400, "resource limit must be between 1 and 100")
	}
	if strings.HasPrefix(filter.Namespace, "ns://") {
		filter.Namespace, err = ParseNamespaceURI(filter.Namespace)
		if err != nil {
			return err
		}
	}
	canonical, _ := json.Marshal(struct {
		Filter  resourceFilter
		Subject string
	}{filter, input.Jwt.Subject})
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(canonical))
	cursor := resourceCursor{Fingerprint: fingerprint}
	if input.Request.Cursor != nil {
		raw, e := base64.RawURLEncoding.DecodeString(*input.Request.Cursor)
		if e != nil || json.Unmarshal(raw, &cursor) != nil || cursor.Fingerprint != fingerprint || cursor.Offset < 0 || cursor.Management < 0 {
			return fail(400, "invalid resource cursor")
		}
	}
	resources := []schema.Resource{}
	managementEntries := []Reference{}
	if admin != nil && (filter.Namespace == "" || filter.Namespace == admin.Name) {
		for _, ref := range managementReferences(definitions, filter.Kind) {
			if filter.Query == "" || strings.Contains(strings.ToLower(ref.Name+" "+ref.Description+" "+ref.URI), strings.ToLower(filter.Query)) {
				managementEntries = append(managementEntries, ref)
			}
		}
	}
	if cursor.Management > len(managementEntries) {
		return fail(400, "invalid resource cursor")
	}
	for cursor.Management < len(managementEntries) && len(resources) < filter.Limit {
		ref := managementEntries[cursor.Management]
		mime := ref.MimeType
		resources = append(resources, schema.Resource{Name: ref.Name, Uri: ref.URI, MimeType: &mime})
		cursor.Management++
	}
	more := cursor.Management < len(managementEntries)
	if !more && (filter.Kind == "" || filter.Kind == "mcp") && (admin == nil || filter.Namespace != admin.Name) {
		remaining := filter.Limit - len(resources)
		id, name := "", ""
		if filter.Namespace != "" {
			if len(filter.Namespace) == 64 {
				id = filter.Namespace
			} else {
				name = filter.Namespace
			}
		}
		rows, e := directoryPage(ctx, session, input.Jwt, input.Auth, id, name, filter.Query, true, remaining+1, cursor.Offset)
		if e != nil {
			return e
		}
		more = len(rows) > remaining
		if len(rows) > remaining {
			rows = rows[:remaining]
		}
		for _, row := range rows {
			mime := "application/json"
			resources = append(resources, schema.Resource{Name: row.Name, Uri: VirtualURI(row.NamespaceId, row.Name), MimeType: &mime})
			cursor.Offset++
		}
	}
	output.Response = &schema.ListResourcesResult{Resources: resources, CacheScope: schema.ListResourcesResultCacheScopePrivate}
	if more {
		raw, _ := json.Marshal(cursor)
		next := base64.RawURLEncoding.EncodeToString(raw)
		output.Response.NextCursor = &next
	}
	return nil
}

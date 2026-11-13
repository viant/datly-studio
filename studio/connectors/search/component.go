package search

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/datly-studio/internal/connectorcatalog"
	"github.com/viant/datly-studio/internal/providererror"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt   *jwt.Claims           `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth  *studioauth.Output    `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name  string                `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	Input sdk.SchemaSearchInput `parameter:"Input,kind=body,in=input,dataType=sdk.SchemaSearchInput,required=false" json:"input"`
}

type Output struct {
	Items              []sdk.SchemaSearchMatch `parameter:"Items,kind=output,in=body,dataType=[]sdk.SchemaSearchMatch" json:"items"`
	NextCursor         string                  `parameter:"NextCursor,kind=output,in=body,dataType=string" json:"nextCursor,omitempty"`
	NextMetadataCursor string                  `parameter:"NextMetadataCursor,kind=output,in=body,dataType=string" json:"nextMetadataCursor,omitempty"`
	MetadataComplete   bool                    `parameter:"MetadataComplete,kind=output,in=body,dataType=bool" json:"metadataComplete"`
	SnapshotID         string                  `parameter:"SnapshotID,kind=output,in=body,dataType=string" json:"snapshotId"`
	ExpiresAt          time.Time               `parameter:"ExpiresAt,kind=output,in=body,dataType=time.Time" json:"expiresAt"`
	Total              int                     `parameter:"Total,kind=output,in=body,dataType=int" json:"total"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"connector,path=/v1/studio/sdk/connectors.search,method=POST,connector=studio,handler=NewSearch" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.connectors.search\",\"description\":\"Search compact table and column metadata in an explicit database or dataset on an authorized connector; reuse cached metadata, follow metadata continuation separately, and inspect selected tables for full schemas\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewSearch" {
		return custom.Factory(NewSearch)
	}
	return nil
}

type handler struct{}

func NewSearch() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("connector search input and output are required")
	}
	connector, err := connectorcatalog.Authorized(ctx, session, input.Jwt, input.Auth, input.Name)
	if err != nil {
		return err
	}
	searcher, err := host.SchemaSearchFromContext(ctx)
	if err != nil {
		return publisherguard.PublicError(503, "host schema search is unavailable", err)
	}
	result, err := searcher.Search(ctx, connector, sdk.Principal{Subject: input.Jwt.Subject}, input.Input)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return publisherguard.PublicError(502, providererror.AppendCatalogStatus("connector schema search is unavailable", err), err)
	}
	if result == nil {
		return fmt.Errorf("connector table catalog returned no result")
	}
	output.Items, output.Total = result.Items, result.Total
	output.NextCursor, output.NextMetadataCursor = result.NextCursor, result.NextMetadataCursor
	output.MetadataComplete, output.SnapshotID, output.ExpiresAt = result.MetadataComplete, result.SnapshotID, result.ExpiresAt
	return nil
}

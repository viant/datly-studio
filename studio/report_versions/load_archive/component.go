package load_archive

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	load "github.com/viant/datly-studio/studio/report_versions/load_dql"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Options struct {
	Archive  []byte `json:"archive"`
	Format   string `json:"format"`
	EntryDQL string `json:"entryDql,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type Input struct {
	NamespaceId *string            `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportId    string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input       Options            `parameter:"Input,kind=body,in=input,dataType=Options,required=true" json:"input"`
}

type Output = load.Output

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.load_archive,method=POST,connector=studio,handler=NewLoadArchive" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.load_archive\",\"description\":\"Import an authorized Datly Studio component archive as a draft version\"}]" caseFormat:"lc"`
}

var VersionDatly = new(Component)
var VersionDatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewLoadArchive" {
		return custom.Factory(NewLoadArchive)
	}
	return nil
}

type loadHandler struct{}

func NewLoadArchive() xhandler.Contract[Input, Output] { return &loadHandler{} }

func (*loadHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	request := &load.Input{}
	if input != nil {
		request.Jwt, request.Auth, request.ReportId = input.Jwt, input.Auth, input.ReportId
		request.NamespaceId = input.NamespaceId
		request.Input.Notes = input.Input.Notes
	}
	return load.ImportBundle(ctx, session, request, output, func() (*sdk.DQLBundle, string, error) {
		bundle, err := sdk.ReadDQLArchive(bytes.NewReader(input.Input.Archive), input.Input.Format)
		if err != nil {
			return nil, "", err
		}
		entry := input.Input.EntryDQL
		if entry == "" && len(bundle.Entries) == 1 {
			entry = bundle.Entries[0]
		}
		for _, candidate := range bundle.Entries {
			if candidate == entry {
				return bundle, entry, nil
			}
		}
		return nil, "", fmt.Errorf("entryDql must select a root DQL document: %v", bundle.Entries)
	})
}

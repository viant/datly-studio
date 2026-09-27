package test_relation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/internal/versionrun"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt       *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	ReportID  string             `parameter:"ReportID,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	VersionNo int                `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo"`
	Relation  string             `parameter:"Relation,kind=body,in=relation,dataType=string,required=true" json:"relation"`
	Input     sdk.ViewTestInput  `parameter:"Input,kind=body,in=input,dataType=sdk.ViewTestInput,required=false" json:"input"`
}

type Output struct {
	Relation         string                    `parameter:"Relation,kind=output,in=body,dataType=string" json:"relation"`
	ParentView       string                    `parameter:"ParentView,kind=output,in=body,dataType=string" json:"parentView"`
	ChildView        string                    `parameter:"ChildView,kind=output,in=body,dataType=string" json:"childView"`
	Kind             string                    `parameter:"Kind,kind=output,in=body,dataType=string" json:"kind"`
	Cardinality      string                    `parameter:"Cardinality,kind=output,in=body,dataType=string" json:"cardinality"`
	Keys             []sdk.RelationKeyEvidence `parameter:"Keys,kind=output,in=body,dataType=[]sdk.RelationKeyEvidence" json:"keys,omitempty"`
	ParentRows       int                       `parameter:"ParentRows,kind=output,in=body,dataType=int" json:"parentRows"`
	MatchedParents   int                       `parameter:"MatchedParents,kind=output,in=body,dataType=int" json:"matchedParents"`
	UnmatchedParents int                       `parameter:"UnmatchedParents,kind=output,in=body,dataType=int" json:"unmatchedParents"`
	AttachedChildren int                       `parameter:"AttachedChildren,kind=output,in=body,dataType=int" json:"attachedChildren"`
	Data             json.RawMessage           `parameter:"Data,kind=output,in=body,dataType=json.RawMessage" json:"data,omitempty"`
	Duration         time.Duration             `parameter:"Duration,kind=output,in=body,dataType=time.Duration" json:"duration"`
	Evidence         sdk.ExecutionEvidence     `parameter:"Evidence,kind=output,in=body,dataType=sdk.ExecutionEvidence" json:"evidence"`
	Diagnostics      []sdk.Diagnostic          `parameter:"Diagnostics,kind=output,in=body,dataType=[]sdk.Diagnostic" json:"diagnostics,omitempty"`
}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"version,path=/v1/studio/sdk/versions.test_relation,method=POST,connector=studio,handler=NewTestRelation" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.versions.test_relation\",\"description\":\"Measure one relation of an authorized exact Datly reader version\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewTestRelation" {
		return custom.Factory(NewTestRelation)
	}
	return nil
}

type handler struct{}

func NewTestRelation() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("relation test input and output are required")
	}
	if strings.TrimSpace(input.Relation) == "" {
		return publisherguard.PublicError(400, "relation is required")
	}
	executionCtx, engine, cancel, err := versionrun.Authorized(ctx, session, input.Jwt, input.Auth, input.ReportID, input.VersionNo, 30*time.Second)
	if err != nil {
		return err
	}
	defer cancel()
	result, err := engine.TestRelation(executionCtx, input.ReportID, input.VersionNo, input.Relation, input.Input)
	if err != nil {
		return versionrun.PublicError(err, "relation test")
	}
	if result == nil || result.Evidence.ReportID != input.ReportID || result.Evidence.VersionNo != input.VersionNo {
		return fmt.Errorf("relation test returned mismatched execution evidence")
	}
	*output = Output{Relation: result.Relation, ParentView: result.ParentView, ChildView: result.ChildView,
		Kind: result.Kind, Cardinality: result.Cardinality, Keys: result.Keys,
		ParentRows: result.ParentRows, MatchedParents: result.MatchedParents,
		UnmatchedParents: result.UnmatchedParents, AttachedChildren: result.AttachedChildren,
		Data: append(json.RawMessage(nil), result.Data...), Duration: result.Duration,
		Evidence: result.Evidence, Diagnostics: result.Diagnostics}
	return nil
}

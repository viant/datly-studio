package delete

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/predicateprojection"
	"github.com/viant/datly-studio/internal/publisherguard"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/authorization_predicates/store_write"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name string             `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name"`
	ETag int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}

type Output struct{}

type Component struct {
	Contract xdatly.Component[Input, Output] `component:"authorization_predicate,path=/v1/studio/sdk/authorization_predicates.delete,method=POST,connector=studio,handler=NewDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.authorization_predicates.delete\",\"description\":\"Soft-delete an authorized predicate link with an exact ETag\"}]" caseFormat:"lc"`
}

var Datly = new(Component)
var DatlyLinkedType = reflect.TypeFor[Component]()

func (Component) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDelete" {
		return custom.Factory(NewDelete)
	}
	return nil
}

type handler struct{}

func NewDelete() xhandler.Contract[Input, Output] { return &handler{} }

func (*handler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if input == nil || output == nil {
		return fmt.Errorf("authorization predicate delete input and output are required")
	}
	invoker, err := publisherguard.AuthorizedInvoker(ctx, session, input.Jwt, input.Auth)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.Name) == "" || input.ETag <= 0 || input.ETag > math.MaxInt {
		return publisherguard.PublicError(409, "authorization predicate etag does not match")
	}
	rows, err := predicateprojection.Read(ctx, invoker, input.Name, "", "", 1, 0)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return publisherguard.PublicError(409, "authorization predicate etag does not match")
	}
	if len(rows) != 1 || rows[0] == nil || rows[0].Name != input.Name {
		return fmt.Errorf("authorization predicate reader returned ambiguous identity")
	}
	if rows[0].Etag != input.ETag {
		return publisherguard.PublicError(409, "authorization predicate etag does not match")
	}
	etag, now := int(input.ETag), time.Now().UTC()
	row := &stored.StoredAuthorizationPredicate{Name: input.Name, Status: "disabled", DeletedAt: &now, UpdatedAt: &now, Etag: &etag,
		Has: &stored.StoredAuthorizationPredicateHas{Name: true, Status: true, DeletedAt: true, UpdatedAt: true, Etag: true}}
	if err = predicateprojection.Write(ctx, invoker, row); err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return publisherguard.PublicError(409, "authorization predicate etag does not match")
		}
		return err
	}
	if session == nil || session.Response() == nil {
		return fmt.Errorf("authorization predicate response is unavailable")
	}
	session.Response().SetStatusCode(http.StatusNoContent)
	return nil
}

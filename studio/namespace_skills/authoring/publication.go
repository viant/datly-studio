package authoring

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type PublishInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string             `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	ETag        int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
	Revision    int                `parameter:"Revision,kind=body,in=revision,dataType=int,required=true" json:"revision"`
}
type UnpublishInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string             `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	ETag        int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}
type PublicationOutput struct {
	Skill             *sdk.NamespaceSkill `parameter:"Skill,kind=output,in=body,dataType=*sdk.NamespaceSkill" json:"skill"`
	PublicationStatus string              `parameter:"PublicationStatus,kind=output,in=body,dataType=string" json:"publicationStatus"`
}
type PublishComponent struct {
	Contract xdatly.Component[PublishInput, PublicationOutput] `component:"namespace_skill_publish,path=/v1/studio/sdk/namespace_skills.publish,method=POST,handler=NewPublish" independentChildTransactions:"true" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.publish\",\"description\":\"Publish an immutable namespace skill revision and reload its MCP listener\"}]" caseFormat:"lc"`
}
type UnpublishComponent struct {
	Contract xdatly.Component[UnpublishInput, PublicationOutput] `component:"namespace_skill_unpublish,path=/v1/studio/sdk/namespace_skills.unpublish,method=POST,handler=NewUnpublish" independentChildTransactions:"true" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.unpublish\",\"description\":\"Unpublish an independent namespace skill and reload its MCP listener\"}]" caseFormat:"lc"`
}

var PublishDatly = new(PublishComponent)
var UnpublishDatly = new(UnpublishComponent)
var PublishDatlyLinkedType = reflect.TypeFor[PublishComponent]()
var UnpublishDatlyLinkedType = reflect.TypeFor[UnpublishComponent]()

func (PublishComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewPublish" {
		return custom.Factory(NewPublish)
	}
	return nil
}
func (UnpublishComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUnpublish" {
		return custom.Factory(NewUnpublish)
	}
	return nil
}
func NewPublish() xhandler.Contract[PublishInput, PublicationOutput] {
	return &handler[PublishInput, PublicationOutput]{run: runPublish}
}
func NewUnpublish() xhandler.Contract[UnpublishInput, PublicationOutput] {
	return &handler[UnpublishInput, PublicationOutput]{run: runUnpublish}
}

var publicationMu sync.Mutex

func runPublish(ctx context.Context, session xhandler.Session, in *PublishInput, out *PublicationOutput) error {
	if in == nil || out == nil || in.Revision < 1 {
		return publicError(400, "positive namespace skill revision is required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	return store.publish(ctx, in.SkillID, in.ETag, in.Revision, out)
}
func runUnpublish(ctx context.Context, session xhandler.Session, in *UnpublishInput, out *PublicationOutput) error {
	if in == nil || out == nil {
		return publicError(400, "namespace skill input/output required")
	}
	store, err := authorize(ctx, session, in.NamespaceId, in.Jwt, in.Auth)
	if err != nil {
		return err
	}
	return store.publish(ctx, in.SkillID, in.ETag, 0, out)
}
func (s *store) publish(ctx context.Context, id string, etag int64, revision int, out *PublicationOutput) error {
	// Only configured deployment credentials may control the runtime.
	origin := os.Getenv("STUDIO_DYNAMIC_HTTP_URL")
	if origin == "" {
		origin = "http://127.0.0.1:8082"
	}
	admin, err := runtimeadmin.New(origin, os.Getenv("STUDIO_RUNTIME_ADMIN_TOKEN"), nil)
	if err != nil {
		return publicError(503, "namespace runtime administration is not configured")
	}
	publicationMu.Lock()
	defer publicationMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := s.head(ctx, id)
	if err != nil {
		return err
	}
	if etag <= 0 || current.Etag != etag {
		return publicError(409, "namespace skill etag does not match")
	}
	if revision > current.LatestRevision {
		return publicError(404, "namespace skill revision not found")
	}
	if revision > 0 {
		rows, err := s.revisions(ctx, id, revision, 2, 0)
		if err != nil {
			return err
		}
		if len(rows) != 1 {
			return publicError(404, "namespace skill revision not found")
		}
	}
	runtimeCtx := sdk.WithNamespaceSelection(ctx, s.namespaceID)
	// Confirm a live namespace listener before changing the publication pointer.
	if _, err = admin.ProbeRuntime(runtimeCtx); err != nil {
		return publicError(503, "namespace MCP is not ready; enable MCP in namespace settings")
	}
	row := mutation(current, s.subject, time.Now().UTC())
	row.SetPublishedRevision(revision)
	changed, err := s.writeCommitted(ctx, row)
	if err != nil {
		return err
	}
	if err = admin.Reload(runtimeCtx, 0); err != nil {
		// Compensate only our exact staged head; never overwrite a concurrent edit.
		recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		restored := mutation(current, s.subject, time.Now().UTC())
		restored.SetEtag(&changed.ETag)
		if _, rollbackErr := s.writeCommitted(recovery, restored); rollbackErr != nil {
			return publicError(503, "namespace skill activation failed; publication changed concurrently and requires retry")
		}
		if reloadErr := admin.Reload(sdk.WithNamespaceSelection(recovery, s.namespaceID), 0); reloadErr != nil {
			return publicError(503, "namespace skill activation failed; previous selection restored but runtime reconciliation requires retry")
		}
		return publicError(503, "namespace skill activation failed; previous publication restored")
	}
	latest, err := s.head(ctx, id)
	if err != nil {
		return err
	}
	if latest.PublishedRevision != revision {
		return publicError(409, "namespace skill publication changed during activation")
	}
	out.Skill = summary(latest)
	out.PublicationStatus = "active"
	if revision == 0 {
		out.PublicationStatus = "unpublished"
	}
	return nil
}

func publicationCommitError() error {
	return fmt.Errorf("namespace skill publication requires a confirmed independent commit")
}

package authoring

import (
	"context"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

type ListOutput struct {
	Items []*sdk.NamespaceSkill `parameter:"Items,kind=output,in=body,dataType=[]*sdk.NamespaceSkill" json:"items"`
}
type GetOutput struct {
	Skill    *sdk.NamespaceSkill         `parameter:"Skill,kind=output,in=body,dataType=*sdk.NamespaceSkill" json:"skill"`
	Revision *sdk.NamespaceSkillRevision `parameter:"Revision,kind=output,in=body,dataType=*sdk.NamespaceSkillRevision" json:"revision"`
}
type DeleteOutput struct {
	Skill *sdk.NamespaceSkill `parameter:"Skill,kind=output,in=body,dataType=*sdk.NamespaceSkill" json:"skill"`
}
type RevisionsOutput struct {
	Items []*sdk.NamespaceSkillRevision `parameter:"Items,kind=output,in=body,dataType=[]*sdk.NamespaceSkillRevision" json:"items"`
}

type handler[I, O any] struct {
	run func(context.Context, xhandler.Session, *I, *O) error
}

func (h *handler[I, O]) Exec(ctx context.Context, s xhandler.Session, in *I, out *O) error {
	return h.run(ctx, s, in, out)
}

type ListInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Query       string             `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query,omitempty"`
	Limit       int                `parameter:"Limit,kind=body,in=limit,dataType=int,required=false" json:"limit,omitempty"`
	Offset      int                `parameter:"Offset,kind=body,in=offset,dataType=int,required=false" json:"offset,omitempty"`
}
type ListComponent struct {
	Contract xdatly.Component[ListInput, ListOutput] `component:"namespace_skill_list,path=/v1/studio/sdk/namespace_skills.list,method=POST,connector=studio,handler=NewList" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.list\",\"description\":\"List independent skills in an owned namespace\"}]" caseFormat:"lc"`
}

var ListDatly = new(ListComponent)
var ListDatlyLinkedType = reflect.TypeFor[ListComponent]()

func (ListComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewList" {
		return custom.Factory(NewList)
	}
	return nil
}
func NewList() xhandler.Contract[ListInput, ListOutput] {
	return &handler[ListInput, ListOutput]{run: runList}
}

type GetInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string             `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	Revision    int                `parameter:"Revision,kind=body,in=revision,dataType=int,required=false" json:"revision,omitempty"`
}
type GetComponent struct {
	Contract xdatly.Component[GetInput, GetOutput] `component:"namespace_skill_get,path=/v1/studio/sdk/namespace_skills.get,method=POST,connector=studio,handler=NewGet" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.get\",\"description\":\"Read an independent namespace skill and its immutable revision\"}]" caseFormat:"lc"`
}

var GetDatly = new(GetComponent)
var GetDatlyLinkedType = reflect.TypeFor[GetComponent]()

func (GetComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewGet" {
		return custom.Factory(NewGet)
	}
	return nil
}
func NewGet() xhandler.Contract[GetInput, GetOutput] {
	return &handler[GetInput, GetOutput]{run: runGet}
}

type CreateInput struct {
	NamespaceId string                   `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Bundle      sdk.NamespaceSkillBundle `parameter:"Bundle,kind=body,in=bundle,dataType=sdk.NamespaceSkillBundle,required=true" json:"bundle"`
}
type CreateComponent struct {
	Contract xdatly.Component[CreateInput, GetOutput] `component:"namespace_skill_create,path=/v1/studio/sdk/namespace_skills.create,method=POST,connector=studio,handler=NewCreate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.create\",\"description\":\"Create an independent namespace skill draft\"}]" caseFormat:"lc"`
}

var CreateDatly = new(CreateComponent)
var CreateDatlyLinkedType = reflect.TypeFor[CreateComponent]()

func (CreateComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewCreate" {
		return custom.Factory(NewCreate)
	}
	return nil
}
func NewCreate() xhandler.Contract[CreateInput, GetOutput] {
	return &handler[CreateInput, GetOutput]{run: runCreate}
}

type UpdateInput struct {
	NamespaceId string                   `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string                   `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	ETag        int64                    `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
	Bundle      sdk.NamespaceSkillBundle `parameter:"Bundle,kind=body,in=bundle,dataType=sdk.NamespaceSkillBundle,required=true" json:"bundle"`
}
type UpdateComponent struct {
	Contract xdatly.Component[UpdateInput, GetOutput] `component:"namespace_skill_update,path=/v1/studio/sdk/namespace_skills.update,method=POST,connector=studio,handler=NewUpdate" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.update\",\"description\":\"Append an independent namespace skill revision at an expected etag\"}]" caseFormat:"lc"`
}

var UpdateDatly = new(UpdateComponent)
var UpdateDatlyLinkedType = reflect.TypeFor[UpdateComponent]()

func (UpdateComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewUpdate" {
		return custom.Factory(NewUpdate)
	}
	return nil
}
func NewUpdate() xhandler.Contract[UpdateInput, GetOutput] {
	return &handler[UpdateInput, GetOutput]{run: runUpdate}
}

type DeleteInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string             `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	ETag        int64              `parameter:"ETag,kind=body,in=etag,dataType=int64,required=true" json:"etag"`
}
type DeleteComponent struct {
	Contract xdatly.Component[DeleteInput, DeleteOutput] `component:"namespace_skill_delete,path=/v1/studio/sdk/namespace_skills.delete,method=POST,connector=studio,handler=NewDelete" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.delete\",\"description\":\"Archive an unpublished independent namespace skill\"}]" caseFormat:"lc"`
}

var DeleteDatly = new(DeleteComponent)
var DeleteDatlyLinkedType = reflect.TypeFor[DeleteComponent]()

func (DeleteComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewDelete" {
		return custom.Factory(NewDelete)
	}
	return nil
}
func NewDelete() xhandler.Contract[DeleteInput, DeleteOutput] {
	return &handler[DeleteInput, DeleteOutput]{run: runDelete}
}

type RevisionsInput struct {
	NamespaceId string             `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=string,required=true" json:"namespaceId"`
	Jwt         *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	SkillID     string             `parameter:"SkillID,kind=body,in=skillId,dataType=string,required=true" json:"skillId"`
	Limit       int                `parameter:"Limit,kind=body,in=limit,dataType=int,required=false" json:"limit,omitempty"`
	Offset      int                `parameter:"Offset,kind=body,in=offset,dataType=int,required=false" json:"offset,omitempty"`
}
type RevisionsComponent struct {
	Contract xdatly.Component[RevisionsInput, RevisionsOutput] `component:"namespace_skill_revisions,path=/v1/studio/sdk/namespace_skills.revisions,method=POST,connector=studio,handler=NewRevisions" mcp:"[{\"kind\":\"tool\",\"name\":\"studio.sdk.namespace_skills.revisions\",\"description\":\"List immutable revisions of an independent namespace skill\"}]" caseFormat:"lc"`
}

var RevisionsDatly = new(RevisionsComponent)
var RevisionsDatlyLinkedType = reflect.TypeFor[RevisionsComponent]()

func (RevisionsComponent) DatlyHandler(name string) func() (rhandler.TypedHandler, error) {
	if name == "NewRevisions" {
		return custom.Factory(NewRevisions)
	}
	return nil
}
func NewRevisions() xhandler.Contract[RevisionsInput, RevisionsOutput] {
	return &handler[RevisionsInput, RevisionsOutput]{run: runRevisions}
}

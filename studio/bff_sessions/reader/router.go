package reader

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for session.
type SessionComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"session,path=/v1/studio/sessions,method=GET,connector=studio,view=session\" routeName:\"session\" caseFormat:\"lc\""
}

// SessionDatlyType keeps the public component type linked for blank-import discovery.
func SessionDatlyType() reflect.Type { return reflect.TypeOf((*SessionComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var SessionDatly = new(SessionComponent)
var SessionDatlyLinkedType = SessionDatlyType()

func (SessionComponent) EmbedFS() *embed.FS {
	return &SessionDatlyResources
}

func (SessionComponent) EmbedNamespace() string {
	return SessionDatlyResourceNamespace
}

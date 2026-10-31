package value

import (
	"github.com/viant/authz"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// ActiveSelection is one committed channel and its natively related metadata.
// It is not an audience admission; the controller still fences source policies
// and the publisher's selected manifest before exposing the selection.
type ActiveSelection struct {
	Channel      contract.Channel
	Bundle       contract.Bundle
	Bindings     []ComponentBinding
	Dependencies []authz.NamedDependency
}

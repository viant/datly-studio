package readercomponent

import (
	"github.com/viant/bindly/locator"
	"github.com/viant/datly/runtime/handler/provider"
	"github.com/viant/xdatly/handler"
	"github.com/viant/xdatly/state"
)

// Paging supplies native invocation-local selectors separately from business input.
func Paging(view string, limit, offset int) locator.Provider {
	return provider.Static(handler.SelectorsKey, state.Selectors{&state.NamedSelector{Name: view, Selector: state.Selector{Limit: limit, Offset: offset}}})
}

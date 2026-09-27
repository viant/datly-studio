package listoptions

import "reflect"

// Options matches the nested versions.list SDK request. Legacy selectors are
// accepted for wire compatibility but deliberately ignored by the reader.
type Options struct {
	State         string   `json:"state,omitempty"`
	AuthoringMode string   `json:"authoringMode,omitempty"`
	CompileStatus string   `json:"compileStatus,omitempty"`
	CreatedBy     string   `json:"createdBy,omitempty"`
	OrderBy       string   `json:"orderBy,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	Offset        int      `json:"offset,omitempty"`
}

var OptionsDatlyLinkedType = reflect.TypeFor[Options]()

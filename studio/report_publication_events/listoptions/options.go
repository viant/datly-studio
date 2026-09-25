package listoptions

import "reflect"

// Options matches the nested SDK publication-event filters on the wire.
// The component validates and normalizes them before SQL execution.
type Options struct {
	Operation string `json:"operation,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

var OptionsDatlyLinkedType = reflect.TypeFor[Options]()

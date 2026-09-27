package get

import (
	"encoding/json"
	"reflect"

	"github.com/viant/datly-studio/sdk"
)

func (*WarmupGetOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[sdk.WarmupRun]()
}

func (output *WarmupGetOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Response == nil {
		return []byte("null"), nil
	}
	return json.Marshal(output.Response)
}

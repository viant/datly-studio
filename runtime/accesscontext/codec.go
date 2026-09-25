package accesscontext

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	xcodec "github.com/viant/xdatly/codec"
	xresponse "github.com/viant/xdatly/response"
)

// EntityIDsCodec is the declared codec that converts the bound context's
// string IDs into the consuming query's declared ID contract:
//
//	#define($_ = $ProjectIDs<[]string,[]int>(param/Auth.Scope.IDs).WithCodec('EntityIDs').Required()...)
//
// An integer contract accepts only canonical decimals ("007", "+7", " 7" and
// non-numeric IDs deny), so an authorized ID can never alias another row; a
// string contract keeps IDs opaque. An empty ID set denies. The codec is
// registered through the ordinary Datly codec factory, never inferred from a
// dimension or parameter name.
const EntityIDsCodec = "EntityIDs"

// Codecs is the codec factory the runtime host supplies to published
// component artifacts.
func Codecs() xcodec.Factory { return codecFactory{} }

type codecFactory struct{}

func (codecFactory) New(config *xcodec.Config, _ ...xcodec.Option) (xcodec.Instance, error) {
	if config == nil || !strings.EqualFold(strings.TrimSpace(config.Body), EntityIDsCodec) {
		name := ""
		if config != nil {
			name = config.Body
		}
		return nil, fmt.Errorf("codec %q is not registered", name)
	}
	if len(config.Args) != 0 {
		return nil, fmt.Errorf("codec %s accepts no arguments", EntityIDsCodec)
	}
	target := config.DestinationType
	if target == nil {
		var err error
		if target, err = declaredIDType(config.OutputTypeExpression); err != nil {
			return nil, err
		}
	}
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target.Kind() != reflect.Slice || !supportedIDElement(target.Elem()) {
		return nil, fmt.Errorf("codec %s requires a slice of string or integer contract, got %s", EntityIDsCodec, target)
	}
	return &entityIDs{target: target}, nil
}

func declaredIDType(expression string) (reflect.Type, error) {
	switch strings.TrimSpace(expression) {
	case "[]string":
		return reflect.TypeOf([]string{}), nil
	case "[]int":
		return reflect.TypeOf([]int{}), nil
	case "[]int64":
		return reflect.TypeOf([]int64{}), nil
	case "[]int32":
		return reflect.TypeOf([]int32{}), nil
	case "[]uint":
		return reflect.TypeOf([]uint{}), nil
	case "[]uint64":
		return reflect.TypeOf([]uint64{}), nil
	case "[]uint32":
		return reflect.TypeOf([]uint32{}), nil
	}
	return nil, fmt.Errorf("codec %s requires an explicit []string or integer slice output type, got %q", EntityIDsCodec, expression)
}

func supportedIDElement(element reflect.Type) bool {
	switch element.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

type entityIDs struct{ target reflect.Type }

func (c *entityIDs) Value(_ context.Context, raw interface{}, _ ...xcodec.Option) (interface{}, error) {
	ids, err := stringIDs(raw)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, forbidden("access context grants no entity IDs")
	}
	element := c.target.Elem()
	result := reflect.MakeSlice(c.target, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || strings.TrimSpace(id) != id || seen[id] {
			return nil, forbidden("access context entity ID is malformed")
		}
		seen[id] = true
		value := reflect.New(element).Elem()
		switch element.Kind() {
		case reflect.String:
			value.SetString(id)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parsed, err := strconv.ParseInt(id, 10, element.Bits())
			if err != nil || strconv.FormatInt(parsed, 10) != id {
				return nil, forbidden("access context entity ID is not a canonical integer")
			}
			value.SetInt(parsed)
		default:
			parsed, err := strconv.ParseUint(id, 10, element.Bits())
			if err != nil || strconv.FormatUint(parsed, 10) != id {
				return nil, forbidden("access context entity ID is not a canonical unsigned integer")
			}
			value.SetUint(parsed)
		}
		result = reflect.Append(result, value)
	}
	return result.Interface(), nil
}

func stringIDs(raw interface{}) ([]string, error) {
	switch actual := raw.(type) {
	case nil:
		return nil, nil
	case []string:
		return actual, nil
	case *[]string:
		if actual == nil {
			return nil, nil
		}
		return *actual, nil
	case string:
		return []string{actual}, nil
	}
	value := reflect.ValueOf(raw)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice || value.Type().Elem().Kind() != reflect.String {
		return nil, &xresponse.Error{Code: 403, Cause: errors.New("access context entity IDs must be bound from the context component")}
	}
	result := make([]string, value.Len())
	for i := range result {
		result[i] = value.Index(i).String()
	}
	return result, nil
}

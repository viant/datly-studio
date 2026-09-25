package accesscontext

import (
	"context"
	"reflect"
	"testing"

	xcodec "github.com/viant/xdatly/codec"
	xresponse "github.com/viant/xdatly/response"
)

func TestEntityIDsCodecConvertsOnlyCanonicalIntegers(t *testing.T) {
	instance, err := Codecs().New(&xcodec.Config{Body: "entityids", DestinationType: reflect.TypeOf([]int{})})
	if err != nil {
		t.Fatal(err)
	}
	value, err := instance.Value(context.Background(), []string{"101", "102", "-3"})
	if err != nil || !reflect.DeepEqual(value, []int{101, 102, -3}) {
		t.Fatalf("value=%#v err=%v", value, err)
	}
	for _, malformed := range [][]string{{"abc"}, {"007"}, {"+101"}, {" 101"}, {"101.0"}, {"1 OR 1=1"}, {"101", "101"}, {""}, {}, nil, {"0x10"}, {"92233720368547758070"}} {
		if value, err := instance.Value(context.Background(), malformed); err == nil || xresponse.ErrorStatusCode(err, 0) != 403 {
			t.Fatalf("ids %q accepted: value=%#v err=%v", malformed, value, err)
		}
	}
	unsigned, err := Codecs().New(&xcodec.Config{Body: EntityIDsCodec, OutputTypeExpression: "[]uint32"})
	if err != nil {
		t.Fatal(err)
	}
	if value, err := unsigned.Value(context.Background(), []string{"4294967295"}); err != nil || !reflect.DeepEqual(value, []uint32{4294967295}) {
		t.Fatalf("value=%#v err=%v", value, err)
	}
	for _, malformed := range [][]string{{"-1"}, {"4294967296"}} {
		if _, err := unsigned.Value(context.Background(), malformed); err == nil {
			t.Fatalf("unsigned accepted %q", malformed)
		}
	}
}

func TestEntityIDsCodecKeepsStringContractsOpaque(t *testing.T) {
	instance, err := Codecs().New(&xcodec.Config{Body: EntityIDsCodec, DestinationType: reflect.TypeOf([]string{})})
	if err != nil {
		t.Fatal(err)
	}
	value, err := instance.Value(context.Background(), []string{"p-1", "007", "1' OR '1'='1"})
	if err != nil || !reflect.DeepEqual(value, []string{"p-1", "007", "1' OR '1'='1"}) {
		t.Fatalf("value=%#v err=%v", value, err)
	}
	for _, malformed := range [][]string{{""}, {" x"}, {"a", "a"}, {}} {
		if _, err := instance.Value(context.Background(), malformed); err == nil {
			t.Fatalf("string contract accepted %q", malformed)
		}
	}
	if _, err := instance.Value(context.Background(), []int{1}); err == nil {
		t.Fatal("non-string source accepted")
	}
}

func TestEntityIDsCodecRejectsUnsupportedDeclarations(t *testing.T) {
	for name, config := range map[string]*xcodec.Config{
		"unknown codec":     {Body: "AsInts", DestinationType: reflect.TypeOf([]int{})},
		"arguments":         {Body: EntityIDsCodec, Args: []string{"x"}, DestinationType: reflect.TypeOf([]int{})},
		"scalar contract":   {Body: EntityIDsCodec, DestinationType: reflect.TypeOf(0)},
		"float contract":    {Body: EntityIDsCodec, DestinationType: reflect.TypeOf([]float64{})},
		"missing contract":  {Body: EntityIDsCodec},
		"unknown expr":      {Body: EntityIDsCodec, OutputTypeExpression: "[]float64"},
		"nil configuration": nil,
	} {
		t.Run(name, func(t *testing.T) {
			if instance, err := Codecs().New(config); err == nil {
				t.Fatalf("accepted: %#v", instance)
			}
		})
	}
}

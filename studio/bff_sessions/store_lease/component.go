// Package store_lease owns the atomic, server-only refresh-token lease.
package store_lease

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	leasewrite "github.com/viant/datly-studio/studio/bff_sessions/lease_write"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/spec"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
)

type Input struct {
	Operation         string `parameter:"Operation,kind=body,in=operation"`
	SessionIDHash     string `parameter:"SessionIDHash,kind=body,in=sessionIdHash"`
	LeaseOwner        string `parameter:"LeaseOwner,kind=body,in=leaseOwner"`
	NowUnix           int64  `parameter:"NowUnix,kind=body,in=nowUnix"`
	LeaseUntilUnix    int64  `parameter:"LeaseUntilUnix,kind=body,in=leaseUntilUnix"`
	PayloadCiphertext []byte `parameter:"PayloadCiphertext,kind=body,in=payloadCiphertext"`
}

type Output struct {
	Applied bool `json:"applied"`
}

type SessionComponent struct {
	Contract xdatly.Component[Input, Output] `component:"lease,path=/_studio/bff-session-lease,method=POST,connector=studio,handler=NewLease,internal=true"`
}

var SessionDatly = new(SessionComponent)
var SessionDatlyLinkedType = reflect.TypeFor[SessionComponent]()

func (SessionComponent) DatlyHandler(name string) func() (handler.TypedHandler, error) {
	if name == "NewLease" {
		return custom.Factory(NewLease)
	}
	return nil
}

type leaseHandler struct{}

func NewLease() xhandler.Contract[Input, Output] { return &leaseHandler{} }

func (*leaseHandler) Exec(ctx context.Context, session xhandler.Session, input *Input, output *Output) error {
	if session == nil || session.Binder() == nil || input == nil || output == nil {
		return errors.New("refresh lease invocation is incomplete")
	}
	if len(input.SessionIDHash) != 64 || len(input.LeaseOwner) < 16 || len(input.LeaseOwner) > 64 ||
		strings.TrimSpace(input.LeaseOwner) != input.LeaseOwner || input.NowUnix <= 0 {
		return errors.New("invalid refresh lease identity")
	}
	if _, err := hex.DecodeString(input.SessionIDHash); err != nil {
		return errors.New("invalid refresh lease session hash")
	}
	row := &leasewrite.LeaseSession{}
	row.SetSessionIdHash(input.SessionIDHash)
	switch input.Operation {
	case "acquire":
		if input.LeaseUntilUnix <= input.NowUnix {
			return errors.New("refresh lease expiry must follow acquisition")
		}
		row.SetRefreshLeaseOwner(input.LeaseOwner)
		row.SetRefreshLeaseUntilUnix(input.LeaseUntilUnix)
	case "complete":
		if len(input.PayloadCiphertext) == 0 {
			return errors.New("refresh completion requires encrypted session payload")
		}
		row.SetPayloadCiphertext(input.PayloadCiphertext)
		row.SetRefreshLeaseOwner("")
		row.SetRefreshLeaseUntilUnix(0)
	case "release":
		row.SetRefreshLeaseOwner("")
		row.SetRefreshLeaseUntilUnix(0)
	default:
		return fmt.Errorf("unsupported refresh lease operation %q", input.Operation)
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return errors.New("native refresh lease writer is unavailable")
	}
	request := &leasewrite.Input{}
	request.SetOperation(input.Operation)
	request.SetLeaseOwner(input.LeaseOwner)
	request.SetNowUnix(input.NowUnix)
	request.SetGuard(true)
	request.SetSessions([]*leasewrite.LeaseSession{row})
	_, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[leasewrite.LeaseComponent]().PkgPath(), Name: "lease"}, Route: spec.RouteRef{Method: "PATCH", Path: "/_studio/bff-session-lease/write"}}, Input: request})
	if err != nil {
		return err
	}
	output.Applied = true
	return nil
}

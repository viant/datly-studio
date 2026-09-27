// Package store_lease owns the atomic, server-only refresh-token lease.
package store_lease

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/xdatly"
	xhandler "github.com/viant/xdatly/handler"
	"github.com/viant/xdatly/response"
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
	value, found, err := session.Binder().Lookup(ctx, handler.TransactionSQLCapabilityKey)
	if err != nil {
		return err
	}
	provider, ok := value.(handler.TransactionSQLProvider)
	if !found || !ok {
		return errors.New("managed refresh lease SQL is unavailable")
	}
	sql, err := provider.Connector(ctx, "studio")
	if err != nil {
		return err
	}
	var query string
	var args []any
	switch input.Operation {
	case "acquire":
		if input.LeaseUntilUnix <= input.NowUnix {
			return errors.New("refresh lease expiry must follow acquisition")
		}
		query = `UPDATE bff_sessions SET refresh_lease_owner=?,refresh_lease_until_unix=?
			WHERE session_id_hash=? AND expires_at_unix>? AND
			(refresh_lease_owner='' OR refresh_lease_until_unix<?)`
		args = []any{input.LeaseOwner, input.LeaseUntilUnix, input.SessionIDHash, input.NowUnix, input.NowUnix}
	case "complete":
		if len(input.PayloadCiphertext) == 0 {
			return errors.New("refresh completion requires encrypted session payload")
		}
		query = `UPDATE bff_sessions SET payload_ciphertext=?,refresh_lease_owner='',refresh_lease_until_unix=0
			WHERE session_id_hash=? AND refresh_lease_owner=? AND refresh_lease_until_unix>=? AND expires_at_unix>?`
		args = []any{input.PayloadCiphertext, input.SessionIDHash, input.LeaseOwner, input.NowUnix, input.NowUnix}
	case "release":
		query = `UPDATE bff_sessions SET refresh_lease_owner='',refresh_lease_until_unix=0
			WHERE session_id_hash=? AND refresh_lease_owner=?`
		args = []any{input.SessionIDHash, input.LeaseOwner}
	default:
		return fmt.Errorf("unsupported refresh lease operation %q", input.Operation)
	}
	result, err := sql.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 1 {
		return errors.New("refresh lease changed multiple sessions")
	}
	output.Applied = count == 1
	if input.Operation == "complete" && !output.Applied {
		return &response.Error{Code: 409, Cause: errors.New("refresh lease ownership changed")}
	}
	return nil
}

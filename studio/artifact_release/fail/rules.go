package fail

import (
	"context"
	"encoding/hex"
	"encoding/json"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	contract "github.com/viant/forge/backend/artifactrelease"
	xhandler "github.com/viant/xdatly/handler"
	"math"
	"strings"
)

// Rules checks business intent; all reads and mutations belong to the generated graph.
type Rules struct {
	Input *Input `bind:"kind=input,required"`
}

func str(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func (rules *Rules) Init(ctx context.Context, row *FailChannel, state xhandler.LifecycleContext[FailChannel, xhandler.NoParent, Output]) error {
	if rules == nil || rules.Input == nil || row == nil || state.Previous == nil {
		return contract.ErrConflict
	}
	input := rules.Input
	if input.ExpectedRevision < 0 || input.ExpectedRevision >= math.MaxInt64 || input.PendingStatus != "pending" || input.Actor == "" || strings.TrimSpace(input.Actor) != input.Actor || strings.ContainsRune(input.Actor, 0) || len(input.OperationId) != 32 || input.OperationId != strings.ToLower(input.OperationId) {
		return contract.ErrInvalid
	}
	if _, err := hex.DecodeString(input.OperationId); err != nil {
		return contract.ErrInvalid
	}
	if len(input.ChannelSnapshot) != 1 || len(input.Intent) != 1 {
		return contract.ErrConflict
	}
	snapshot, intent := input.ChannelSnapshot[0], input.Intent[0]
	if snapshot == nil || intent == nil || str(row.ScopeId) != input.ScopeId || str(row.Name) != input.Name || snapshot.Revision != input.ExpectedRevision || str(snapshot.PendingOperationId) != input.OperationId || row.Revision != input.ExpectedRevision || str(intent.OperationId) != input.OperationId || str(intent.ScopeId) != input.ScopeId || str(intent.ChannelName) != input.Name || str(intent.Status) != "pending" || intent.ExpectedRevision != input.ExpectedRevision || intent.TargetRevision != input.ExpectedRevision+1 {
		return contract.ErrConflict
	}
	scope := releasevalue.Scope{Tenant: str(snapshot.TenantId), Account: str(snapshot.AccountId), Workspace: str(snapshot.WorkspaceId), Namespace: str(snapshot.Namespace)}
	if !scope.Valid() || scope.ID() != input.ScopeId || intent.NextChannelJson == nil {
		return contract.ErrInvalid
	}
	var next contract.Channel
	if releasevalue.StrictJSON(*intent.NextChannelJson, &next) != nil || contract.ValidateChannel(next) != nil || !scope.Owns(next.Key) || next.Key.Name != input.Name || next.Revision != uint64(intent.TargetRevision) || next.ActiveBundleHash != str(intent.TargetBundleHash) {
		return contract.ErrInvalid
	}
	if len(row.Operation) != 1 || row.Operation[0] == nil || row.UpdatedAt == nil || str(row.UpdatedBy) != input.Actor {
		return contract.ErrInvalid
	}
	op := row.Operation[0]
	if str(op.OperationId) != input.OperationId || str(op.ScopeId) != input.ScopeId || str(op.ChannelName) != input.Name || op.UpdatedAt == nil || op.CompletedAt == nil || !op.UpdatedAt.Equal(*row.UpdatedAt) || !op.CompletedAt.Equal(*row.UpdatedAt) || str(op.UpdatedBy) != input.Actor {
		return contract.ErrInvalid
	}
	_ = json.Valid
	if str(op.Status) != "failed" || op.FailureCode == nil || len(*op.FailureCode) == 0 || len(*op.FailureCode) > 64 {
		return contract.ErrInvalid
	}
	for _, c := range *op.FailureCode {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return contract.ErrInvalid
		}
	}
	row.SetPendingOperationId(nil)
	return nil
}
func (*Rules) Validate(context.Context, *FailChannel, xhandler.LifecycleContext[FailChannel, xhandler.NoParent, Output]) error {
	return nil
}

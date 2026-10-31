package stage

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	contract "github.com/viant/forge/backend/artifactrelease"
	xhandler "github.com/viant/xdatly/handler"
)

// These are application-owned business hooks. DQL owns every persistence view,
// Current read, execution predicate, link, presence marker and queued mutation.
type StageRules struct {
	Input *Input `bind:"kind=input,required"`
}
type BindingRules struct {
	Input *Input `bind:"kind=input,required"`
}

func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func rawCopy(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	copy := append(json.RawMessage(nil), (*value)...)
	return &copy
}
func sourceScope(input *Input) releasevalue.Scope {
	if input == nil {
		return releasevalue.Scope{}
	}
	return releasevalue.Scope{Tenant: input.ScopeTenant, Account: input.ScopeAccount, Workspace: input.ScopeWorkspace, Namespace: input.ScopeNamespace}
}
func channelValue(scope releasevalue.Scope, name string, description *string, policies *json.RawMessage, active *string, revision int64) (contract.Channel, error) {
	if description == nil || policies == nil || revision < 0 {
		return contract.Channel{}, contract.ErrInvalid
	}
	result := contract.Channel{Key: contract.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: name}, Description: *description, ActiveBundleHash: text(active), Revision: uint64(revision)}
	if releasevalue.StrictJSON(*policies, &result.Policies) != nil || contract.ValidateChannel(result) != nil {
		return result, contract.ErrInvalid
	}
	return result, nil
}
func (rules *StageRules) Init(ctx context.Context, row *StageChannel, state xhandler.LifecycleContext[StageChannel, xhandler.NoParent, Output]) error {
	input := rules.Input
	scope := sourceScope(input)
	previous := state.Previous
	if input == nil || row == nil || previous == nil || !scope.Valid() || scope.ID() != input.ScopeId || row.ScopeId == nil || *row.ScopeId != input.ScopeId || text(row.Name) != input.Name || input.Actor == "" || strings.TrimSpace(input.Actor) != input.Actor || strings.ContainsRune(input.Actor, 0) {
		return contract.ErrInvalid
	}
	reads, err := input.ReadIndexes(ctx)
	if err != nil {
		return err
	}
	if len(input.ChannelSnapshot) != 1 {
		return contract.ErrInvalid
	}
	snapshot := input.ChannelSnapshot[0]
	if snapshot == nil {
		return contract.ErrInvalid
	}
	if text(row.ExpectedTenantId) != scope.Tenant || text(row.ExpectedAccountId) != scope.Account || text(row.ExpectedWorkspaceId) != scope.Workspace || text(row.ExpectedNamespace) != scope.Namespace || text(snapshot.TenantId) != scope.Tenant || text(snapshot.AccountId) != scope.Account || text(snapshot.WorkspaceId) != scope.Workspace || text(snapshot.Namespace) != scope.Namespace {
		return contract.ErrInvalid
	}
	current, err := channelValue(scope, input.Name, row.ExpectedDescription, row.ExpectedPolicyRefsJson, row.ExpectedActiveBundleHash, row.Revision)
	if err != nil {
		return err
	}
	stored, err := channelValue(scope, input.Name, snapshot.Description, snapshot.PolicyRefsJson, snapshot.ActiveBundleHash, snapshot.Revision)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, stored) || row.Revision != input.ExpectedRevision || previous.PendingOperationId != nil {
		return contract.ErrConflict
	}
	if input.TargetBundleHash != row.TargetBundleHash || !releasevalue.ValidHash(row.TargetBundleHash) || row.PendingOperationId == nil || len(*row.PendingOperationId) != 32 || *row.PendingOperationId != strings.ToLower(*row.PendingOperationId) || row.UpdatedAt == nil || text(row.UpdatedBy) != input.Actor {
		return contract.ErrInvalid
	}
	if _, err = hex.DecodeString(*row.PendingOperationId); err != nil {
		return contract.ErrInvalid
	}
	bundleRow := reads.CurrentBundleByKey[StageHandlerCurrentBundleKey{ScopeId: input.ScopeId, BundleHash: input.TargetBundleHash}]
	if bundleRow == nil || bundleRow.BundleJson == nil || text(bundleRow.TenantId) != scope.Tenant || text(bundleRow.AccountId) != scope.Account || text(bundleRow.WorkspaceId) != scope.Workspace || text(bundleRow.Namespace) != scope.Namespace {
		return contract.ErrInvalid
	}
	var bundle contract.Bundle
	if releasevalue.StrictJSON(*bundleRow.BundleJson, &bundle) != nil || contract.ValidateBundle(bundle) != nil || bundle.Hash != input.TargetBundleHash || bundle.Namespace != scope.Namespace {
		return contract.ErrInvalid
	}
	if len(row.Operation) != 1 || row.Operation[0] == nil {
		return contract.ErrInvalid
	}
	operation := row.Operation[0]
	if text(operation.OperationId) != *row.PendingOperationId || text(operation.ScopeId) != input.ScopeId || text(operation.ChannelName) != input.Name || operation.ExpectedRevision != int64(current.Revision) {
		return contract.ErrInvalid
	}
	if operation.TargetRevision != int64(current.Revision)+1 || text(operation.TargetBundleHash) != bundle.Hash || operation.NextChannelJson == nil || text(operation.Status) != "pending" || operation.CreatedAt == nil || operation.UpdatedAt == nil || !operation.CreatedAt.Equal(*operation.UpdatedAt) || !operation.UpdatedAt.Equal(*row.UpdatedAt) || text(operation.CreatedBy) != input.Actor || text(operation.UpdatedBy) != input.Actor || operation.CompletedAt != nil || operation.FailureCode != nil {
		return contract.ErrInvalid
	}
	var next contract.Channel
	if releasevalue.StrictJSON(*operation.NextChannelJson, &next) != nil || contract.ValidateChannel(next) != nil || next.Key != current.Key || next.Revision != current.Revision+1 || next.ActiveBundleHash != bundle.Hash {
		return contract.ErrInvalid
	}
	if operation.DependenciesJson == nil {
		empty := json.RawMessage(`[]`)
		operation.SetDependenciesJson(&empty)
	}
	if _, err = releasevalue.DecodeDependencies(*operation.DependenciesJson); err != nil {
		return err
	}
	existingBinding := reads.CurrentBindingByKey[StageHandlerCurrentBindingKey{ScopeId: input.ScopeId, BundleHash: input.TargetBundleHash}]
	if !releasevalue.ValidHash(input.BindingsHash) {
		return contract.ErrInvalid
	}
	if row.Binding == nil {
		if existingBinding == nil || existingBinding.BindingsJson == nil || text(existingBinding.BindingsHash) != input.BindingsHash {
			return contract.ErrConflict
		}
		if _, err := releasevalue.DecodeBindings(bundle, *existingBinding.BindingsJson, input.BindingsHash); err != nil {
			return err
		}
		return nil
	}
	if existingBinding != nil {
		return contract.ErrConflict
	}
	if text(row.Binding.BindingsHash) != input.BindingsHash {
		return contract.ErrInvalid
	}
	if row.Binding.CreatedAt == nil || row.Binding.UpdatedAt == nil || text(row.Binding.CreatedBy) != input.Actor || text(row.Binding.UpdatedBy) != input.Actor {
		return contract.ErrInvalid
	}
	return nil
}
func (rules *StageRules) Validate(context.Context, *StageChannel, xhandler.LifecycleContext[StageChannel, xhandler.NoParent, Output]) error {
	return nil
}

func (rules *BindingRules) Init(ctx context.Context, row *BundleBinding, state xhandler.LifecycleContext[BundleBinding, StageChannel, Output]) error {
	if rules == nil || rules.Input == nil || row == nil || state.Parent == nil {
		return contract.ErrInvalid
	}
	input := rules.Input
	if text(row.ScopeId) != input.ScopeId || text(row.BundleHash) != input.TargetBundleHash || row.BindingsJson == nil || row.BindingsHash == nil {
		return contract.ErrInvalid
	}
	reads, err := input.ReadIndexes(ctx)
	if err != nil {
		return err
	}
	source := reads.CurrentBundleByKey[StageHandlerCurrentBundleKey{ScopeId: input.ScopeId, BundleHash: input.TargetBundleHash}]
	if source == nil || source.BundleJson == nil {
		return contract.ErrInvalid
	}
	var bundle contract.Bundle
	if releasevalue.StrictJSON(*source.BundleJson, &bundle) != nil {
		return contract.ErrInvalid
	}
	_, err = releasevalue.DecodeBindings(bundle, *row.BindingsJson, *row.BindingsHash)
	if err != nil {
		return err
	}
	if state.Previous != nil {
		return contract.ErrConflict
	}
	return nil
}
func (rules *BindingRules) Validate(context.Context, *BundleBinding, xhandler.LifecycleContext[BundleBinding, StageChannel, Output]) error {
	return nil
}

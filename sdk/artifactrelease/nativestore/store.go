// Package nativestore adapts trusted release operations to stock-transcribed
// Datly components. It does not own a SQL connection or transaction.
package nativestore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/viant/authz"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	bundleview "github.com/viant/datly-studio/studio/artifact_release/bundle"
	channelsview "github.com/viant/datly-studio/studio/artifact_release/channels"
	completeview "github.com/viant/datly-studio/studio/artifact_release/complete"
	createview "github.com/viant/datly-studio/studio/artifact_release/create_channel"
	failview "github.com/viant/datly-studio/studio/artifact_release/fail"
	historyview "github.com/viant/datly-studio/studio/artifact_release/history"
	intentview "github.com/viant/datly-studio/studio/artifact_release/intent"
	putview "github.com/viant/datly-studio/studio/artifact_release/put_bundle"
	snapshotview "github.com/viant/datly-studio/studio/artifact_release/snapshot"
	stageview "github.com/viant/datly-studio/studio/artifact_release/stage"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	contract "github.com/viant/forge/backend/artifactrelease"
	xhandler "github.com/viant/xdatly/handler"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

type Invoker interface {
	InvokeComponent(context.Context, exec.ComponentRequest) (any, error)
}
type Store struct {
	invoker Invoker
	scope   releasevalue.Scope
	actor   func(context.Context) (string, error)
	clock   func() time.Time
}

func New(invoker Invoker, scope releasevalue.Scope, actor func(context.Context) (string, error)) (*Store, error) {
	if invoker == nil || !scope.Valid() || actor == nil {
		return nil, contract.ErrInvalid
	}
	return &Store{invoker: invoker, scope: scope, actor: actor, clock: func() time.Time { return time.Now().UTC() }}, nil
}
func (s *Store) ready(ctx context.Context) bool {
	return s != nil && s.invoker != nil && ctx != nil && ctx.Err() == nil
}
func (s *Store) requestActor(ctx context.Context) (string, error) {
	a, err := s.actor(ctx)
	if err != nil {
		return "", err
	}
	if a == "" || strings.TrimSpace(a) != a || strings.ContainsRune(a, 0) {
		return "", contract.ErrInvalid
	}
	return a, nil
}
func (s *Store) invoke(ctx context.Context, typ reflect.Type, name, method string, input any) (any, error) {
	value, err := s.invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: typ.PkgPath(), Name: name}, Route: spec.RouteRef{Method: method, Path: "/_studio/artifact-release-store/" + name}}, Input: input})
	var conflict *xhandler.Conflict
	if errors.As(err, &conflict) {
		return nil, errors.Join(contract.ErrConflict, err)
	}
	return value, err
}
func str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func (s *Store) snapshot(ctx context.Context, key contract.Key) (*snapshotview.ReleaseChannel, error) {
	if !s.ready(ctx) || !s.scope.Owns(key) {
		return nil, contract.ErrInvalid
	}
	i := &snapshotview.Input{}
	i.SetScopeId(s.scope.ID())
	i.SetName(key.Name)
	v, e := s.invoke(ctx, reflect.TypeFor[snapshotview.SnapshotComponent](), "snapshot", "GET", i)
	if e != nil {
		return nil, e
	}
	o, ok := v.(*snapshotview.Output)
	if !ok {
		return nil, contract.ErrInvalid
	}
	if len(o.Data) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(o.Data) != 1 || o.Data[0] == nil {
		return nil, contract.ErrInvalid
	}
	r := o.Data[0]
	if str(r.TenantId) != s.scope.Tenant || str(r.AccountId) != s.scope.Account || str(r.WorkspaceId) != s.scope.Workspace || str(r.Namespace) != s.scope.Namespace || str(r.ScopeId) != s.scope.ID() || str(r.Name) != key.Name {
		return nil, contract.ErrInvalid
	}
	return r, nil
}
func (s *Store) Channel(ctx context.Context, key contract.Key) (contract.Channel, error) {
	r, e := s.snapshot(ctx, key)
	if e != nil {
		return contract.Channel{}, e
	}
	return channel(r)
}
func channel(r *snapshotview.ReleaseChannel) (contract.Channel, error) {
	c := contract.Channel{Key: contract.Key{Tenant: str(r.TenantId), Account: str(r.AccountId), Workspace: str(r.WorkspaceId), Namespace: str(r.Namespace), Name: str(r.Name)}, Description: str(r.Description), ActiveBundleHash: str(r.ActiveBundleHash)}
	if r.Revision < 0 || r.PolicyRefsJson == nil {
		return c, contract.ErrInvalid
	}
	c.Revision = uint64(r.Revision)
	if releasevalue.StrictJSON(*r.PolicyRefsJson, &c.Policies) != nil || contract.ValidateChannel(c) != nil {
		return c, contract.ErrInvalid
	}
	return c, nil
}
func (s *Store) storedBundle(ctx context.Context, hash string) (*bundleview.StoredBundle, error) {
	if !s.ready(ctx) || !releasevalue.ValidHash(hash) {
		return nil, contract.ErrInvalid
	}
	i := &bundleview.Input{}
	i.SetScopeId(s.scope.ID())
	i.SetBundleHash(hash)
	v, e := s.invoke(ctx, reflect.TypeFor[bundleview.BundleComponent](), "bundle", "GET", i)
	if e != nil {
		return nil, e
	}
	o, ok := v.(*bundleview.Output)
	if !ok {
		return nil, contract.ErrInvalid
	}
	if len(o.Data) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(o.Data) != 1 || o.Data[0] == nil {
		return nil, contract.ErrInvalid
	}
	r := o.Data[0]
	if str(r.TenantId) != s.scope.Tenant || str(r.AccountId) != s.scope.Account || str(r.WorkspaceId) != s.scope.Workspace || str(r.Namespace) != s.scope.Namespace || str(r.ScopeId) != s.scope.ID() || str(r.BundleHash) != hash {
		return nil, contract.ErrInvalid
	}
	return r, nil
}
func (s *Store) Bundle(ctx context.Context, hash string) (contract.Bundle, error) {
	r, e := s.storedBundle(ctx, hash)
	if e != nil {
		return contract.Bundle{}, e
	}
	return decodeBundle(r)
}
func decodeBundle(r *bundleview.StoredBundle) (contract.Bundle, error) {
	var b contract.Bundle
	if r.BundleJson == nil || releasevalue.StrictJSON(*r.BundleJson, &b) != nil || b.Hash != str(r.BundleHash) || contract.ValidateBundle(b) != nil {
		return b, contract.ErrInvalid
	}
	return b, nil
}
func (s *Store) BundleBindings(ctx context.Context, b contract.Bundle) ([]releasevalue.ComponentBinding, error) {
	if !s.ready(ctx) || b.Namespace != s.scope.Namespace || contract.ValidateBundle(b) != nil {
		return nil, contract.ErrInvalid
	}
	r, e := s.storedBundle(ctx, b.Hash)
	if e != nil {
		return nil, e
	}
	if r.Binding == nil || r.Binding.BindingsJson == nil {
		return nil, sql.ErrNoRows
	}
	return releasevalue.DecodeBindings(b, *r.Binding.BindingsJson, str(r.Binding.BindingsHash))
}
func (s *Store) ActiveDependencies(ctx context.Context, key contract.Key) ([]authz.NamedDependency, error) {
	r, e := s.snapshot(ctx, key)
	if e != nil {
		return nil, e
	}
	if r.ActiveDependenciesJson == nil {
		return releasevalue.DecodeDependencies(nil)
	}
	return releasevalue.DecodeDependencies(*r.ActiveDependenciesJson)
}
func (s *Store) StageRevision(ctx context.Context, current, next contract.Channel, b contract.Bundle) (string, error) {
	return s.StageRevisionWithBindingsAndDependencies(ctx, current, next, b, nil, nil)
}
func (s *Store) StageRevisionWithBindings(ctx context.Context, current, next contract.Channel, b contract.Bundle, bindings []releasevalue.ComponentBinding) (string, error) {
	return s.StageRevisionWithBindingsAndDependencies(ctx, current, next, b, bindings, nil)
}
func (s *Store) StageRevisionWithBindingsAndDependencies(ctx context.Context, current, next contract.Channel, b contract.Bundle, bindings []releasevalue.ComponentBinding, deps []authz.NamedDependency) (string, error) {
	if !s.ready(ctx) || !s.scope.Owns(current.Key) || next.Key != current.Key || current.Revision >= math.MaxInt64 || next.Revision != current.Revision+1 || next.ActiveBundleHash != b.Hash || b.Namespace != s.scope.Namespace || contract.ValidateChannel(current) != nil || contract.ValidateChannel(next) != nil || contract.ValidateBundle(b) != nil {
		return "", contract.ErrInvalid
	}
	_, raw, hash, e := releasevalue.ValidateBindings(b, bindings)
	if e != nil {
		return "", e
	}
	dr, e := releasevalue.MarshalDependencies(deps)
	if e != nil {
		return "", e
	}
	actor, e := s.requestActor(ctx)
	if e != nil {
		return "", e
	}
	prior, e := s.storedBundle(ctx, b.Hash)
	if e != nil {
		return "", e
	}
	stored, e := decodeBundle(prior)
	if e != nil {
		return "", e
	}
	if !reflect.DeepEqual(b, stored) {
		return "", contract.ErrConflict
	}
	if prior.Binding != nil {
		if prior.Binding.BindingsJson == nil || str(prior.Binding.BindingsHash) != hash {
			return "", contract.ErrConflict
		}
		existing, e := releasevalue.DecodeBindings(b, *prior.Binding.BindingsJson, hash)
		if e != nil {
			return "", e
		}
		incoming, e := releasevalue.DecodeBindings(b, raw, hash)
		if e != nil || !reflect.DeepEqual(existing, incoming) {
			return "", contract.ErrConflict
		}
	}
	nonce := make([]byte, 16)
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	id := hex.EncodeToString(nonce)
	now := s.clock().UTC()
	scope := s.scope.ID()
	name := current.Key.Name
	row := &stageview.StageChannel{}
	row.SetScopeId(&scope)
	row.SetName(&name)
	row.SetRevision(int64(current.Revision))
	row.SetPendingOperationId(&id)
	row.SetUpdatedAt(&now)
	row.SetUpdatedBy(&actor)
	row.SetTargetBundleHash(b.Hash)
	row.SetExpectedTenantId(&s.scope.Tenant)
	row.SetExpectedAccountId(&s.scope.Account)
	row.SetExpectedWorkspaceId(&s.scope.Workspace)
	row.SetExpectedNamespace(&s.scope.Namespace)
	row.SetExpectedDescription(&current.Description)
	pol, _ := json.Marshal(current.Policies)
	polJSON := json.RawMessage(pol)
	row.SetExpectedPolicyRefsJson(&polJSON)
	if current.ActiveBundleHash != "" {
		row.SetExpectedActiveBundleHash(&current.ActiveBundleHash)
	} else {
		row.SetExpectedActiveBundleHash(nil)
	}
	op := &stageview.StageOperation{}
	op.SetOperationId(&id)
	op.SetScopeId(&scope)
	op.SetChannelName(&name)
	op.SetExpectedRevision(int64(current.Revision))
	op.SetTargetRevision(int64(next.Revision))
	op.SetTargetBundleHash(&b.Hash)
	nr, _ := json.Marshal(next)
	nextJSON, depJSON := json.RawMessage(nr), json.RawMessage(dr)
	op.SetNextChannelJson(&nextJSON)
	op.SetDependenciesJson(&depJSON)
	status := "pending"
	op.SetStatus(&status)
	op.SetCreatedAt(&now)
	op.SetCreatedBy(&actor)
	op.SetUpdatedAt(&now)
	op.SetUpdatedBy(&actor)
	row.SetOperation([]*stageview.StageOperation{op})
	if prior.Binding == nil {
		binding := &stageview.BundleBinding{}
		binding.SetScopeId(&scope)
		binding.SetBundleHash(&b.Hash)
		br := json.RawMessage(raw)
		binding.SetBindingsJson(&br)
		binding.SetBindingsHash(&hash)
		binding.SetCreatedAt(&now)
		binding.SetCreatedBy(&actor)
		binding.SetUpdatedAt(&now)
		binding.SetUpdatedBy(&actor)
		row.SetBinding(binding)
	}
	i := &stageview.Input{}
	i.SetScopeId(scope)
	i.SetScopeTenant(s.scope.Tenant)
	i.SetScopeAccount(s.scope.Account)
	i.SetScopeWorkspace(s.scope.Workspace)
	i.SetScopeNamespace(s.scope.Namespace)
	i.SetActor(actor)
	i.SetName(name)
	i.SetTargetBundleHash(b.Hash)
	i.SetBindingsHash(hash)
	i.SetExpectedRevision(int64(current.Revision))
	i.SetNoPending(true)
	i.SetStage([]*stageview.StageChannel{row})
	_, e = s.invoke(ctx, reflect.TypeFor[stageview.StageComponent](), "stage", "PATCH", i)
	if e != nil {
		return "", e
	}
	return id, nil
}

func validOperation(id string) bool {
	if len(id) != 32 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}
func (s *Store) intent(ctx context.Context, id string) (*intentview.ReleaseOperation, error) {
	if !s.ready(ctx) || !validOperation(id) {
		return nil, contract.ErrInvalid
	}
	i := &intentview.Input{}
	i.SetScopeId(s.scope.ID())
	i.SetOperationId(id)
	v, e := s.invoke(ctx, reflect.TypeFor[intentview.IntentComponent](), "intent", "GET", i)
	if e != nil {
		return nil, e
	}
	o, ok := v.(*intentview.Output)
	if !ok {
		return nil, contract.ErrInvalid
	}
	if len(o.Data) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(o.Data) != 1 || o.Data[0] == nil || str(o.Data[0].ScopeId) != s.scope.ID() || str(o.Data[0].OperationId) != id {
		return nil, contract.ErrInvalid
	}
	return o.Data[0], nil
}
func (s *Store) PendingDependencies(ctx context.Context, id string) ([]authz.NamedDependency, error) {
	r, e := s.intent(ctx, id)
	if e != nil {
		return nil, e
	}
	if str(r.Status) != "pending" {
		return nil, sql.ErrNoRows
	}
	if r.DependenciesJson == nil {
		return releasevalue.DecodeDependencies(nil)
	}
	return releasevalue.DecodeDependencies(*r.DependenciesJson)
}
func (s *Store) applied(ctx context.Context, key contract.Key, expected uint64, id string) (bool, error) {
	c, e := s.Channel(ctx, key)
	if e != nil {
		return false, e
	}
	if c.Revision != expected+1 {
		return false, nil
	}
	op, e := s.intent(ctx, id)
	if e != nil {
		return false, e
	}
	return str(op.ChannelName) == key.Name && str(op.Status) == "applied" && op.ExpectedRevision == int64(expected) && op.TargetRevision == int64(expected+1), nil
}

func (s *Store) CompleteRevision(ctx context.Context, key contract.Key, expected uint64, id string) error {
	if !s.ready(ctx) || !s.scope.Owns(key) || expected >= math.MaxInt64 || !validOperation(id) {
		return contract.ErrInvalid
	}

	actor, e := s.requestActor(ctx)
	if e != nil {
		return e
	}
	if done, err := s.applied(ctx, key, expected, id); err != nil {
		return err
	} else if done {
		return nil
	}
	now := s.clock().UTC()
	scope := s.scope.ID()
	row := &completeview.CompleteChannel{}
	row.SetScopeId(&scope)
	row.SetName(&key.Name)
	row.SetRevision(int64(expected))
	row.SetUpdatedAt(&now)
	row.SetUpdatedBy(&actor)
	op := &completeview.CompleteOperation{}
	op.SetOperationId(&id)
	op.SetScopeId(&scope)
	op.SetChannelName(&key.Name)
	op.SetUpdatedAt(&now)
	op.SetUpdatedBy(&actor)
	op.SetCompletedAt(&now)
	status := "applied"
	op.SetStatus(&status)
	row.SetOperation([]*completeview.CompleteOperation{op})
	i := &completeview.Input{}
	i.SetScopeId(scope)
	i.SetName(key.Name)
	i.SetOperationId(id)
	i.SetExpectedRevision(int64(expected))
	i.SetPendingStatus("pending")
	i.SetActor(actor)
	i.SetComplete([]*completeview.CompleteChannel{row})
	_, e = s.invoke(ctx, reflect.TypeFor[completeview.CompleteComponent](), "complete", "PATCH", i)
	if e != nil {
		if done, checkErr := s.applied(ctx, key, expected, id); checkErr == nil && done {
			return nil
		}
	}
	return e
}

func (s *Store) FailRevision(ctx context.Context, key contract.Key, expected uint64, id string, code string) error {
	if !s.ready(ctx) || !s.scope.Owns(key) || expected >= math.MaxInt64 || !validOperation(id) {
		return contract.ErrInvalid
	}
	if code == "" || len(code) > 64 {
		return contract.ErrInvalid
	}
	for _, c := range code {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return contract.ErrInvalid
		}
	}
	actor, e := s.requestActor(ctx)
	if e != nil {
		return e
	}
	now := s.clock().UTC()
	scope := s.scope.ID()
	row := &failview.FailChannel{}
	row.SetScopeId(&scope)
	row.SetName(&key.Name)
	row.SetRevision(int64(expected))
	row.SetUpdatedAt(&now)
	row.SetUpdatedBy(&actor)
	op := &failview.FailOperation{}
	op.SetOperationId(&id)
	op.SetScopeId(&scope)
	op.SetChannelName(&key.Name)
	op.SetUpdatedAt(&now)
	op.SetUpdatedBy(&actor)
	op.SetCompletedAt(&now)
	status := "failed"
	op.SetStatus(&status)
	op.SetFailureCode(&code)
	row.SetOperation([]*failview.FailOperation{op})
	i := &failview.Input{}
	i.SetScopeId(scope)
	i.SetName(key.Name)
	i.SetOperationId(id)
	i.SetExpectedRevision(int64(expected))
	i.SetPendingStatus("pending")
	i.SetActor(actor)
	i.SetFail([]*failview.FailChannel{row})
	_, e = s.invoke(ctx, reflect.TypeFor[failview.FailComponent](), "fail", "PATCH", i)
	return e
}

func (s *Store) PutBundle(ctx context.Context, b contract.Bundle) error {
	if !s.ready(ctx) || b.Namespace != s.scope.Namespace || contract.ValidateBundle(b) != nil {
		return contract.ErrInvalid
	}
	a, e := s.requestActor(ctx)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(b)
	if e != nil {
		return e
	}
	now, scope := s.clock().UTC(), s.scope.ID()
	j := json.RawMessage(raw)
	row := &putview.NewBundle{ScopeId: &scope, TenantId: &s.scope.Tenant, AccountId: &s.scope.Account, WorkspaceId: &s.scope.Workspace, Namespace: &s.scope.Namespace, BundleHash: &b.Hash, BundleJson: &j, CreatedAt: &now, CreatedBy: &a, UpdatedAt: &now, UpdatedBy: &a}
	i := &putview.Input{}
	i.SetScopeId(scope)
	i.SetActor(a)
	i.SetRecords([]*putview.NewBundle{row})
	_, e = s.invoke(ctx, reflect.TypeFor[putview.PutBundleComponent](), "put_bundle", "POST", i)
	if e != nil {
		prior, readErr := s.Bundle(ctx, b.Hash)
		if readErr == nil {
			if reflect.DeepEqual(prior, b) {
				return nil
			}
			return contract.ErrConflict
		}
	}
	return e
}
func (s *Store) CreateChannel(ctx context.Context, c contract.Channel) error {
	if !s.ready(ctx) || !s.scope.Owns(c.Key) || contract.ValidateChannel(c) != nil || c.Revision != 0 || c.ActiveBundleHash != "" {
		return contract.ErrInvalid
	}
	a, e := s.requestActor(ctx)
	if e != nil {
		return e
	}
	pol, e := json.Marshal(c.Policies)
	if e != nil {
		return e
	}
	j := json.RawMessage(pol)
	now, scope := s.clock().UTC(), s.scope.ID()
	row := &createview.NewChannel{ScopeId: &scope, TenantId: &s.scope.Tenant, AccountId: &s.scope.Account, WorkspaceId: &s.scope.Workspace, Namespace: &s.scope.Namespace, Name: &c.Key.Name, Description: &c.Description, PolicyRefsJson: &j, CreatedAt: &now, CreatedBy: &a, UpdatedAt: &now, UpdatedBy: &a}
	i := &createview.Input{}
	i.SetScopeId(scope)
	i.SetActor(a)
	i.SetRecords([]*createview.NewChannel{row})
	_, e = s.invoke(ctx, reflect.TypeFor[createview.CreateChannelComponent](), "create_channel", "POST", i)
	if e != nil {
		prior, readErr := s.Channel(ctx, c.Key)
		if readErr == nil {
			if reflect.DeepEqual(prior, c) {
				return nil
			}
			return contract.ErrConflict
		}
	}
	return e
}
func (s *Store) ListChannels(ctx context.Context) ([]contract.Channel, error) {
	if !s.ready(ctx) {
		return nil, contract.ErrInvalid
	}
	i := &channelsview.Input{}
	i.SetScopeId(s.scope.ID())
	v, e := s.invoke(ctx, reflect.TypeFor[channelsview.ChannelsComponent](), "channels", "GET", i)
	if e != nil {
		return nil, e
	}
	o, ok := v.(*channelsview.Output)
	if !ok {
		return nil, contract.ErrInvalid
	}
	result := make([]contract.Channel, 0, len(o.Data))
	for _, r := range o.Data {
		if r == nil {
			return nil, contract.ErrInvalid
		}
		key := contract.Key{Tenant: s.scope.Tenant, Account: s.scope.Account, Workspace: s.scope.Workspace, Namespace: s.scope.Namespace, Name: str(r.Name)}
		c, e := s.Channel(ctx, key)
		if e != nil {
			return nil, e
		}
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key.Name < result[j].Key.Name })
	return result, nil
}

func (s *Store) History(ctx context.Context, key contract.Key) ([]releasevalue.HistoryEntry, error) {
	if !s.ready(ctx) || !s.scope.Owns(key) {
		return nil, contract.ErrInvalid
	}
	i := &historyview.Input{}
	i.SetScopeId(s.scope.ID())
	i.SetName(key.Name)
	v, e := s.invoke(ctx, reflect.TypeFor[historyview.HistoryComponent](), "history", "GET", i)
	if e != nil {
		return nil, e
	}
	o, ok := v.(*historyview.Output)
	if !ok {
		return nil, contract.ErrInvalid
	}
	var result []releasevalue.HistoryEntry
	for _, r := range o.Data {
		if r == nil || str(r.ScopeId) != s.scope.ID() || str(r.ChannelName) != key.Name || !validOperation(str(r.OperationId)) || r.NextChannelJson == nil || r.CreatedAt == nil {
			return nil, contract.ErrInvalid
		}
		item := releasevalue.HistoryEntry{OperationID: str(r.OperationId), Status: str(r.Status), CreatedAt: *r.CreatedAt, CreatedBy: str(r.CreatedBy), CompletedAt: r.CompletedAt, FailureCode: str(r.FailureCode)}
		if releasevalue.StrictJSON(*r.NextChannelJson, &item.Next) != nil || contract.ValidateChannel(item.Next) != nil || item.Next.Key != key || (item.Status != "pending" && item.Status != "applied" && item.Status != "failed") {
			return nil, contract.ErrInvalid
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].OperationID < result[j].OperationID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}
func (s *Store) Pending(ctx context.Context) ([]contract.PendingRevision, error) {
	channels, e := s.ListChannels(ctx)
	if e != nil {
		return nil, e
	}
	result := make([]contract.PendingRevision, 0)
	for _, c := range channels {
		r, e := s.snapshot(ctx, c.Key)
		if e != nil {
			return nil, e
		}
		if r.PendingOperationId == nil {
			continue
		}
		current, e := channel(r)
		if e != nil {
			return nil, e
		}
		op := r.Operation
		if op == nil || str(op.ScopeId) != s.scope.ID() || str(op.ChannelName) != c.Key.Name || str(op.OperationId) != *r.PendingOperationId || !validOperation(*r.PendingOperationId) || str(op.Status) != "pending" || op.ExpectedRevision < 0 || op.TargetRevision != op.ExpectedRevision+1 || current.Revision != uint64(op.ExpectedRevision) || op.NextChannelJson == nil {
			return nil, contract.ErrInvalid
		}
		var next contract.Channel
		if releasevalue.StrictJSON(*op.NextChannelJson, &next) != nil || contract.ValidateChannel(next) != nil || next.Key != c.Key || next.Revision != uint64(op.TargetRevision) || next.ActiveBundleHash != str(op.TargetBundleHash) {
			return nil, contract.ErrInvalid
		}
		result = append(result, contract.PendingRevision{OperationID: *r.PendingOperationId, Current: current, Next: next, BundleHash: str(op.TargetBundleHash)})
	}
	return result, nil
}

var _ contract.ChannelStore = (*Store)(nil)

package artifactrelease

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	identity "github.com/viant/forge/backend/resource"
)

// ControllerConfig is supplied only by a trusted embedding host. Callbacks
// derive scope, editor authority, policy heads and closed artifact bytes from
// server-owned state; no browser field can supply those capabilities.
type ControllerConfig struct {
	Scope sqlstore.Scope
	// ProjectionTenant is the configured recipient identity tenant on the wire.
	// Empty keeps the storage tenant. Named policy heads and dependency records
	// always stay in Scope.Tenant.
	ProjectionTenant string
	Store            *sqlstore.Store
	Publisher        *artifactcohort.Publisher
	VerifyScope      func(context.Context) (sqlstore.Scope, error)
	Authorize        func(context.Context, string) error
	Candidates       func(context.Context) ([]contract.Candidate, error)
	Prepare          func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error)
	// CompileBindings derives exact native component pins from validated closed
	// bytes; client aliases never enter this companion record.
	CompileBindings func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error)
	// RestoreArtifacts loads exact previously published bytes from trusted
	// producer persistence during restart. It must not require a user lease or
	// discover mutable working resources.
	RestoreArtifacts func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error)
	// AvailableRoot checks the producer's committed publication/archive state,
	// independently of retained immutable bytes. It must work on restart.
	AvailableRoot    func(context.Context, artifactcohort.RootRef) error
	ValidateArtifact func(context.Context, *artifactcohort.Artifact) error
	NamedDocuments   func(context.Context, []authz.NamedReference) ([]authz.Document, error)
	Fence            func(context.Context, []authz.NamedDependency) error
	// AdmitAudience evaluates the named policy with the existing authz engine
	// and verified actor facts. Bounded decisions require the producer's own
	// data-scope binder; the controller never flattens them into a grant.
	AdmitAudience func(context.Context, contract.Channel, authz.NamedReference) (bool, error)
	Notify        func(context.Context)
}

type Controller struct {
	config          ControllerConfig
	gate            *releasegate.Gate
	syncMu          sync.Mutex
	syncFingerprint string
}

// ActiveSelection is an immutable committed release pointer. It is still not
// an actor decision: callers must evaluate the channel's named policies using
// the verified facts and preserve the selected source's row/entity guards.
type ActiveSelection struct {
	Channel      contract.Channel
	Bundle       contract.Bundle
	Bindings     []sqlstore.ComponentBinding
	Dependencies []authz.NamedDependency
}

type ActiveSnapshot struct {
	ManifestHash string
	Selections   []ActiveSelection
}

// ActiveSnapshot reads committed selections under the same gate as Publisher
// projection. The returned values are detached, so business execution may run
// without holding this lock. FenceSnapshot checks for a change before release.
func (c *Controller) ActiveSnapshot(ctx context.Context) (ActiveSnapshot, error) {
	if err := c.check(ctx, c.config.Scope.Namespace, "use"); err != nil {
		return ActiveSnapshot{}, err
	}
	var out ActiveSnapshot
	err := c.gate.Read(func() error {
		manifest, err := c.config.Publisher.Snapshot()
		if err != nil {
			return err
		}
		channels, err := c.config.Store.ListChannels(ctx)
		if err != nil {
			return err
		}
		out.ManifestHash = manifest.Hash
		for _, channel := range channels {
			if channel.ActiveBundleHash == "" {
				continue
			}
			bundle, readErr := c.config.Store.Bundle(ctx, channel.ActiveBundleHash)
			if readErr != nil {
				return readErr
			}
			if !manifestSelectsChannel(manifest, channel, bundle, c.projectionTenant()) {
				continue
			}
			available := true
			for _, root := range bundle.Roots {
				if c.config.AvailableRoot(ctx, root) != nil {
					available = false
					break
				}
			}
			if !available {
				continue
			}
			bindings, readErr := c.config.Store.BundleBindings(ctx, bundle)
			if readErr != nil {
				return readErr
			}
			dependencies, readErr := c.config.Store.ActiveDependencies(ctx, channel.Key)
			if readErr != nil {
				return readErr
			}
			if len(dependencies) == 0 {
				continue
			}
			if readErr = c.config.Fence(ctx, dependencies); readErr != nil {
				continue
			}
			out.Selections = append(out.Selections, ActiveSelection{Channel: channel, Bundle: bundle, Bindings: bindings, Dependencies: dependencies})
		}
		return nil
	})
	return out, err
}

func manifestSelectsChannel(manifest artifactcohort.Manifest, channel contract.Channel, bundle contract.Bundle, projectionTenant string) bool {
	for _, ref := range channel.Policies {
		name := contract.ReleaseCohortName(channel.Key, ref)
		matched := false
		for _, cohort := range manifest.Cohorts {
			if cohort.Tenant != projectionTenant || cohort.Name != name {
				continue
			}
			if len(cohort.Roots) != len(bundle.Roots) {
				return false
			}
			roots := map[string]artifactcohort.RootRef{}
			for _, root := range cohort.Roots {
				roots[root.URI] = root
			}
			for _, root := range bundle.Roots {
				if roots[root.URI] != root {
					return false
				}
			}
			matched = true
			break
		}
		if !matched {
			return false
		}
	}
	return true
}

func (c *Controller) FenceSnapshot(ctx context.Context, manifestHash string) error {
	if manifestHash == "" {
		return contract.ErrInvalid
	}
	if err := c.check(ctx, c.config.Scope.Namespace, "use"); err != nil {
		return err
	}
	return c.gate.Read(func() error {
		current, err := c.config.Publisher.Snapshot()
		if err != nil {
			return err
		}
		if current.Hash != manifestHash {
			return contract.ErrConflict
		}
		return nil
	})
}

// AdmitRoot admits only the exact revision selected by an active channel and
// its current named audience. The returned manifest hash fences data release.
func (c *Controller) AdmitRoot(ctx context.Context, root artifactcohort.RootRef) (string, error) {
	if c.config.AdmitAudience == nil {
		return "", contract.ErrInvalid
	}
	snapshot, err := c.ActiveSnapshot(ctx)
	if err != nil {
		return "", err
	}
	for _, selection := range snapshot.Selections {
		selected := false
		for _, candidate := range selection.Bundle.Roots {
			if candidate == root {
				selected = true
				break
			}
		}
		if !selected {
			continue
		}
		for _, ref := range selection.Channel.Policies {
			allowed, decisionErr := c.config.AdmitAudience(ctx, selection.Channel, ref)
			if decisionErr != nil {
				return "", decisionErr
			}
			if allowed {
				return snapshot.ManifestHash, nil
			}
		}
	}
	return "", contract.ErrInvalid
}

func (c *Controller) FenceRoot(ctx context.Context, root artifactcohort.RootRef, manifestHash string) error {
	if err := c.FenceSnapshot(ctx, manifestHash); err != nil {
		return err
	}
	current, err := c.AdmitRoot(ctx, root)
	if err != nil {
		return err
	}
	if current != manifestHash {
		return contract.ErrConflict
	}
	return nil
}

var _ Manager = (*Controller)(nil)

func NewController(config ControllerConfig) (*Controller, error) {
	if config.Store == nil || config.Publisher == nil || config.VerifyScope == nil || config.Authorize == nil || config.Candidates == nil || config.Prepare == nil || config.CompileBindings == nil || config.RestoreArtifacts == nil || config.AvailableRoot == nil || config.ValidateArtifact == nil || config.NamedDocuments == nil || config.Fence == nil || config.AdmitAudience == nil || !(contract.Key{Tenant: config.Scope.Tenant, Account: config.Scope.Account, Workspace: config.Scope.Workspace, Namespace: config.Scope.Namespace, Name: "scope"}).Valid() {
		return nil, contract.ErrInvalid
	}
	if config.ProjectionTenant != "" && !(contract.Key{Tenant: config.ProjectionTenant, Account: config.Scope.Account, Workspace: config.Scope.Workspace, Namespace: config.Scope.Namespace, Name: "scope"}).Valid() {
		return nil, contract.ErrInvalid
	}
	c := &Controller{config: config}
	gate, err := releasegate.New(config.Store, releasegate.Projection{Reconcile: c.reconcile, Snapshot: c.snapshot, Project: c.project, Restore: c.restore})
	if err != nil {
		return nil, err
	}
	c.gate = gate
	return c, nil
}

func (c *Controller) Gate() *releasegate.Gate              { return c.gate }
func (c *Controller) Publisher() *artifactcohort.Publisher { return c.config.Publisher }
func (c *Controller) projectionTenant() string {
	if c.config.ProjectionTenant != "" {
		return c.config.ProjectionTenant
	}
	return c.config.Scope.Tenant
}
func (c *Controller) Recover(ctx context.Context) error { return c.gate.Recover(ctx) }

// SyncOnce reconciles a second process's committed release changes. All SQL,
// policy-head, and source-availability checks happen off the serving path.
// Pending intents are intentionally ignored until startup recovery: their
// originating process may still be projecting them.
func (c *Controller) SyncOnce(ctx context.Context) (bool, error) {
	if c == nil || ctx == nil || ctx.Err() != nil {
		return false, contract.ErrInvalid
	}
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	fingerprint, err := c.committedFingerprint(ctx)
	if err != nil {
		c.gate.Block()
		return false, err
	}
	if fingerprint == c.syncFingerprint && c.gate.Read(func() error { return nil }) == nil {
		return false, nil
	}
	changed, err := c.gate.SyncCommitted(ctx)
	if err != nil {
		return false, err
	}
	c.syncFingerprint = fingerprint
	if changed && c.config.Notify != nil {
		c.config.Notify(ctx)
	}
	return changed, nil
}

// RunSync follows the host's bounded background reconciliation convention.
// A failed pass blocks release reads and is retried; cancellation stops the
// worker without depending on a data request or an MCP session.
func (c *Controller) RunSync(ctx context.Context, interval time.Duration, observe func(bool, error)) error {
	if c == nil || ctx == nil || interval <= 0 {
		return contract.ErrInvalid
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		changed, err := c.SyncOnce(ctx)
		if observe != nil {
			observe(changed, err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (c *Controller) committedFingerprint(ctx context.Context) (string, error) {
	channels, err := c.config.Store.ListChannels(ctx)
	if err != nil {
		return "", err
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].Key.Name < channels[j].Key.Name })
	type activeState struct {
		Channel     contract.Channel
		Roots       []bool
		PolicyFresh bool
	}
	states := make([]activeState, 0, len(channels))
	// Several channels may select the same immutable root or named policy head.
	// Share checks only within this background pass; never retain an allow/deny
	// across passes or use this memo on the serving path.
	rootAvailable := make(map[artifactcohort.RootRef]bool)
	policyFresh := make(map[string]bool)
	for _, channel := range channels {
		if channel.ActiveBundleHash == "" {
			continue
		}
		bundle, err := c.config.Store.Bundle(ctx, channel.ActiveBundleHash)
		if err != nil {
			return "", err
		}
		state := activeState{Channel: channel, Roots: make([]bool, len(bundle.Roots))}
		for i, root := range bundle.Roots {
			if available, found := rootAvailable[root]; found {
				state.Roots[i] = available
				continue
			}
			availabilityErr := c.config.AvailableRoot(ctx, root)
			if availabilityErr != nil && !expectedReleaseWithdrawal(availabilityErr) {
				return "", availabilityErr
			}
			state.Roots[i] = availabilityErr == nil
			rootAvailable[root] = state.Roots[i]
		}
		deps, err := c.config.Store.ActiveDependencies(ctx, channel.Key)
		if err != nil {
			return "", err
		}
		if len(deps) > 0 {
			canonical, marshalErr := json.Marshal(deps)
			if marshalErr != nil {
				return "", marshalErr
			}
			if fresh, found := policyFresh[string(canonical)]; found {
				state.PolicyFresh = fresh
				states = append(states, state)
				continue
			}
			fenceErr := c.config.Fence(ctx, deps)
			if fenceErr != nil && !expectedReleaseWithdrawal(fenceErr) {
				return "", fenceErr
			}
			state.PolicyFresh = fenceErr == nil
			policyFresh[string(canonical)] = state.PolicyFresh
		}
		states = append(states, state)
	}
	raw, err := json.Marshal(states)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func expectedReleaseWithdrawal(err error) bool {
	return errors.Is(err, contract.ErrInvalid) || errors.Is(err, contract.ErrConflict) ||
		errors.Is(err, authz.ErrDenied) || errors.Is(err, artifactcohort.ErrDenied) || errors.Is(err, identity.ErrResourceDenied) || errors.Is(err, sql.ErrNoRows)
}

// RefreshPolicies is a producer-owned post-commit hook for a named policy
// update/archive. It advances only affected channel revisions with the same
// exact bundle. An invalid new head withdraws only that channel's cohorts;
// persisted old dependencies keep API/runtime admission denied until retry.
func (c *Controller) RefreshPolicies(ctx context.Context, changed authz.Resource) error {
	if c == nil || ctx == nil || ctx.Err() != nil || changed.Kind != "policy" || changed.Tenant != c.config.Scope.Tenant || changed.ID == "" {
		return contract.ErrInvalid
	}
	channels, err := c.config.Store.ListChannels(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, current := range channels {
		if current.ActiveBundleHash == "" {
			continue
		}
		deps, readErr := c.config.Store.ActiveDependencies(ctx, current.Key)
		if readErr != nil {
			return readErr
		}
		affected := false
		for _, dep := range deps {
			if dep.Resource == changed {
				affected = true
				break
			}
		}
		if !affected {
			continue
		}
		bundle, readErr := c.config.Store.Bundle(ctx, current.ActiveBundleHash)
		if readErr != nil {
			return readErr
		}
		bindings, readErr := c.config.Store.BundleBindings(ctx, bundle)
		if readErr != nil {
			return readErr
		}
		next, readErr := contract.PreparePromotion(current, current.Revision, bundle)
		if readErr != nil {
			return readErr
		}
		compiled, compileErr := c.compile(ctx, next, bundle)
		if compileErr == nil {
			stage := func(ctx context.Context, old, new contract.Channel, selected contract.Bundle) (string, error) {
				return c.config.Store.StageRevisionWithBindingsAndDependencies(ctx, old, new, selected, bindings, compiled.Dependencies)
			}
			compileErr = c.gate.PromoteWithStage(ctx, current, next, bundle, nil, stage)
		}
		if compileErr != nil {
			if withdrawErr := c.withdrawDerived(ctx, current.Key); withdrawErr != nil {
				failures = append(failures, errors.Join(compileErr, withdrawErr))
				continue
			}
			failures = append(failures, compileErr)
		}
		if c.config.Notify != nil {
			c.config.Notify(ctx)
		}
	}
	return errors.Join(failures...)
}

func (c *Controller) withdrawDerived(ctx context.Context, key contract.Key) error {
	return c.gate.UpdateOrBlock(func() error {
		manifest, err := c.config.Publisher.Snapshot()
		if err != nil {
			return err
		}
		policies, cohorts, err := contract.MergeProjectedActivation(manifest, key, c.projectionTenant(), contract.Compilation{})
		if err != nil {
			return err
		}
		_, err = c.config.Publisher.ReplaceCatalog(ctx, policies, cohorts, manifest.Generation)
		return err
	})
}

func (c *Controller) check(ctx context.Context, namespace, action string) error {
	if c == nil || ctx == nil || ctx.Err() != nil || namespace != c.config.Scope.Namespace {
		return contract.ErrInvalid
	}
	actual, err := c.config.VerifyScope(ctx)
	if err != nil {
		return err
	}
	if actual != c.config.Scope {
		return contract.ErrInvalid
	}
	return c.config.Authorize(ctx, action)
}

func (c *Controller) List(ctx context.Context, in contract.ListInput) (contract.ListOutput, error) {
	if err := in.Validate(); err != nil {
		return contract.ListOutput{}, err
	}
	if err := c.check(ctx, in.Namespace, "list"); err != nil {
		return contract.ListOutput{}, err
	}
	out := contract.ListOutput{Channels: []contract.Result{}}
	err := c.gate.Read(func() error {
		manifest, manifestErr := c.config.Publisher.Snapshot()
		if manifestErr != nil {
			return manifestErr
		}
		channels, err := c.config.Store.ListChannels(ctx)
		if err != nil {
			return err
		}
		for _, channel := range channels {
			status, active := contract.StatusPending, false
			result := contract.Result{Channel: channel, Status: status, Activated: active}
			if channel.Revision > 0 && channel.ActiveBundleHash != "" {
				bundle, readErr := c.config.Store.Bundle(ctx, channel.ActiveBundleHash)
				if readErr != nil {
					return readErr
				}
				result.Bundle = &bundle
				status, active = contract.StatusActive, true
				if !manifestSelectsChannel(manifest, channel, bundle, c.projectionTenant()) {
					status, active = contract.StatusFailed, false
				}
				for _, root := range bundle.Roots {
					if c.config.AvailableRoot(ctx, root) != nil {
						status, active = contract.StatusFailed, false
						break
					}
				}
				deps, depErr := c.config.Store.ActiveDependencies(ctx, channel.Key)
				if depErr != nil {
					return depErr
				}
				if len(deps) == 0 || c.config.Fence(ctx, deps) != nil {
					status, active = contract.StatusFailed, false
				}
			}
			result.Status, result.Activated = status, active
			out.Channels = append(out.Channels, result)
		}
		return nil
	})
	return out, err
}

func (c *Controller) Candidates(ctx context.Context, in contract.CandidatesInput) (contract.CandidatesOutput, error) {
	if err := in.Validate(); err != nil {
		return contract.CandidatesOutput{}, err
	}
	if err := c.check(ctx, in.Namespace, "candidates"); err != nil {
		return contract.CandidatesOutput{}, err
	}
	out := contract.CandidatesOutput{Artifacts: []contract.Candidate{}}
	var generation uint64
	if err := c.gate.Read(func() error {
		generation = c.config.Publisher.Generation()
		return nil
	}); err != nil {
		return out, err
	}
	// The producer callback may read its own release admission index. Never
	// hold Gate's read lock across that callback: a queued reconciliation writer
	// would block a nested read while waiting for this read to finish.
	items, err := c.config.Candidates(ctx)
	if err != nil {
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, item := range items {
		if validateRootPartition(item.Root, c.config.Scope.Namespace) != nil || seen[artifactcohort.RootKey(item.Root)] {
			return out, contract.ErrInvalid
		}
		seen[artifactcohort.RootKey(item.Root)] = true
	}
	sortCandidates(items)
	if err = c.gate.Read(func() error {
		if c.config.Publisher.Generation() != generation {
			return contract.ErrConflict
		}
		return nil
	}); err != nil {
		return out, err
	}
	out.Artifacts = append(out.Artifacts, items...)
	return out, nil
}

func (c *Controller) Publish(ctx context.Context, in contract.PublishInput) (contract.Result, error) {
	if err := in.Validate(); err != nil {
		return contract.Result{}, err
	}
	if err := c.check(ctx, in.Namespace, "publish"); err != nil {
		return contract.Result{}, err
	}
	// Preparation is producer-side strict closure validation. RetainBatch only
	// stores exact historical bytes; it never advances a follow pointer.
	artifacts, err := c.config.Prepare(ctx, in.Roots)
	if err != nil {
		return contract.Result{}, err
	}
	if len(artifacts) != len(in.Roots) {
		return contract.Result{}, contract.ErrInvalid
	}
	requested := map[string]bool{}
	for _, ref := range in.Roots {
		requested[artifactcohort.RootKey(ref)] = true
	}
	for _, artifact := range artifacts {
		if artifact == nil || !requested[artifactcohort.RootKey(artifact.Root)] || artifactcohort.ValidateArtifact(artifact) != nil {
			return contract.Result{}, contract.ErrInvalid
		}
		if err := c.config.AvailableRoot(ctx, artifact.Root); err != nil {
			return contract.Result{}, err
		}
		delete(requested, artifactcohort.RootKey(artifact.Root))
	}
	if len(requested) != 0 {
		return contract.Result{}, contract.ErrInvalid
	}
	if _, err = c.config.Publisher.RetainBatch(ctx, artifacts, c.config.ValidateArtifact); err != nil {
		return contract.Result{}, err
	}
	bundle, err := contract.NewBundle(ctx, in.Namespace, in.Roots, func(_ context.Context, ref artifactcohort.RootRef) (*artifactcohort.Artifact, error) {
		return c.config.Publisher.Get(ref)
	})
	if err != nil {
		return contract.Result{}, err
	}
	bindings, err := c.config.CompileBindings(ctx, bundle, artifacts)
	if err != nil {
		return contract.Result{}, err
	}
	if err = c.config.Store.PutBundle(ctx, bundle); err != nil {
		return contract.Result{}, err
	}
	key := c.key(in.Name)
	current, err := c.config.Store.Channel(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		if in.ExpectedRevision != 0 {
			return contract.Result{}, contract.ErrConflict
		}
		current, err = contract.NewChannel(key, in.Description, in.Policies)
		if err != nil {
			return contract.Result{}, err
		}
		if err = c.config.Store.CreateChannel(ctx, current); err != nil {
			return contract.Result{}, err
		}
	} else if err != nil {
		return contract.Result{}, err
	}
	next, err := contract.PrepareChannelUpdate(current, in.ExpectedRevision, in.Description, in.Policies, bundle)
	if err != nil {
		return contract.Result{}, err
	}
	compiled, err := c.compile(ctx, next, bundle)
	if err != nil {
		return contract.Result{}, err
	}
	stage := func(ctx context.Context, current, next contract.Channel, bundle contract.Bundle) (string, error) {
		return c.config.Store.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, bindings, compiled.Dependencies)
	}
	if err = c.gate.PromoteWithStage(ctx, current, next, bundle, func(ctx context.Context) error { return c.check(ctx, in.Namespace, "publish") }, stage); err != nil {
		return contract.Result{}, err
	}
	if c.config.Notify != nil {
		c.config.Notify(ctx)
	}
	return contract.Result{Channel: next, Bundle: &bundle, Status: contract.StatusActive, Activated: true}, nil
}

func (c *Controller) Promote(ctx context.Context, in contract.PromoteInput) (contract.Result, error) {
	if err := in.Validate(); err != nil {
		return contract.Result{}, err
	}
	if err := c.check(ctx, in.Namespace, "promote"); err != nil {
		return contract.Result{}, err
	}
	bundle, err := c.config.Store.Bundle(ctx, in.BundleHash)
	if err != nil {
		return contract.Result{}, err
	}
	bindings, err := c.config.Store.BundleBindings(ctx, bundle)
	if err != nil {
		return contract.Result{}, err
	}
	current, err := c.config.Store.Channel(ctx, c.key(in.Name))
	if err != nil {
		return contract.Result{}, err
	}
	next, err := contract.PreparePromotion(current, in.ExpectedRevision, bundle)
	if err != nil {
		return contract.Result{}, err
	}
	compiled, err := c.compile(ctx, next, bundle)
	if err != nil {
		return contract.Result{}, err
	}
	stage := func(ctx context.Context, current, next contract.Channel, bundle contract.Bundle) (string, error) {
		return c.config.Store.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, bindings, compiled.Dependencies)
	}
	if err = c.gate.PromoteWithStage(ctx, current, next, bundle, func(ctx context.Context) error { return c.check(ctx, in.Namespace, "promote") }, stage); err != nil {
		return contract.Result{}, err
	}
	if c.config.Notify != nil {
		c.config.Notify(ctx)
	}
	return contract.Result{Channel: next, Bundle: &bundle, Status: contract.StatusActive, Activated: true}, nil
}

func (c *Controller) History(ctx context.Context, in contract.HistoryInput) (contract.HistoryOutput, error) {
	if err := in.Validate(); err != nil {
		return contract.HistoryOutput{}, err
	}
	if err := c.check(ctx, in.Namespace, "history"); err != nil {
		return contract.HistoryOutput{}, err
	}
	out := contract.HistoryOutput{Entries: []contract.HistoryEntry{}}
	err := c.gate.Read(func() error {
		items, err := c.config.Store.History(ctx, c.key(in.Name))
		if err != nil {
			return err
		}
		for _, item := range items {
			status := item.Status
			if status != contract.StatusActive {
				if status == "applied" {
					status = contract.StatusActive
				}
			}
			created := item.CreatedAt
			out.Entries = append(out.Entries, contract.HistoryEntry{Revision: item.Next.Revision, BundleHash: item.Next.ActiveBundleHash, Description: item.Next.Description, Status: status, CreatedAt: &created})
		}
		return nil
	})
	return out, err
}

func (c *Controller) key(name string) contract.Key {
	s := c.config.Scope
	return contract.Key{Tenant: s.Tenant, Account: s.Account, Workspace: s.Workspace, Namespace: s.Namespace, Name: name}
}

func (c *Controller) snapshot(context.Context) (any, error) { return c.config.Publisher.Snapshot() }
func (c *Controller) restore(ctx context.Context, prior any) error {
	manifest, ok := prior.(artifactcohort.Manifest)
	if !ok {
		return contract.ErrInvalid
	}
	current, err := c.config.Publisher.Snapshot()
	if err != nil {
		return err
	}
	_, err = c.config.Publisher.ReplaceCatalog(ctx, manifest.Policies, manifest.Cohorts, current.Generation)
	return err
}
func (c *Controller) compile(ctx context.Context, channel contract.Channel, bundle contract.Bundle) (contract.Compilation, error) {
	docs, err := c.config.NamedDocuments(ctx, channel.Policies)
	if err != nil {
		return contract.Compilation{}, err
	}
	result, err := contract.CompileCohorts(ctx, channel, bundle, docs)
	if err != nil {
		return contract.Compilation{}, err
	}
	if err = c.config.Fence(ctx, result.Dependencies); err != nil {
		return contract.Compilation{}, err
	}
	if target := c.projectionTenant(); target != channel.Key.Tenant {
		for i := range result.Policies {
			result.Policies[i].Resource.Tenant = target
		}
		for i := range result.Cohorts {
			result.Cohorts[i].Tenant = target
			result.Cohorts[i].PolicyResource.Tenant = target
		}
	}
	return result, nil
}

func (c *Controller) retainRoots(ctx context.Context, roots []artifactcohort.RootRef) error {
	var missing []artifactcohort.RootRef
	for _, root := range roots {
		artifact, err := c.config.Publisher.Get(root)
		if err == nil && artifactcohort.ValidateArtifact(artifact) == nil && artifact.Root == root {
			continue
		}
		missing = append(missing, root)
	}
	if len(missing) == 0 {
		return nil
	}
	artifacts, err := c.config.RestoreArtifacts(ctx, missing)
	if err != nil {
		return err
	}
	if len(artifacts) != len(missing) {
		return contract.ErrInvalid
	}
	expected := map[string]artifactcohort.RootRef{}
	for _, root := range missing {
		expected[artifactcohort.RootKey(root)] = root
	}
	for _, artifact := range artifacts {
		if artifact == nil || expected[artifactcohort.RootKey(artifact.Root)] != artifact.Root || artifactcohort.ValidateArtifact(artifact) != nil {
			return contract.ErrInvalid
		}
		delete(expected, artifactcohort.RootKey(artifact.Root))
	}
	if len(expected) != 0 {
		return contract.ErrInvalid
	}
	_, err = c.config.Publisher.RetainBatch(ctx, artifacts, c.config.ValidateArtifact)
	return err
}
func (c *Controller) project(ctx context.Context, item contract.PendingRevision) error {
	bundle, err := c.config.Store.Bundle(ctx, item.BundleHash)
	if err != nil {
		return err
	}
	if _, err = c.config.Store.BundleBindings(ctx, bundle); err != nil {
		return err
	}
	if err = c.retainRoots(ctx, bundle.Roots); err != nil {
		return err
	}
	compiled, err := c.compile(ctx, item.Next, bundle)
	if err != nil {
		return err
	}
	stagedDeps, err := c.config.Store.PendingDependencies(ctx, item.OperationID)
	if err != nil || !reflect.DeepEqual(stagedDeps, compiled.Dependencies) {
		return contract.ErrConflict
	}
	manifest, err := c.config.Publisher.Snapshot()
	if err != nil {
		return err
	}
	policies, cohorts, err := contract.MergeProjectedActivation(manifest, item.Next.Key, c.projectionTenant(), compiled)
	if err != nil {
		return err
	}
	if _, err = c.config.Publisher.ReplaceCatalog(ctx, policies, cohorts, manifest.Generation); err != nil {
		return err
	}
	return c.config.Fence(ctx, compiled.Dependencies)
}
func (c *Controller) reconcile(ctx context.Context) error {
	channels, err := c.config.Store.ListChannels(ctx)
	if err != nil {
		return err
	}
	manifest, err := c.config.Publisher.Snapshot()
	if err != nil {
		return err
	}
	for _, channel := range channels {
		compiled := contract.Compilation{}
		if channel.ActiveBundleHash != "" {
			bundle, readErr := c.config.Store.Bundle(ctx, channel.ActiveBundleHash)
			if readErr != nil {
				return readErr
			}
			if _, readErr = c.config.Store.BundleBindings(ctx, bundle); readErr != nil {
				return readErr
			}
			available := true
			for _, root := range bundle.Roots {
				if availabilityErr := c.config.AvailableRoot(ctx, root); availabilityErr != nil {
					if !expectedReleaseWithdrawal(availabilityErr) {
						return availabilityErr
					}
					available = false
					break
				}
			}
			if available {
				if readErr = c.retainRoots(ctx, bundle.Roots); readErr != nil {
					return readErr
				}
				compiled, readErr = c.compile(ctx, channel, bundle)
				if readErr != nil {
					if !errors.Is(readErr, contract.ErrConflict) && !errors.Is(readErr, contract.ErrInvalid) && !errors.Is(readErr, authz.ErrDenied) && !errors.Is(readErr, sql.ErrNoRows) {
						return readErr
					}
					compiled = contract.Compilation{}
				} else {
					activeDeps, depErr := c.config.Store.ActiveDependencies(ctx, channel.Key)
					if depErr != nil {
						return depErr
					}
					if !reflect.DeepEqual(activeDeps, compiled.Dependencies) {
						compiled = contract.Compilation{}
					}
				}
			}
		}
		manifest.Policies, manifest.Cohorts, err = contract.MergeProjectedActivation(manifest, channel.Key, c.projectionTenant(), compiled)
		if err != nil {
			return err
		}
		manifest.Hash = ""
	}
	if len(channels) == 0 {
		return nil
	}
	current, err := c.config.Publisher.Snapshot()
	if err != nil {
		return err
	}
	// ReplaceCatalog orders cohorts before hashing. Apply that same canonical
	// order here so a multi-channel no-op does not advance generation or emit a
	// second notification simply because channels were scanned by name.
	sort.Slice(manifest.Cohorts, func(i, j int) bool {
		return manifest.Cohorts[i].Tenant+"|"+manifest.Cohorts[i].Name < manifest.Cohorts[j].Tenant+"|"+manifest.Cohorts[j].Name
	})
	// Publisher recomputes the two cohort hashes from the resolved policies and
	// selected roots. The merge candidate carries empty derived hashes, so
	// compare the authoritative inputs after confirming named dependencies
	// above; otherwise every unchanged multi-channel pass advances generation.
	currentCohorts := append([]artifactcohort.Cohort(nil), current.Cohorts...)
	for i := range currentCohorts {
		currentCohorts[i].Hash, currentCohorts[i].PolicyHash = "", ""
	}
	for i := range manifest.Cohorts {
		manifest.Cohorts[i].Hash, manifest.Cohorts[i].PolicyHash = "", ""
	}
	samePolicies := len(current.Policies) == 0 && len(manifest.Policies) == 0 || reflect.DeepEqual(current.Policies, manifest.Policies)
	sameCohorts := len(currentCohorts) == 0 && len(manifest.Cohorts) == 0 || reflect.DeepEqual(currentCohorts, manifest.Cohorts)
	if samePolicies && sameCohorts {
		return nil
	}
	_, err = c.config.Publisher.ReplaceCatalog(ctx, manifest.Policies, manifest.Cohorts, current.Generation)
	return err
}

func validateRootPartition(ref artifactcohort.RootRef, namespace string) error {
	partition, err := artifactcohort.RootPartition(ref)
	if err != nil || ref.Version == 0 || len(ref.Hash) != 64 || partition != artifactcohort.PartitionKey(namespace, ref.Kind) {
		return contract.ErrInvalid
	}
	return nil
}

func sortCandidates(items []contract.Candidate) {
	sort.Slice(items, func(i, j int) bool { return items[i].Root.URI < items[j].Root.URI })
}

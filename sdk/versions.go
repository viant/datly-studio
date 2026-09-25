package sdk

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OperationVersionCreate       = "versions.create"
	OperationVersionGet          = "versions.get"
	OperationVersionList         = "versions.list"
	OperationVersionApply        = "versions.apply"
	OperationVersionValidate     = "versions.validate"
	OperationVersionDescriptor   = "versions.descriptor"
	OperationVersionExportDQL    = "versions.export_dql"
	OperationVersionInspect      = "versions.inspect"
	OperationVersionBuilder      = "versions.builder"
	OperationVersionTestView     = "versions.test_view"
	OperationVersionTestRelation = "versions.test_relation"
	OperationVersionTestCompose  = "versions.test_compose"
	OperationVersionWarmup       = "versions.warmup"
	OperationVersionWarmupGet    = "versions.warmup_get"
	OperationVersionWarmupList   = "versions.warmup_list"
)

type ReportVersion struct {
	ReportID            string          `json:"reportId"`
	VersionNo           int             `json:"versionNo"`
	State               string          `json:"state"`
	AuthoringMode       string          `json:"authoringMode"`
	AuthoredSQL         string          `json:"authoredSql,omitempty"`
	AuthoredDQL         string          `json:"authoredDql,omitempty"`
	ComponentSpec       json.RawMessage `json:"componentSpec,omitempty"`
	DQLExportLimits     json.RawMessage `json:"dqlExportLimits,omitempty"`
	TypeManifest        json.RawMessage `json:"typeManifest,omitempty"`
	ResourceManifest    json.RawMessage `json:"resourceManifest,omitempty"`
	ComponentDescriptor json.RawMessage `json:"componentDescriptor,omitempty"`
	SpecFormatVersion   string          `json:"specFormatVersion"`
	SpecHash            string          `json:"specHash"`
	GeneratedDQL        string          `json:"generatedDql,omitempty"`
	CompileStatus       string          `json:"compileStatus"`
	CompileDiagnostics  json.RawMessage `json:"compileDiagnostics,omitempty"`
	DatlyVersion        string          `json:"datlyVersion"`
	CompilerVersion     string          `json:"compilerVersion"`
	SourceRevision      int64           `json:"sourceRevision"`
	Notes               string          `json:"notes,omitempty"`
	CreatedBy           string          `json:"createdBy"`
	CreatedAt           time.Time       `json:"createdAt"`
	ValidatedAt         *time.Time      `json:"validatedAt,omitempty"`
	PublishedAt         *time.Time      `json:"publishedAt,omitempty"`
}

type CreateVersionInput struct {
	AuthoringMode string          `json:"authoringMode"`
	AuthoredSQL   string          `json:"authoredSql,omitempty"`
	AuthoredDQL   string          `json:"authoredDql,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	CreatedBy     string          `json:"createdBy,omitempty"`
	ComponentSpec json.RawMessage `json:"componentSpec,omitempty"`
}
type ListVersionsInput struct {
	State         string   `json:"state,omitempty"`
	AuthoringMode string   `json:"authoringMode,omitempty"`
	CompileStatus string   `json:"compileStatus,omitempty"`
	CreatedBy     string   `json:"createdBy,omitempty"`
	OrderBy       string   `json:"orderBy,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	Offset        int      `json:"offset,omitempty"`
}
type VersionPage struct {
	Items  []*ReportVersion `json:"items"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// EditCommand requires the positive ExpectedSourceRevision of the version being edited.
type EditCommand struct {
	Kind                   string          `json:"kind"`
	ExpectedSourceRevision int64           `json:"expectedSourceRevision"`
	Payload                json.RawMessage `json:"payload"`
}
type EditResult struct {
	Version     *ReportVersion `json:"version"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
}
type ValidationResult struct {
	Valid       bool           `json:"valid"`
	Version     *ReportVersion `json:"version"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
}
type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}
type ComponentDescriptor struct {
	Component json.RawMessage `json:"component"`
	Types     json.RawMessage `json:"types,omitempty"`
	Resources json.RawMessage `json:"resources,omitempty"`
}
type DQLExport struct {
	DQL         string   `json:"dql"`
	Complete    bool     `json:"complete"`
	Limitations []string `json:"limitations,omitempty"`
}

// ReaderInspection is the stable Studio SDK envelope around Datly's stateless
// reader builder. Structure and Operation use JSON because their nested DQL
// shape evolves with Datly; versioning and authorization stay in Studio.
type ReaderInspection struct {
	Version      *ReportVersion     `json:"version"`
	DQL          string             `json:"dql,omitempty"`
	Structure    json.RawMessage    `json:"structure,omitempty"`
	Diagnostics  []Diagnostic       `json:"diagnostics,omitempty"`
	Capabilities ReportCapabilities `json:"capabilities"`
}

type ReportCapabilities struct {
	CanManageACL bool `json:"canManageAcl"`
	CanView      bool `json:"canView"`
	CanRun       bool `json:"canRun"`
	CanEdit      bool `json:"canEdit"`
	CanPublish   bool `json:"canPublish"`
	CanUseDQL    bool `json:"canUseDql"`
}

// ReaderBuilderCommand requires the positive ExpectedSourceRevision of the version being edited.
type ReaderBuilderCommand struct {
	ExpectedSourceRevision int64           `json:"expectedSourceRevision"`
	Operation              json.RawMessage `json:"operation"`
}

type ReaderBuilderResult struct {
	Applied    bool              `json:"applied"`
	Inspection *ReaderInspection `json:"inspection"`
}

type ViewTestInput struct {
	Input json.RawMessage `json:"input,omitempty"`
	Limit int             `json:"limit,omitempty"`
}

type ViewTestResult struct {
	View        string            `json:"view"`
	Data        json.RawMessage   `json:"data,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Diagnostics []Diagnostic      `json:"diagnostics,omitempty"`
	Evidence    ExecutionEvidence `json:"evidence"`
}

type RelationKeyEvidence struct {
	ParentColumn string `json:"parentColumn"`
	ChildColumn  string `json:"childColumn"`
}

type RelationTestResult struct {
	Relation         string                `json:"relation"`
	ParentView       string                `json:"parentView"`
	ChildView        string                `json:"childView"`
	Kind             string                `json:"kind"`
	Cardinality      string                `json:"cardinality"`
	Keys             []RelationKeyEvidence `json:"keys,omitempty"`
	ParentRows       int                   `json:"parentRows"`
	MatchedParents   int                   `json:"matchedParents"`
	UnmatchedParents int                   `json:"unmatchedParents"`
	AttachedChildren int                   `json:"attachedChildren"`
	Data             json.RawMessage       `json:"data,omitempty"`
	Duration         time.Duration         `json:"duration"`
	Evidence         ExecutionEvidence     `json:"evidence"`
	Diagnostics      []Diagnostic          `json:"diagnostics,omitempty"`
}
type CubeComposeTestInput struct {
	Cubes []json.RawMessage `json:"cubes"`
	SQL   string            `json:"sql"`
}
type CubeComposeTestResult struct {
	Data        json.RawMessage `json:"data,omitempty"`
	Duration    time.Duration   `json:"duration"`
	Diagnostics []Diagnostic    `json:"diagnostics,omitempty"`
}

type WarmupTarget struct {
	View           string `json:"view"`
	CacheName      string `json:"cacheName,omitempty"`
	CacheProvider  string `json:"cacheProvider,omitempty"`
	ConnectorName  string `json:"connectorName,omitempty"`
	IndexColumn    string `json:"indexColumn,omitempty"`
	IndexParameter string `json:"indexParameter,omitempty"`
}

// WarmupRun is durable evidence from one server-owned Datly cache warmup run.
type WarmupRun struct {
	RunID          string        `json:"runId"`
	ReportID       string        `json:"reportId"`
	VersionNo      int           `json:"versionNo"`
	SourceRevision int64         `json:"sourceRevision"`
	SpecHash       string        `json:"specHash"`
	PlanKey        string        `json:"planKey"`
	Status         string        `json:"status"`
	RequestedBy    string        `json:"requestedBy"`
	RequestedAt    time.Time     `json:"requestedAt"`
	CreatedAt      *time.Time    `json:"createdAt,omitempty"`
	CreatedBy      *string       `json:"createdBy,omitempty"`
	UpdatedAt      *time.Time    `json:"updatedAt,omitempty"`
	UpdatedBy      *string       `json:"updatedBy,omitempty"`
	StartedAt      *time.Time    `json:"startedAt,omitempty"`
	CompletedAt    *time.Time    `json:"completedAt,omitempty"`
	PlannedCases   int           `json:"plannedCases"`
	CompletedCases int           `json:"completedCases"`
	MaxCases       *int          `json:"maxCases,omitempty"`
	RowLimit       *int          `json:"rowLimit,omitempty"`
	Entries        int           `json:"entries"`
	Duration       time.Duration `json:"duration,omitempty"`
	Target         WarmupTarget  `json:"target"`
	Diagnostics    []Diagnostic  `json:"diagnostics,omitempty"`
}

type WarmupResult = WarmupRun

type ListWarmupRunsInput struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type WarmupRunPage struct {
	Items  []*WarmupRun `json:"items"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type VersionService interface {
	Download(context.Context, string, int) (*ComponentDownload, error)
	LoadDQL(context.Context, string, LoadDQLInput) (*DQLLoadResult, error)
	LoadArchive(context.Context, string, LoadArchiveInput) (*DQLLoadResult, error)
	Create(context.Context, string, CreateVersionInput) (*ReportVersion, error)
	Get(context.Context, string, int) (*ReportVersion, error)
	List(context.Context, string, ListVersionsInput) (*VersionPage, error)
	Apply(context.Context, string, int, EditCommand) (*EditResult, error)
	Validate(context.Context, string, int) (*ValidationResult, error)
	Descriptor(context.Context, string, int) (*ComponentDescriptor, error)
	ExportDQL(context.Context, string, int) (*DQLExport, error)
	Inspect(context.Context, string, int) (*ReaderInspection, error)
	ApplyReaderCommand(context.Context, string, int, ReaderBuilderCommand) (*ReaderBuilderResult, error)
	TestView(context.Context, string, int, string, ViewTestInput) (*ViewTestResult, error)
	TestRelation(context.Context, string, int, string, ViewTestInput) (*RelationTestResult, error)
	TestCompose(context.Context, string, int, CubeComposeTestInput) (*CubeComposeTestResult, error)
	Warmup(context.Context, string, int) (*WarmupResult, error)
	WarmupRun(context.Context, string, string) (*WarmupRun, error)
	ListWarmupRuns(context.Context, string, int, ListWarmupRunsInput) (*WarmupRunPage, error)
}

type versionClient struct{ transport Transport }
type versionIdentity struct {
	ReportID  string `json:"reportId"`
	VersionNo int    `json:"versionNo"`
}

func (c versionClient) Create(ctx context.Context, id string, input CreateVersionInput) (*ReportVersion, error) {
	return invoke[ReportVersion](ctx, c.transport, OperationVersionCreate, struct {
		ReportID string             `json:"reportId"`
		Input    CreateVersionInput `json:"input"`
	}{id, input})
}
func (c versionClient) Get(ctx context.Context, id string, v int) (*ReportVersion, error) {
	return invoke[ReportVersion](ctx, c.transport, OperationVersionGet, versionIdentity{id, v})
}
func (c versionClient) List(ctx context.Context, id string, input ListVersionsInput) (*VersionPage, error) {
	return invoke[VersionPage](ctx, c.transport, OperationVersionList, struct {
		ReportID string            `json:"reportId"`
		Input    ListVersionsInput `json:"input"`
	}{id, input})
}
func (c versionClient) Apply(ctx context.Context, id string, v int, cmd EditCommand) (*EditResult, error) {
	return invoke[EditResult](ctx, c.transport, OperationVersionApply, struct {
		versionIdentity
		Command EditCommand `json:"command"`
	}{versionIdentity{id, v}, cmd})
}
func (c versionClient) Validate(ctx context.Context, id string, v int) (*ValidationResult, error) {
	return invoke[ValidationResult](ctx, c.transport, OperationVersionValidate, versionIdentity{id, v})
}
func (c versionClient) Descriptor(ctx context.Context, id string, v int) (*ComponentDescriptor, error) {
	return invoke[ComponentDescriptor](ctx, c.transport, OperationVersionDescriptor, versionIdentity{id, v})
}
func (c versionClient) ExportDQL(ctx context.Context, id string, v int) (*DQLExport, error) {
	return invoke[DQLExport](ctx, c.transport, OperationVersionExportDQL, versionIdentity{id, v})
}
func (c versionClient) Inspect(ctx context.Context, id string, v int) (*ReaderInspection, error) {
	return invoke[ReaderInspection](ctx, c.transport, OperationVersionInspect, versionIdentity{id, v})
}
func (c versionClient) ApplyReaderCommand(ctx context.Context, id string, v int, command ReaderBuilderCommand) (*ReaderBuilderResult, error) {
	return invoke[ReaderBuilderResult](ctx, c.transport, OperationVersionBuilder, struct {
		versionIdentity
		Command ReaderBuilderCommand `json:"command"`
	}{versionIdentity{id, v}, command})
}
func (c versionClient) TestView(ctx context.Context, id string, v int, view string, input ViewTestInput) (*ViewTestResult, error) {
	return invoke[ViewTestResult](ctx, c.transport, OperationVersionTestView, struct {
		versionIdentity
		View  string        `json:"view"`
		Input ViewTestInput `json:"input"`
	}{versionIdentity{id, v}, view, input})
}
func (c versionClient) TestRelation(ctx context.Context, id string, v int, relation string, input ViewTestInput) (*RelationTestResult, error) {
	return invoke[RelationTestResult](ctx, c.transport, OperationVersionTestRelation, struct {
		versionIdentity
		Relation string        `json:"relation"`
		Input    ViewTestInput `json:"input"`
	}{versionIdentity{id, v}, relation, input})
}
func (c versionClient) TestCompose(ctx context.Context, id string, v int, input CubeComposeTestInput) (*CubeComposeTestResult, error) {
	return invoke[CubeComposeTestResult](ctx, c.transport, OperationVersionTestCompose, struct {
		versionIdentity
		Input CubeComposeTestInput `json:"input"`
	}{versionIdentity{id, v}, input})
}
func (c versionClient) Warmup(ctx context.Context, id string, v int) (*WarmupResult, error) {
	return invoke[WarmupResult](ctx, c.transport, OperationVersionWarmup, versionIdentity{id, v})
}

func (c versionClient) WarmupRun(ctx context.Context, id, runID string) (*WarmupRun, error) {
	return invoke[WarmupRun](ctx, c.transport, OperationVersionWarmupGet, struct {
		ReportID string `json:"reportId"`
		RunID    string `json:"runId"`
	}{id, runID})
}

func (c versionClient) ListWarmupRuns(ctx context.Context, id string, v int, input ListWarmupRunsInput) (*WarmupRunPage, error) {
	return invoke[WarmupRunPage](ctx, c.transport, OperationVersionWarmupList, struct {
		versionIdentity
		Input ListWarmupRunsInput `json:"input"`
	}{versionIdentity{id, v}, input})
}

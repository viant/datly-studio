package sqltransport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	imported "github.com/viant/datly-studio/studio/report_versions/store_import"
	pointer "github.com/viant/datly-studio/studio/reports/store_draft_pointer"
	xhandler "github.com/viant/xdatly/handler"
)

const (
	importSpecFormatVersion = "studio.v1"
	importDatlyVersion      = "v1"
	importCompilerVersion   = "studio.v1"
)

func (t *Transport) loadDQL(ctx context.Context, operation string, input, output any) error {
	// DQL access and component editing are separate grants.
	if err := t.authorize(ctx, sdk.OperationVersionCreate, input); err != nil {
		return err
	}
	var request struct {
		ReportID string `json:"reportId"`
		Input    struct {
			DQL      string `json:"dql"`
			Archive  []byte `json:"archive"`
			Format   string `json:"format"`
			EntryDQL string `json:"entryDql"`
			Notes    string `json:"notes"`
		} `json:"input"`
	}
	if err := decode(input, &request); err != nil {
		return invalid(err)
	}
	var bundle *sdk.DQLBundle
	var err error
	if operation == sdk.OperationVersionLoadDQL {
		bundle, err = sdk.ReadDQL(strings.NewReader(request.Input.DQL))
	} else {
		bundle, err = sdk.ReadDQLArchive(bytes.NewReader(request.Input.Archive), request.Input.Format)
	}
	if err != nil {
		return invalid(err)
	}
	entry := request.Input.EntryDQL
	if entry == "" && len(bundle.Entries) == 1 {
		entry = bundle.Entries[0]
	}
	found := false
	for _, candidate := range bundle.Entries {
		if candidate == entry {
			found = true
		}
	}
	if !found {
		return invalid(fmt.Errorf("entryDql must select a root DQL document: %v", bundle.Entries))
	}
	report, err := t.getReportValue(ctx, request.ReportID)
	if err != nil {
		return err
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	files := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		files = append(files, name)
	}
	sort.Strings(files)
	next, err := t.nextVersionNo(ctx, report.ID)
	if err != nil {
		return internal(err)
	}
	// The import transaction spans the version, its resource files and the
	// report draft pointer. Both Datly writers join it and only this method
	// commits, so a failed file insert or a stale report etag rolls back all.
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer tx.Rollback()
	now := t.now()
	source := string(bundle.Files[entry])
	version := importedVersionRow(report.ID, next, source, importSpecHash(report.ID, next, source, bundle, files), request.Input.Notes, principal.Subject, now)
	namespace := importNamespace(report)
	for _, name := range files {
		version.File = append(version.File, importedResourceFileRow(namespace, name, bundle.Files[name], now))
	}
	version.Has.File = len(version.File) > 0
	if err = t.writeImportedVersion(ctx, tx, version); err != nil {
		return classify(err, "report version", fmt.Sprintf("%s/%d", report.ID, next))
	}
	if err = t.writeDraftPointer(ctx, tx, draftPointerRow(report, next, now)); err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: fmt.Sprintf("report %q was modified concurrently", report.ID), Cause: err}
		}
		return internal(err)
	}
	if err = tx.Commit(); err != nil {
		return internal(err)
	}
	value, err := t.getVersionValue(ctx, report.ID, next)
	if err != nil {
		return err
	}
	return assign(output, &sdk.DQLLoadResult{Version: value, EntryDQL: entry, Entries: bundle.Entries, Files: files})
}

// importSpecHash binds the version hash to the whole bundle, not only the
// entry document, so two imports with different dependencies never collide on
// the (report_id, spec_hash) uniqueness rule.
func importSpecHash(reportID string, versionNo int, source string, bundle *sdk.DQLBundle, files []string) string {
	bundleHash := sha256.New()
	for _, name := range files {
		fmt.Fprintf(bundleHash, "%d:%s:%d:", len(name), name, len(bundle.Files[name]))
		bundleHash.Write(bundle.Files[name])
	}
	return hashVersion(reportID, versionNo, "dql", "", source, []byte(fmt.Sprintf(`{"bundleSha256":"%x"}`, bundleHash.Sum(nil))))
}

func importNamespace(report *sdk.Report) string {
	digest := sha256.Sum256([]byte(report.ID))
	return fmt.Sprintf("%s.imports.%x", report.OwnerPackage, digest[:8])
}

func importedVersionRow(reportID string, versionNo int, source, specHash, notes, createdBy string, createdAt time.Time) *imported.ImportedVersion {
	authored, generated := source, source
	row := &imported.ImportedVersion{ReportId: reportID, VersionNo: versionNo, State: "draft", AuthoringMode: "dql",
		AuthoredDql: &authored, GeneratedDql: &generated, ComponentSpecJson: json.RawMessage(`{}`),
		SpecFormatVersion: importSpecFormatVersion, SpecHash: specHash, TypeManifestJson: json.RawMessage(`{}`),
		CompileStatus: "pending", DatlyVersion: importDatlyVersion, CompilerVersion: importCompilerVersion,
		SourceRevision: 1, Notes: importNotes(notes), CreatedBy: createdBy, CreatedAt: createdAt,
		Has: &imported.ImportedVersionHas{ReportId: true, VersionNo: true, State: true, AuthoringMode: true,
			AuthoredDql: true, GeneratedDql: true, ComponentSpecJson: true, SpecFormatVersion: true, SpecHash: true,
			TypeManifestJson: true, CompileStatus: true, DatlyVersion: true, CompilerVersion: true,
			SourceRevision: true, Notes: true, CreatedBy: true, CreatedAt: true}}
	return row
}

// importNotes keeps the caller's note text verbatim and stores NULL for blank
// input, matching the nullable column semantics of every other version write.
func importNotes(notes string) *string {
	if strings.TrimSpace(notes) == "" {
		return nil
	}
	return &notes
}

// importedResourceFileRow carries the caller-owned file facts. The writer's
// lifecycle hook derives the resource identity and content digests and links
// the file to its version.
func importedResourceFileRow(namespace, path string, content []byte, createdAt time.Time) *imported.ImportedResourceFile {
	return &imported.ImportedResourceFile{Namespace: namespace, ResourcePath: path, Content: content, CreatedAt: createdAt,
		Has: &imported.ImportedResourceFileHas{Namespace: true, ResourcePath: true, Content: true, CreatedAt: true}}
}

func draftPointerRow(report *sdk.Report, versionNo int, updatedAt time.Time) *pointer.DraftPointer {
	etag := report.ETag
	draft := versionNo
	return &pointer.DraftPointer{Id: report.ID, CurrentDraftVersion: &draft, Etag: &etag, UpdatedAt: &updatedAt,
		Has: &pointer.DraftPointerHas{Id: true, CurrentDraftVersion: true, Etag: true, UpdatedAt: true}}
}

package sqltransport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/viant/datly-studio/sdk"
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
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer tx.Rollback()
	var next int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_no),0)+1 FROM report_versions WHERE report_id=?`, report.ID).Scan(&next); err != nil {
		return internal(err)
	}
	source := string(bundle.Files[entry])
	now := t.now()
	files := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		files = append(files, name)
	}
	sort.Strings(files)
	bundleHash := sha256.New()
	for _, name := range files {
		fmt.Fprintf(bundleHash, "%d:%s:%d:", len(name), name, len(bundle.Files[name]))
		bundleHash.Write(bundle.Files[name])
	}
	hash := hashVersion(report.ID, next, "dql", "", source, []byte(fmt.Sprintf(`{"bundleSha256":"%x"}`, bundleHash.Sum(nil))))
	_, err = tx.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,notes,created_by,created_at) VALUES(?,?,'draft','dql',?,?,'{}','studio.v1',?,'{}','pending','v1','studio.v1',1,?,?,?)`, report.ID, next, source, source, hash, nullable(request.Input.Notes), principal.Subject, now)
	if err != nil {
		return classify(err, "report version", report.ID)
	}
	reportDigest := sha256.Sum256([]byte(report.ID))
	namespace := fmt.Sprintf("%s.imports.%x", report.OwnerPackage, reportDigest[:8])
	for _, name := range files {
		content := bundle.Files[name]
		digest := sha256.Sum256(content)
		id := sha256.Sum256([]byte(name))
		_, err = tx.ExecContext(ctx, `INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, report.ID, next, hex.EncodeToString(id[:]), namespace, name, content, len(content), hex.EncodeToString(digest[:]), !utf8.Valid(content), now)
		if err != nil {
			return internal(err)
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE reports SET current_draft_version=?,etag=etag+1,updated_at=? WHERE id=?`, next, now, report.ID); err != nil {
		return internal(err)
	}
	if err = tx.Commit(); err != nil {
		return internal(err)
	}
	version, err := t.getVersionValue(ctx, report.ID, next)
	if err != nil {
		return err
	}
	return assign(output, &sdk.DQLLoadResult{Version: version, EntryDQL: entry, Entries: bundle.Entries, Files: files})
}

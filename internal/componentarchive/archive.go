package componentarchive

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
)

// ErrInvalid identifies source or resource data that cannot form a safe archive.
var ErrInvalid = errors.New("invalid component archive")

const (
	MaxResourceFiles = 2000
	MaxResourceBytes = 32 << 20
	MaxSourceBytes   = 4 << 20
	MaxArchiveBytes  = 16 << 20
)

func CheckSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("%w: component has no DQL source", ErrInvalid)
	}
	if len(source) > MaxSourceBytes {
		return fmt.Errorf("%w: component DQL exceeds %d bytes", ErrInvalid, MaxSourceBytes)
	}
	return nil
}

// CheckBudget rejects excessive resources before a caller reads file contents.
func CheckBudget(count, contentBytes int64) error {
	if count < 0 || contentBytes < 0 {
		return fmt.Errorf("resource budget returned invalid totals")
	}
	if count > MaxResourceFiles {
		return fmt.Errorf("%w: component has more than %d resource files", ErrInvalid, MaxResourceFiles)
	}
	if contentBytes > MaxResourceBytes {
		return fmt.Errorf("%w: component resources exceed %d bytes", ErrInvalid, MaxResourceBytes)
	}
	return nil
}

type File struct {
	ReportID     string
	VersionNo    int
	ResourcePath string
	Content      []byte
}

// Build assembles a deterministic, importable component ZIP after its caller
// has authorized the exact version. It has no storage or identity capability.
func Build(ctx context.Context, reportID string, versionNo int, source string, rows []File) (*sdk.ComponentDownload, error) {
	if err := CheckSource(source); err != nil {
		return nil, err
	}
	if len(rows) > MaxResourceFiles {
		return nil, CheckBudget(int64(len(rows)), 0)
	}
	files := make(map[string][]byte, len(rows)+1)
	var contentBytes int64
	for _, row := range rows {
		if int64(len(row.Content)) > MaxResourceBytes-contentBytes {
			return nil, CheckBudget(int64(len(rows)), MaxResourceBytes+1)
		}
		contentBytes += int64(len(row.Content))
		if err := CheckBudget(int64(len(rows)), contentBytes); err != nil {
			return nil, err
		}
		if row.ReportID != reportID || row.VersionNo != versionNo {
			return nil, fmt.Errorf("resource reader returned a mismatched row")
		}
		name := row.ResourcePath
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00") {
			return nil, fmt.Errorf("%w: unsafe resource path %q", ErrInvalid, name)
		}
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("%w: duplicate resource path %s", ErrInvalid, name)
		}
		files[name] = row.Content
	}
	delegated, err := delegateSQL(ctx, source, files)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	entry := "component.dql"
	for index := 1; files[entry] != nil; index++ {
		entry = fmt.Sprintf("component-%d.dql", index)
	}
	files[entry] = []byte(delegated)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, name := range names {
		file, createErr := archive.Create(name)
		if createErr != nil {
			return nil, createErr
		}
		if _, writeErr := file.Write(files[name]); writeErr != nil {
			return nil, writeErr
		}
	}
	if err = archive.Close(); err != nil {
		return nil, err
	}
	if buffer.Len() > MaxArchiveBytes {
		return nil, fmt.Errorf("%w: ZIP archive exceeds %d bytes", ErrInvalid, MaxArchiveBytes)
	}
	return &sdk.ComponentDownload{Filename: fmt.Sprintf("component-v%d.zip", versionNo), MediaType: "application/zip", Archive: buffer.Bytes(), EntryDQL: entry, Files: names}, nil
}

// Datly supplies source spans so Studio never parses SQL to find view boundaries.
func delegateSQL(ctx context.Context, source string, files map[string][]byte) (string, error) {
	inspected := readerbuilder.New(readerbuilder.Config{}).Apply(ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
	if inspected.Structure == nil {
		return "", fmt.Errorf("Datly could not inspect component SQL")
	}
	views := inspected.Structure.Views
	sort.Slice(views, func(i, j int) bool { return views[i].SourceSpan.Start > views[j].SourceSpan.Start })
	boundary := len(source)
	for index, view := range views {
		start, end := view.SourceSpan.Start, view.SourceSpan.End
		if start < 0 || end <= start || end > boundary {
			return "", fmt.Errorf("invalid or overlapping SQL source span for %s", view.Name)
		}
		query := source[start:end]
		if strings.HasPrefix(strings.TrimSpace(query), "${embed:") {
			continue
		}
		name := fmt.Sprintf("sql/view-%d.sql", index+1)
		for suffix := 1; files[name] != nil; suffix++ {
			name = fmt.Sprintf("sql/view-%d-%d.sql", index+1, suffix)
		}
		files[name] = []byte(query)
		source = source[:start] + "${embed:" + name + "}" + source[end:]
		boundary = start
	}
	return source, nil
}

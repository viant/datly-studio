package sqltransport

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
)

func (t *Transport) downloadComponent(ctx context.Context, input, output any) error {
	var identity versionIdentityRequest
	if err := decode(input, &identity); err != nil {
		return invalid(err)
	}
	version, err := t.getVersionValue(ctx, identity.ReportID, identity.VersionNo)
	if err != nil {
		return err
	}
	source := version.GeneratedDQL
	if source == "" {
		source = version.AuthoredDQL
	}
	if strings.TrimSpace(source) == "" {
		return invalid(fmt.Errorf("component has no DQL source"))
	}
	files := map[string][]byte{}
	rows, err := t.DB.QueryContext(ctx, `SELECT resource_path,content FROM report_resource_files WHERE report_id=? AND version_no=?`, identity.ReportID, identity.VersionNo)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var content []byte
		if err = rows.Scan(&name, &content); err != nil {
			return internal(err)
		}
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00") {
			return invalid(fmt.Errorf("unsafe resource path %q", name))
		}
		if _, ok := files[name]; ok {
			return invalid(fmt.Errorf("duplicate resource path %s", name))
		}
		files[name] = content
	}
	if err = rows.Err(); err != nil {
		return internal(err)
	}
	rows.Close()
	source, err = delegateSQL(ctx, source, files)
	if err != nil {
		return invalid(err)
	}
	entry := "component.dql"
	for index := 1; files[entry] != nil; index++ {
		entry = fmt.Sprintf("component-%d.dql", index)
	}
	files[entry] = []byte(source)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, name := range names {
		file, e := archive.Create(name)
		if e != nil {
			return internal(e)
		}
		if _, e = file.Write(files[name]); e != nil {
			return internal(e)
		}
	}
	if err = archive.Close(); err != nil {
		return internal(err)
	}
	return assign(output, &sdk.ComponentDownload{Filename: fmt.Sprintf("component-v%d.zip", identity.VersionNo), MediaType: "application/zip", Archive: buffer.Bytes(), EntryDQL: entry, Files: names})
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
		// Existing delegated SQL and its resource paths remain intact.
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

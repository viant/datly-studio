package sqltransport

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_resource_files/store_download"
	"github.com/viant/datly/authoring/readerbuilder"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
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
	rows, err := t.downloadResourceFiles(ctx, identity.ReportID, identity.VersionNo)
	if err != nil {
		return internal(err)
	}
	for _, row := range rows {
		if row == nil || row.ReportId != identity.ReportID || row.VersionNo != identity.VersionNo {
			return internal(fmt.Errorf("download resource reader returned a mismatched row"))
		}
		name, content := row.ResourcePath, row.Content
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00") {
			return invalid(fmt.Errorf("unsafe resource path %q", name))
		}
		if _, ok := files[name]; ok {
			return invalid(fmt.Errorf("duplicate resource path %s", name))
		}
		files[name] = content
	}
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

func (t *Transport) downloadResourceFiles(ctx context.Context, reportID string, versionNo int) ([]*stored.DownloadResourceFile, error) {
	resources := resource.New()
	if err := resources.Register(stored.FileDatlyResourceNamespace, stored.FileDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.FileComponent{}), "store_download",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo,
		Has: &stored.InputHas{ReportId: true, VersionNo: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("download resource reader returned %T", value)
	}
	return output.Files, nil
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

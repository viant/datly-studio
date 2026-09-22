package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly/bootstrap"
	standaloneconfig "github.com/viant/datly/standalone/config"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/datly/transcribe/dql"
	"github.com/viant/datly/typecatalog"
)

func runTranscribe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: datly transcribe get|patch|post|put [options] module/package")
		return 2
	}
	operation := args[1]
	if operation != "get" && operation != "patch" && operation != "post" && operation != "put" {
		fmt.Fprintln(stderr, "transcribe requires get|patch|post|put")
		return 2
	}
	flags := flag.NewFlagSet("transcribe "+operation, flag.ContinueOnError)
	flags.SetOutput(stderr)
	directory := flags.String("dir", ".", "project root")
	connector := flags.String("connector", "", "schema connector name")
	driver := flags.String("driver", "", "database/sql driver")
	dsn := flags.String("dsn", "", "schema discovery DSN")
	schema := flags.Bool("schema", false, "enable schema discovery")
	if err := flags.Parse(args[2:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "transcribe requires one module-qualified source package")
		return 2
	}
	configuration, err := (standaloneconfig.Loader{}).Load(ctx, filepath.Join(*directory, "datly.yaml"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if configuration.GoBootstrap == nil || len(configuration.GoBootstrap.Packages) == 0 {
		fmt.Fprintln(stderr, "datly.yaml GoBootstrap.Packages is required")
		return 1
	}
	sourcePackage := studioDQLPackage(flags.Arg(0), configuration.GoBootstrap.Packages)
	imports, err := studioDQLImports(*directory, sourcePackage)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	reflectionPackages := append([]string{flags.Arg(0)}, imports...)
	reflected, err := bootstrap.ReflectPackages(reflectionPackages)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// Transcription reads DQL sources with AST discovery for ordinary packages.
	// Studio's static DQL overlays a linked package from the project-root dql/
	// tree, so compile it directly with that linked package as type authority.
	// #package remains the generated Go destination.
	var db *sql.DB
	var refiner *column.Refiner
	if *schema {
		if *connector == "" || *driver == "" || *dsn == "" {
			fmt.Fprintln(stderr, "-schema requires -connector, -driver and -dsn")
			return 2
		}
		db, err = sql.Open(*driver, *dsn)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer db.Close()
		refiner = column.New(column.Connections{*connector: db})
	}
	var compiled *transcribe.Result
	if sourcePackage != flags.Arg(0) {
		compiled, err = compileStudioDQL(ctx, *directory, sourcePackage, flags.Arg(0), *connector, reflected.Types, refiner)
	} else {
		discovery := &transcribe.Discovery{BaseDir: *directory, Include: []string{sourcePackage}, Types: reflected.Types, Connector: *connector, ColumnRefiner: refiner}
		var project *transcribe.ProjectGeneration
		project, err = discovery.Compile(ctx)
		if err == nil && len(project.Components) != 1 {
			err = fmt.Errorf("transcribe requires exactly one component; found %d", len(project.Components))
		}
		if err == nil {
			compiled = project.Components[0]
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	generated, err := (transcribe.Generator{Operation: operation, Language: transcribe.HandlerTarget("go"), EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: *directory})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Generated go %s: %d artifacts\n", operation, len(generated.Result.Files))
	return 0
}

func compileStudioDQL(ctx context.Context, root, sourcePackage, componentPackage, connector string, types *typecatalog.Catalog, refiner *column.Refiner) (*transcribe.Result, error) {
	directory, path, payload, err := studioDQLFile(root, sourcePackage)
	if err != nil {
		return nil, err
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		return nil, err
	}
	return transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: componentPackage, Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Path: path, Text: string(payload), Connector: connector, Resources: resources, Types: types, ColumnRefiner: refiner,
	})
}

func studioDQLPackage(componentPackage string, bootstrapPackages []string) string {
	componentPackage = strings.TrimSpace(componentPackage)
	for _, configured := range bootstrapPackages {
		marker := "/studio/"
		index := strings.Index(configured, marker)
		if index < 0 {
			continue
		}
		modulePath := configured[:index]
		prefix := modulePath + "/studio/"
		if strings.HasPrefix(componentPackage, prefix) {
			return modulePath + "/dql/studio/" + strings.TrimPrefix(componentPackage, prefix)
		}
	}
	return componentPackage
}

func studioDQLImports(root, sourcePackage string) ([]string, error) {
	_, _, payload, err := studioDQLFile(root, sourcePackage)
	if err != nil {
		return nil, err
	}
	prepared := dql.PrepareSource(string(payload))
	if prepared.TypeContext == nil {
		return nil, nil
	}
	imports := map[string]bool{}
	for _, item := range prepared.TypeContext.Imports {
		if packagePath := strings.TrimSpace(item.Package); packagePath != "" {
			imports[packagePath] = true
		}
	}
	result := make([]string, 0, len(imports))
	for packagePath := range imports {
		result = append(result, packagePath)
	}
	sort.Strings(result)
	return result, nil
}

func studioDQLFile(root, sourcePackage string) (string, string, []byte, error) {
	marker := "/dql/"
	index := strings.Index(sourcePackage, marker)
	if index < 0 {
		return "", "", nil, fmt.Errorf("Studio DQL package %q must be below dql/", sourcePackage)
	}
	directory := filepath.Join(root, filepath.FromSlash(sourcePackage[index+1:]))
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", "", nil, err
	}
	files := make([]string, 0, 1)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".dql") {
			files = append(files, entry.Name())
		}
	}
	if len(files) != 1 {
		return "", "", nil, fmt.Errorf("expected exactly one DQL component in %s, found %d", directory, len(files))
	}
	path := filepath.Join(directory, files[0])
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, err
	}
	return directory, path, payload, nil
}

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/viant/datly-studio/schema"
)

func main() {
	driver := flag.String("driver", "mysql", "Target database driver")
	table := flag.String("table", "", "Optional canonical table name (required for SQLite)")
	output := flag.String("output", "", "Generated schema script path")
	flag.Parse()
	if flag.NArg() != 0 || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: studio-schema -driver mysql|sqlite [-table name] -output <path>")
		os.Exit(2)
	}
	var ddl string
	var err error
	switch {
	case *driver == "mysql" && *table == "":
		ddl, err = schema.MySQLDDL()
	case *driver == "mysql":
		ddl, err = schema.MySQLTableStatement(*table)
	case *driver == "sqlite" && *table != "":
		ddl, err = schema.SQLiteTableStatements(*table)
	default:
		err = fmt.Errorf("SQLite requires -table; supported drivers are mysql and sqlite")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, []byte(ddl), 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

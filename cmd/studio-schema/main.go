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
	output := flag.String("output", "", "Generated schema script path")
	flag.Parse()
	if flag.NArg() != 0 || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: studio-schema -driver mysql -output <path>")
		os.Exit(2)
	}
	if *driver != "mysql" {
		fmt.Fprintln(os.Stderr, "studio-schema only generates the MySQL Endly script")
		os.Exit(2)
	}
	ddl, err := schema.MySQLDDL()
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

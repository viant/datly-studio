package studiohost

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// prepareSQLiteStorage creates the configured file's parent directory and
// enables foreign keys on every connection opened by the in-process services.
func prepareSQLiteStorage(dsn string) (string, error) {
	raw := strings.TrimPrefix(dsn, "file:")
	path, query, _ := strings.Cut(raw, "?")
	options, err := url.ParseQuery(query)
	if err != nil {
		return "", errors.New("invalid SQLite storage options")
	}
	if path == ":memory:" && options.Get("cache") != "shared" {
		return "", errors.New("Studio SQLite storage must be file-backed or use a shared-cache memory URI")
	}
	for _, pragma := range options["_pragma"] {
		if strings.HasPrefix(strings.ToLower(pragma), "foreign_keys") && strings.ToLower(pragma) != "foreign_keys(1)" {
			return "", errors.New("Studio storage requires foreign keys")
		}
	}
	options.Add("_pragma", "foreign_keys(1)")
	if path != ":memory:" && options.Get("mode") != "memory" && options.Get("mode") != "ro" && options.Get("mode") != "rw" {
		decoded, err := url.PathUnescape(path)
		if err != nil {
			return "", errors.New("invalid SQLite storage path")
		}
		if strings.HasPrefix(decoded, "//") {
			return "", errors.New("SQLite storage requires a local file path")
		}
		if err := os.MkdirAll(filepath.Dir(decoded), 0700); err != nil {
			return "", errors.New("Studio storage directory could not be created")
		}
	}
	return "file:" + path + "?" + options.Encode(), nil
}

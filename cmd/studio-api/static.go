package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// staticAssets serves only files beneath an operator-selected public build
// directory. This is a generic BFF deployment hook, not an application route.
type staticAssets struct{ root *os.Root }

func newStaticAssets(directory string) (*staticAssets, error) {
	if !filepath.IsAbs(directory) {
		return nil, fmt.Errorf("-static-root must be an absolute directory")
	}
	if filepath.Clean(directory) == string(os.PathSeparator) {
		return nil, fmt.Errorf("-static-root must be a dedicated build directory")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("-static-root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("-static-root must be a directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("-static-root: %w", err)
	}
	index, err := root.Open("index.html")
	if err != nil {
		root.Close()
		return nil, fmt.Errorf("-static-root requires index.html: %w", err)
	}
	indexInfo, err := index.Stat()
	index.Close()
	if err != nil || !indexInfo.Mode().IsRegular() {
		root.Close()
		return nil, fmt.Errorf("-static-root requires a regular index.html")
	}
	return &staticAssets{root: root}, nil
}

func (s *staticAssets) Close() error { return s.root.Close() }

func (s *staticAssets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	escaped := strings.ToLower(r.URL.EscapedPath())
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") || strings.Contains(escaped, "%25") {
		http.NotFound(w, r)
		return
	}
	name, ok := staticFileName(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := s.root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, info.ModTime(), file)
}

func staticFileName(urlPath string) (string, bool) {
	if urlPath == "/" {
		return "index.html", true
	}
	if !strings.HasPrefix(urlPath, "/") || strings.Contains(urlPath, "//") || strings.ContainsRune(urlPath, '\\') {
		return "", false
	}
	parts := strings.Split(strings.TrimPrefix(urlPath, "/"), "/")
	for _, part := range parts {
		if part == "" || strings.HasPrefix(part, ".") || strings.ContainsRune(part, 0) {
			return "", false
		}
	}
	return strings.Join(parts, "/"), true
}

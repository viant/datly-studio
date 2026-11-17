package studioapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// staticAssets serves only files beneath an operator-selected public build
// directory. This is a generic BFF deployment hook, not an application route.
type staticAssets struct {
	root  *os.Root
	files fs.FS
}

func newStaticFS(files fs.FS) (*staticAssets, error) {
	if files == nil {
		return nil, fmt.Errorf("public asset filesystem is required")
	}
	info, err := fs.Stat(files, "index.html")
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("public asset filesystem requires a regular index.html")
	}
	return &staticAssets{files: files}, nil
}

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

func (s *staticAssets) Close() error {
	if s.root != nil {
		return s.root.Close()
	}
	return nil
}

func (s *staticAssets) open(name string) (fs.File, error) {
	if s.root != nil {
		return s.root.Open(name)
	}
	return s.files.Open(name)
}

// Vite's default output uses an eight-character content hash before the extension.
var fingerprintedBundle = regexp.MustCompile(`^assets/[^/]+-[A-Za-z0-9_-]{8}\.(js|css)$`)

func acceptsGzip(header string) bool {
	wildcard := false
	for _, entry := range strings.Split(header, ",") {
		coding, parameter, _ := strings.Cut(entry, ";")
		q := 1.0
		if parameter != "" {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") || err != nil || !(parsed >= 0 && parsed <= 1) {
				continue
			}
			q = parsed
		}
		switch strings.ToLower(strings.TrimSpace(coding)) {
		case "gzip":
			return q > 0
		case "*":
			wildcard = q > 0
		}
	}
	return wildcard
}

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
	file, err := s.open(name)
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
	immutable := fingerprintedBundle.MatchString(name)
	if immutable {
		w.Header().Add("Vary", "Accept-Encoding")
		if acceptsGzip(strings.Join(r.Header.Values("Accept-Encoding"), ",")) {
			packed, err := s.open(name + ".gz")
			if err == nil {
				defer packed.Close()
				packedInfo, statErr := packed.Stat()
				if statErr != nil || !packedInfo.Mode().IsRegular() {
					http.Error(w, "asset unavailable", http.StatusInternalServerError)
					return
				}
				file, info = packed, packedInfo
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
			} else if !errors.Is(err, fs.ErrNotExist) {
				http.Error(w, "asset unavailable", http.StatusInternalServerError)
				return
			}
		}
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	content, ok := file.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "asset unavailable", http.StatusInternalServerError)
			return
		}
		content = bytes.NewReader(data)
	}
	if immutable {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, name, info.ModTime(), content)
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

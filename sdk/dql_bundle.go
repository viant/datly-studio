package sdk

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

// DQLBundle preserves archive-relative resources and lists every root DQL entry.
// Loading a bundle does not execute, validate, or publish its components.
type DQLBundle struct {
	Entries []string          `json:"entries"`
	Files   map[string][]byte `json:"files"`
}

const (
	MaxDQLArchiveBytes = 16 << 20
	MaxDQLBundleBytes  = 64 << 20
	MaxDQLBundleFiles  = 1024
)

// ReadDQL reads one complete UTF-8 DQL document, preserving its source text.
func ReadDQL(reader io.Reader) (*DQLBundle, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxDQLArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxDQLArchiveBytes {
		return nil, fmt.Errorf("DQL exceeds size limit")
	}
	if !utf8.Valid(data) || strings.TrimSpace(string(data)) == "" {
		return nil, fmt.Errorf("DQL must be nonempty UTF-8 text")
	}
	return &DQLBundle{Entries: []string{"main.dql"}, Files: map[string][]byte{"main.dql": data}}, nil
}

// ReadDQLArchive reads ZIP, TAR, or TAR.GZ into memory. Root .dql files are
// entry points; nested files retain their paths for dependency resolution.
// It never extracts onto the host filesystem or follows archive links.
func ReadDQLArchive(reader io.Reader, format string) (*DQLBundle, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxDQLArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxDQLArchiveBytes {
		return nil, fmt.Errorf("archive exceeds size limit")
	}
	result := &DQLBundle{Files: map[string][]byte{}}
	total, count := 0, 0
	seen := map[string]bool{}
	add := func(name string, mode fs.FileMode, source io.Reader) error {
		count++
		if count > MaxDQLBundleFiles {
			return fmt.Errorf("archive has too many entries")
		}
		name = strings.TrimSuffix(name, "/")
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\:\x00") {
			return fmt.Errorf("unsafe archive path %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate archive path %q", name)
		}
		seen[name] = true
		if mode.IsDir() {
			return nil
		}
		if !mode.IsRegular() {
			return fmt.Errorf("archive links and special files are unsupported: %s", name)
		}
		content, err := io.ReadAll(io.LimitReader(source, int64(MaxDQLBundleBytes-total)+1))
		if err != nil {
			return err
		}
		total += len(content)
		if total > MaxDQLBundleBytes {
			return fmt.Errorf("expanded archive exceeds size limit")
		}
		if strings.EqualFold(path.Ext(name), ".dql") && (!utf8.Valid(content) || strings.TrimSpace(string(content)) == "") {
			return fmt.Errorf("DQL %s must be nonempty UTF-8 text", name)
		}
		result.Files[name] = content
		if !strings.Contains(name, "/") && strings.EqualFold(path.Ext(name), ".dql") {
			result.Entries = append(result.Entries, name)
		}
		return nil
	}
	switch strings.ToLower(format) {
	case "zip":
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, file := range archive.File {
			stream, err := file.Open()
			if err != nil {
				return nil, err
			}
			err = add(file.Name, file.Mode(), stream)
			closeErr := stream.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	case "tar", "tar.gz", "tgz":
		var source io.Reader = bytes.NewReader(data)
		if strings.ToLower(format) != "tar" {
			compressed, err := gzip.NewReader(source)
			if err != nil {
				return nil, err
			}
			defer compressed.Close()
			source = compressed
		}
		archive := tar.NewReader(source)
		for {
			header, err := archive.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA && header.Typeflag != tar.TypeDir {
				return nil, fmt.Errorf("archive links and special files are unsupported: %s", header.Name)
			}
			if err = add(header.Name, header.FileInfo().Mode(), archive); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("archive format must be zip, tar, or tar.gz")
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("archive must contain at least one .dql file at its root")
	}
	for name := range result.Files {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, exists := result.Files[parent]; exists {
				return nil, fmt.Errorf("archive file conflicts with directory: %s", parent)
			}
		}
	}
	sort.Strings(result.Entries)
	return result, nil
}

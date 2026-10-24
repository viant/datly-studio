package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	dexec "github.com/viant/datly/exec"
	gate "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/spec"
)

type cacheAdministrationKey struct{}
type cacheAdministration struct {
	reportID string
	version  int
}

// authorizeCacheAdministration is reached only from the server-owned cache
// request. The published generation supplies the component-to-report mapping;
// no client-supplied component identity or data execution permission is used.
func authorizeCacheAdministration(ctx context.Context, target dexec.ComponentTarget, reports map[spec.Key]string, versions map[string]int) error {
	admin, ok := ctx.Value(cacheAdministrationKey{}).(cacheAdministration)
	if !ok || admin.reportID == "" || admin.version < 1 || reports[target.Component] != admin.reportID || versions[admin.reportID] != admin.version {
		return errors.New("cache administration is unavailable")
	}
	return nil
}

// InvalidatePublishedCacheView invalidates one exact cached view in the active
// published reader. The caller is a trusted namespace administration callback;
// normal HTTP/MCP requests cannot create its private context marker.
func (s *Service) InvalidatePublishedCacheView(ctx context.Context, reportID string, version int, routePath, view string) ([]dexec.CacheInvalidation, error) {
	if s == nil || s.manager == nil || reportID == "" || version < 1 || view == "" || !strings.HasPrefix(routePath, "/") || strings.ContainsAny(routePath, "?%#\\\r\n") || strings.Contains(routePath, "..") {
		return nil, errors.New("exact published cache target and view are required")
	}
	body, err := json.Marshal(struct {
		View  string `json:"view"`
		Scope string `json:"scope"`
	}{view, "all"})
	if err != nil {
		return nil, err
	}
	request := httptest.NewRequest(http.MethodPost, gate.DefaultCacheInvalidateURI+routePath, bytes.NewReader(body))
	request = request.WithContext(context.WithValue(ctx, cacheAdministrationKey{}, cacheAdministration{reportID: reportID, version: version}))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	s.manager.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		return nil, fmt.Errorf("published cache invalidation failed with status %d", response.Code)
	}
	var result struct {
		Status      string                    `json:"status"`
		Invalidated []dexec.CacheInvalidation `json:"invalidated"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		return nil, err
	}
	if result.Status != "ok" || len(result.Invalidated) != 1 || result.Invalidated[0].View != view || result.Invalidated[0].Scope != "all" || result.Invalidated[0].Generation == "" || result.Invalidated[0].Error != "" {
		return nil, errors.New("published cache invalidation returned no exact view generation")
	}
	return result.Invalidated, nil
}

func (s *Service) cacheInvalidateHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	values := request.Header.Values("X-Studio-Runtime-Token")
	if len(values) != 1 || !validAdminToken(s.config.Admin.Token, values[0]) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	var input struct {
		NamespaceID string `json:"namespaceId"`
		ReportID    string `json:"reportId"`
		VersionNo   int    `json:"versionNo"`
		RoutePath   string `json:"routePath"`
		View        string `json:"view"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(new(any)) != io.EOF || input.NamespaceID == "" || input.NamespaceID != s.config.NamespaceID {
		http.Error(response, "exact namespace cache target is required", http.StatusBadRequest)
		return
	}
	items, err := s.InvalidatePublishedCacheView(request.Context(), input.ReportID, input.VersionNo, input.RoutePath, input.View)
	if err != nil {
		http.Error(response, "exact published cache view is unavailable", http.StatusConflict)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(response).Encode(struct {
		NamespaceID string                    `json:"namespaceId"`
		Invalidated []dexec.CacheInvalidation `json:"invalidated"`
	}{input.NamespaceID, items})
}

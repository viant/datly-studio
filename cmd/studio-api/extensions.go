package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk/httptransport"
)

const extensionProxyMount = "/v1/studio/extensions"

type extensionProxyConfig struct {
	target *url.URL
	prefix string
}

// resolveExtensionProxyConfig keeps the extension route absent unless both
// operator-provided settings are valid and the BFF is authenticated.
func resolveExtensionProxyConfig(mode, backendURL, upstreamPrefix string) (*extensionProxyConfig, error) {
	backendURL = strings.TrimSpace(backendURL)
	upstreamPrefix = strings.TrimSpace(upstreamPrefix)
	if backendURL == "" && upstreamPrefix == "" {
		return nil, nil
	}
	if mode != string(httptransport.Authenticated) {
		return nil, errors.New("extension proxy requires authenticated mode")
	}
	if backendURL == "" || upstreamPrefix == "" {
		return nil, errors.New("-extension-backend-url and -extension-upstream-prefix must be configured together")
	}
	target, err := url.Parse(backendURL)
	if err != nil || target == nil || (target.Scheme != "http" && target.Scheme != "https") ||
		target.Hostname() == "" || target.User != nil || target.Opaque != "" ||
		(target.Path != "" && target.Path != "/") || target.RawPath != "" ||
		target.RawQuery != "" || target.ForceQuery || target.Fragment != "" {
		return nil, errors.New("-extension-backend-url must be an HTTP(S) origin without credentials, path, query, or fragment")
	}
	if !validExtensionPath(upstreamPrefix) || upstreamPrefix == "/" ||
		upstreamPrefix == "/_studio" || strings.HasPrefix(upstreamPrefix, "/_studio/") {
		return nil, fmt.Errorf("invalid -extension-upstream-prefix %q", upstreamPrefix)
	}
	return &extensionProxyConfig{target: target, prefix: upstreamPrefix}, nil
}

func newExtensionProxy(sessions *bffauth.Service, config *extensionProxyConfig) (http.Handler, error) {
	if sessions == nil || config == nil {
		return nil, errors.New("authenticated extension proxy requires sessions and configuration")
	}
	proxy, err := sessions.Proxy(config.target, extensionProxyMount)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		upstreamPath := strings.TrimPrefix(request.URL.Path, extensionProxyMount)
		if !strings.HasPrefix(request.URL.Path, extensionProxyMount+"/") ||
			!validExtensionRequestPath(request.URL) ||
			(upstreamPath != config.prefix && !strings.HasPrefix(upstreamPath, config.prefix+"/")) {
			http.NotFound(response, request)
			return
		}
		forward := request.Clone(request.Context())
		forward.URL.RawPath = ""
		forward.Host = config.target.Host
		proxy.ServeHTTP(response, forward)
	}), nil
}

// routeExtensionProxy runs the path guard before ServeMux can normalize and
// redirect dot-segment paths inside the extension namespace.
func routeExtensionProxy(next, extension http.Handler) http.Handler {
	if extension == nil {
		return next
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL != nil && strings.HasPrefix(request.URL.Path, extensionProxyMount) {
			extension.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func validExtensionRequestPath(value *url.URL) bool {
	if value == nil || !validExtensionPath(strings.TrimSuffix(value.Path, "/")) {
		return false
	}
	raw := strings.ToLower(value.EscapedPath())
	if strings.Contains(raw, "%2f") || strings.Contains(raw, "%5c") || strings.Contains(raw, "%25") {
		return false
	}
	decoded, err := url.PathUnescape(value.EscapedPath())
	return err == nil && decoded == value.Path
}

func validExtensionPath(value string) bool {
	if !utf8.ValidString(value) || !strings.HasPrefix(value, "/") ||
		strings.HasSuffix(value, "/") || strings.ContainsAny(value, "\\?#%") ||
		strings.Contains(value, "//") || path.Clean(value) != value {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

package accessconfig

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
)

func keyfunc(publicKeyFile, certURL string) (jwtlib.Keyfunc, error) {
	if (publicKeyFile == "") == (certURL == "") {
		return nil, errors.New("exactly one ACL public key file or cert URL is required")
	}
	if publicKeyFile != "" {
		pem, err := os.ReadFile(publicKeyFile)
		if err != nil {
			return nil, errors.New("ACL public key is unavailable")
		}
		key, err := jwtlib.ParseRSAPublicKeyFromPEM(pem)
		if err != nil {
			return nil, errors.New("ACL public key is invalid")
		}
		return func(*jwtlib.Token) (any, error) { return key, nil }, nil
	}
	endpoint, err := url.Parse(certURL)
	if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Opaque != "" || endpoint.Path == "" {
		return nil, errors.New("ACL cert URL is invalid")
	}
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && (strings.EqualFold(endpoint.Hostname(), "localhost") || net.ParseIP(endpoint.Hostname()) != nil && net.ParseIP(endpoint.Hostname()).IsLoopback())) {
		return nil, errors.New("ACL cert URL requires HTTPS or loopback HTTP")
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var mu sync.Mutex
	var keys jwk.Set
	var expires time.Time
	return func(token *jwtlib.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("ACL token has no key ID")
		}
		mu.Lock()
		defer mu.Unlock()
		if keys == nil || !time.Now().Before(expires) || !hasKey(keys, kid) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
			if err != nil {
				return nil, err
			}
			response, err := client.Do(request)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("ACL cert endpoint returned %s", response.Status)
			}
			payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
			if err != nil || len(payload) > 1<<20 {
				return nil, errors.New("ACL cert response is unavailable or too large")
			}
			keys, err = jwk.Parse(payload)
			if err != nil {
				return nil, errors.New("ACL cert response is invalid")
			}
			expires = time.Now().Add(5 * time.Minute)
		}
		key, ok := keys.LookupKeyID(kid)
		if !ok {
			return nil, errors.New("ACL token key ID is unknown")
		}
		var public any
		if err := key.Raw(&public); err != nil {
			return nil, errors.New("ACL token public key is invalid")
		}
		if _, ok := public.(*rsa.PublicKey); !ok {
			return nil, errors.New("ACL token requires an RSA public key")
		}
		return public, nil
	}, nil
}

func hasKey(keys jwk.Set, kid string) bool {
	_, ok := keys.LookupKeyID(kid)
	return ok
}

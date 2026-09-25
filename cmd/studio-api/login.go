package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/viant/datly-studio/sdk/httptransport"
	"golang.org/x/oauth2"
)

func resolveLoginConfig(mode, origin, authURL, tokenURL, clientID, clientSecret, redirectURL, scopes string) (*oauth2.Config, error) {
	authURL, tokenURL, clientID, redirectURL = strings.TrimSpace(authURL), strings.TrimSpace(tokenURL), strings.TrimSpace(clientID), strings.TrimSpace(redirectURL)
	if authURL == "" && tokenURL == "" && clientID == "" && clientSecret == "" && redirectURL == "" {
		return nil, nil
	}
	if mode != string(httptransport.Authenticated) {
		return nil, errors.New("BFF OAuth login requires authenticated mode")
	}
	if authURL == "" || tokenURL == "" || clientID == "" || redirectURL == "" {
		return nil, errors.New("BFF OAuth login requires authorization URL, token URL, client ID, and redirect URL")
	}
	if redirectURL != origin+"/v1/studio/auth/callback" {
		return nil, fmt.Errorf("BFF OAuth redirect URL must exactly match the Studio public origin and callback path")
	}
	auth, err := url.Parse(authURL)
	if err != nil || auth == nil || !auth.IsAbs() || auth.Host == "" || auth.User != nil || auth.RawQuery != "" || auth.Fragment != "" {
		return nil, errors.New("BFF OAuth authorization URL must be a canonical endpoint")
	}
	token, err := url.Parse(tokenURL)
	if err != nil || token == nil || !token.IsAbs() || token.Host == "" || token.User != nil || token.RawQuery != "" || token.Fragment != "" {
		return nil, errors.New("BFF OAuth token URL must be a canonical endpoint")
	}
	if auth.Scheme != token.Scheme || auth.Host != token.Host {
		return nil, errors.New("BFF OAuth endpoints must share one trusted origin")
	}
	items, seen := []string{}, map[string]bool{}
	for _, raw := range strings.Split(scopes, ",") {
		scope := strings.TrimSpace(raw)
		if scope == "" {
			return nil, errors.New("BFF OAuth scopes contain an empty value")
		}
		if !seen[scope] {
			items = append(items, scope)
			seen[scope] = true
		}
	}
	if !seen["openid"] {
		return nil, errors.New("BFF OAuth login requires the openid scope")
	}
	return &oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL,
		Scopes: items, Endpoint: oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL}}, nil
}

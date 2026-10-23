package bffauth

import (
	"context"
	"github.com/viant/datly-studio/sdk"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

// freshIdentitySession is the shared renewal path for native browser identity
// requests and the opaque-cookie BFF proxy. Refresh credentials stay server-side.
func (l *Login) freshIdentitySession(ctx context.Context, id string) (session, error) {
	coordinator, distributed := l.sessions.store.(refreshLeaseStore)
	if !distributed {
		l.refreshMu.Lock()
		defer l.refreshMu.Unlock()
	}
	current, err := l.sessions.sessionForID(ctx, id)
	if err != nil || current.refreshToken == "" {
		return session{}, identitySessionFailure(http.StatusUnauthorized, "sign in required")
	}
	if !l.now().Add(time.Minute).Before(current.idTokenExpiresAt) {
		owner := ""
		if distributed {
			owner, err = randomID()
			if err != nil {
				return session{}, identitySessionFailure(http.StatusServiceUnavailable, "identity refresh unavailable")
			}
			acquired, leaseErr := coordinator.AcquireRefreshLease(ctx, id, owner, l.now(), 30*time.Second)
			if leaseErr != nil {
				return session{}, identitySessionFailure(http.StatusServiceUnavailable, "identity refresh unavailable")
			}
			if !acquired {
				current, err = l.awaitRefreshedSession(ctx, id, current.token)
				if err != nil {
					return session{}, identitySessionFailure(http.StatusServiceUnavailable, "identity refresh is in progress")
				}
				return current, nil
			}
			defer func() {
				releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				_ = coordinator.ReleaseRefreshLease(releaseCtx, id, owner, l.now())
			}()
			current, err = l.sessions.sessionForID(ctx, id)
			if err != nil || current.refreshToken == "" {
				return session{}, identitySessionFailure(http.StatusUnauthorized, "sign in required")
			}
			if l.now().Add(time.Minute).Before(current.idTokenExpiresAt) {
				return current, nil
			}
		}
		refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		refreshCtx = context.WithValue(refreshCtx, oauth2.HTTPClient, l.client)
		base := &oauth2.Token{RefreshToken: current.refreshToken, Expiry: l.now().Add(-time.Second)}
		refreshed, refreshErr := l.config.TokenSource(refreshCtx, base).Token()
		if refreshErr != nil || refreshed == nil {
			return session{}, identitySessionFailure(http.StatusServiceUnavailable, "identity token refresh failed")
		}
		idToken, _ := refreshed.Extra("id_token").(string)
		claims, verifyErr := l.sessions.verifier.VerifyClaims(refreshCtx, idToken)
		if verifyErr != nil || claims == nil || validateTokenBinding(claims, l.sessions.config.Issuer, l.sessions.config.Audience) != nil ||
			claims.ExpiresAt == nil || !l.now().Before(claims.ExpiresAt.Time) ||
			claims.ExpiresAt.Time.After(l.now().Add(maximumBrowserIDTokenTTL)) || claims.Subject != current.principal.Subject {
			return session{}, identitySessionFailure(http.StatusUnauthorized, "refreshed identity token was rejected")
		}
		current.token, current.claims, current.idTokenExpiresAt = idToken, claims, claims.ExpiresAt.Time
		if refreshed.RefreshToken != "" {
			current.refreshToken = refreshed.RefreshToken
		}
		commitCtx, commitCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer commitCancel()
		if distributed {
			err = coordinator.CompleteRefreshLease(commitCtx, id, owner, l.now(), current)
		} else {
			err = l.sessions.store.Put(commitCtx, id, current)
		}
		if err != nil {
			return session{}, identitySessionFailure(http.StatusServiceUnavailable, "identity session update failed")
		}
	}
	return current, nil
}

func identitySessionFailure(status int, message string) error {
	code := sdk.ErrorUnauthorized
	if status == http.StatusServiceUnavailable {
		code = sdk.ErrorUnavailable
	}
	return &sdk.Error{Code: code, Message: message}
}

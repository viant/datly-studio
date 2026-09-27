import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Card, Spinner } from '@blueprintjs/core';
import { StudioAPI } from './studioApi.js';
import { StudioApp } from './StudioApp.jsx';
import { sessionURL } from './session.js';
import { BrowserIdentity } from './browserIdentity.js';

const noExtensions = [];

export function StudioShell({ config, extensions = noExtensions }) {
  const [session, setSession] = useState(() => config.mode === 'development' ? { state: 'ready', subject: config.development.subject } : { state: 'loading' });
  const identity = useMemo(() => config.authentication?.mode === 'identity-token' ? new BrowserIdentity(config) : null, [config]);
  const api = useMemo(() => new StudioAPI(config, { identity, onUnauthorized: () => { identity?.clear(); setSession({ state: 'unauthenticated' }); } }), [config, identity]);
  const signOut = async () => {
    if (!identity) return;
    await identity.signOut();
    setSession({ state: 'unauthenticated' });
  };
  const inspectSession = async () => {
    if (config.mode !== 'authenticated') return;
    setSession({ state: 'loading' });
    try {
      if (identity) {
        await identity.token();
        setSession({ state: 'ready', subject: identity.subject });
        return;
      }
      const response = await fetch(sessionURL(config, config.authentication.mePath), { headers: { Accept: 'application/json' }, credentials: 'include' });
      const value = await response.json().catch(() => ({}));
      if (!response.ok || value?.authenticated !== true) { setSession({ state: 'unauthenticated' }); return; }
      setSession({ state: 'ready', subject: value.subject || 'Authenticated user' });
    } catch (cause) { setSession(cause.status === 401 ? { state: 'unauthenticated' } : { state: 'error', message: cause.message }); }
  };
  useEffect(() => { inspectSession(); }, [config]);
  if (session.state === 'ready') return <StudioApp api={api} mode={config.mode} subject={session.subject} brand={config.brand || 'Datly Studio'} extensions={extensions} onSignOut={identity ? signOut : undefined}/>;
  if (session.state === 'loading') return <AuthWindow><Spinner size={30}/><h1>{config.brand ? `Checking ${config.brand} session` : 'Checking Studio session'}</h1><p>{identity ? 'Restoring your in-memory identity token.' : 'Restoring your secure server-side session.'}</p></AuthWindow>;
  if (session.state === 'error') return <AuthWindow><Callout intent="danger" title="Studio authentication is unavailable" role="alert">{session.message}</Callout><Button icon="refresh" intent="primary" onClick={inspectSession}>Try again</Button></AuthWindow>;
  const loginConfig = identity ? { apiBaseURL: config.authentication.authBaseURL } : config;
  return <AuthWindow><div className="studio-auth-mark">{(config.brand || 'Datly Studio').charAt(0)}</div><h1>Sign in to {config.brand || 'Datly Studio'}</h1><p>{identity ? 'Your identity token stays in this page only. Sign in to restore it.' : 'Your session is missing or expired. Authentication continues through the deployment identity provider; tokens are never stored in this browser.'}</p><a className="bp6-button bp6-intent-primary bp6-large" href={sessionURL(loginConfig, config.authentication.loginPath)}><span className="bp6-button-text">Continue to sign in</span></a><Button minimal icon="refresh" onClick={inspectSession}>Check session again</Button></AuthWindow>;
}

function AuthWindow({ children }) { return <main className="studio-auth-shell"><Card className="studio-card studio-auth-card" elevation={0}>{children}</Card></main>; }

import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Card, Spinner } from '@blueprintjs/core';
import { StudioAPI } from './studioApi.js';
import { StudioApp } from './StudioApp.jsx';
import { sessionURL } from './session.js';

const noExtensions = [];

export function StudioShell({ config, extensions = noExtensions }) {
  const [session, setSession] = useState(() => config.mode === 'development' ? { state: 'ready', subject: config.development.subject } : { state: 'loading' });
  const api = useMemo(() => new StudioAPI(config, { onUnauthorized: () => setSession({ state: 'unauthenticated' }) }), [config]);
  const inspectSession = async () => {
    if (config.mode !== 'authenticated') return;
    setSession({ state: 'loading' });
    try {
      const response = await fetch(sessionURL(config, config.authentication.mePath), { headers: { Accept: 'application/json' }, credentials: 'include' });
      const value = await response.json().catch(() => ({}));
      if (!response.ok || value?.authenticated !== true) { setSession({ state: 'unauthenticated' }); return; }
      setSession({ state: 'ready', subject: value.subject || 'Authenticated user' });
    } catch (cause) { setSession({ state: 'error', message: cause.message }); }
  };
  useEffect(() => { inspectSession(); }, [config]);
  if (session.state === 'ready') return <StudioApp api={api} mode={config.mode} subject={session.subject} brand={config.brand || 'Datly Studio'} extensions={extensions}/>;
  if (session.state === 'loading') return <AuthWindow><Spinner size={30}/><h1>{config.brand ? `Checking ${config.brand} session` : 'Checking Studio session'}</h1><p>Restoring your secure server-side session.</p></AuthWindow>;
  if (session.state === 'error') return <AuthWindow><Callout intent="danger" title="Studio authentication is unavailable" role="alert">{session.message}</Callout><Button icon="refresh" intent="primary" onClick={inspectSession}>Try again</Button></AuthWindow>;
  return <AuthWindow><div className="studio-auth-mark">{(config.brand || 'Datly Studio').charAt(0)}</div><h1>Sign in to {config.brand || 'Datly Studio'}</h1><p>Your session is missing or expired. Authentication continues through the deployment identity provider; tokens are never stored in this browser.</p><a className="bp6-button bp6-intent-primary bp6-large" href={sessionURL(config, config.authentication.loginPath)}><span className="bp6-button-text">Continue to sign in</span></a><Button minimal icon="refresh" onClick={inspectSession}>Check session again</Button></AuthWindow>;
}

function AuthWindow({ children }) { return <main className="studio-auth-shell"><Card className="studio-card studio-auth-card" elevation={0}>{children}</Card></main>; }

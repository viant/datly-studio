import React, { useEffect, useMemo, useState } from 'react';
import {AuthenticationWindow} from './AuthenticationWindow.jsx';
import { StudioAPI } from './studioApi.js';
import { StudioApp } from './StudioApp.jsx';
import { sessionURL, loginURL } from './session.js';
import { BrowserIdentity } from './browserIdentity.js';

const noExtensions = [];

export function StudioShell({ config, extensions = noExtensions, renderNamespaceSkills, nativeCallTool }) {
  const [session, setSession] = useState(() => config.mode === 'development' ? { state: 'ready', subject: config.development.subject } : { state: 'loading' });
  const identity = useMemo(() => config.authentication?.mode === 'identity-token' ? new BrowserIdentity(config) : null, [config]);
  const api = useMemo(() => new StudioAPI(config, { identity, nativeCallTool, onUnauthorized: () => { identity?.clear(); setSession({ state: 'unauthenticated' }); } }), [config, identity, nativeCallTool]);
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
  if (session.state === 'ready') return <StudioApp api={api} mode={config.mode} subject={session.subject} brand={config.brand || 'Data Studio'} extensions={extensions} renderNamespaceSkills={renderNamespaceSkills} onSignOut={identity ? signOut : undefined}/>;
  const loginConfig = identity ? { apiBaseURL: config.authentication.authBaseURL } : config;
  const login=session.state==='unauthenticated'&&config.authentication.enabled!==false?{href:loginURL(loginConfig,config.authentication.loginPath),label:config.authentication.label||'Continue to sign in'}:null;
  return <AuthenticationWindow key={session.state} state={session.state} brand={config.brand||'Data Studio'} message={session.message} identity={Boolean(identity)} login={login} onRetry={inspectSession}/>;
}

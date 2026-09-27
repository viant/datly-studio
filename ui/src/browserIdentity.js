// The ID token exists only in this page's memory. The auth-only backend owns
// PKCE, the refresh token and its HttpOnly session cookie.
export class BrowserIdentity {
  constructor(config, fetcher = globalThis.fetch.bind(globalThis)) {
    // The callback returns to the auth host and its SameSite=Lax cookie must
    // accompany token restoration. Host the UI on that same origin.
    if (globalThis.location?.origin && new URL(config.authentication.authBaseURL).origin !== globalThis.location.origin) {
      throw new Error('Studio UI and authentication must use the same origin');
    }
    this.config = config;
    this.fetcher = fetcher;
    this.idToken = '';
    this.subject = '';
    this.pending = null;
    this.epoch = 0;
    this.signingOut = false;
    this.signOutPromise = null;
  }

  clear() { this.epoch++; this.idToken = ''; this.subject = ''; }

  async signOut() {
    if (this.signOutPromise) return this.signOutPromise;
    this.signOutPromise = (async () => {
      this.signingOut = true;
      this.epoch++;
      const auth = this.config.authentication;
      const url = new URL('/v1/studio/auth/session', `${auth.authBaseURL}/`);
      try {
        const response = await this.fetcher(url.toString(), {
          method: 'DELETE', headers: { Accept: 'application/json' }, credentials: 'include',
        });
        if (!response.ok) throw new Error(`Sign out failed (${response.status})`);
        this.clear();
      } finally { this.signingOut = false; }
    })().finally(() => { this.signOutPromise = null; });
    return this.signOutPromise;
  }

  async token(force = false) {
    if (this.signingOut) throw new Error('Sign out is in progress');
    if (this.idToken && !force) return this.idToken;
    if (this.pending) return this.pending;
    this.pending = this.restore().finally(() => { this.pending = null; });
    return this.pending;
  }

  async restore() {
    const epoch = this.epoch;
    const auth = this.config.authentication;
    const url = new URL(auth.tokenPath, `${auth.authBaseURL}/`);
    const response = await this.fetcher(url.toString(), {
      method: 'POST', headers: { Accept: 'application/json' }, credentials: 'include',
    });
    if (epoch !== this.epoch || this.signingOut) throw new Error('Identity session changed');
    if (!response.ok) {
      this.clear();
      const error = new Error(response.status === 401 ? 'Sign in required' : `Identity session unavailable (${response.status})`);
      error.status = response.status;
      throw error;
    }
    let value;
    try { value = await response.json(); }
    catch {
      this.clear();
      throw new Error('Identity service returned an invalid token response');
    }
    if (typeof value?.id_token !== 'string' || !value.id_token || typeof value.subject !== 'string' || !value.subject) {
      this.clear();
      throw new Error('Identity service returned an invalid token response');
    }
    this.idToken = value.id_token;
    this.subject = value.subject;
    return this.idToken;
  }
}

const development = 'development';
const authenticated = 'authenticated';

function requiredString(value, name) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Studio UI configuration requires ${name}`);
  }
  return value.trim();
}

function localURL(value) {
  const url = new URL(value);
  return url.hostname === '127.0.0.1' || url.hostname === 'localhost' || url.hostname === '::1';
}

// validateConfig intentionally accepts only public browser settings. Browser
// configuration must never contain a client secret or a development bypass
// for an authenticated deployment.
export function validateConfig(value) {
  if (!value || typeof value !== 'object') throw new Error('Studio UI configuration is required');
  const mode = requiredString(value.mode, 'mode');
  const apiBaseURL = requiredString(value.apiBaseURL, 'apiBaseURL').replace(/\/$/, '');
  const brand = value.brand == null ? {} : { brand: requiredString(value.brand, 'brand') };
  if (brand.brand && (brand.brand.length > 80 || /[\u0000-\u001f\u007f]/.test(brand.brand))) throw new Error('Studio UI brand must be short plain text');
  if (mode === development) {
    if (!localURL(apiBaseURL)) throw new Error('development apiBaseURL must use localhost or a loopback address');
    const subject = requiredString(value.development?.subject, 'development.subject');
    return { mode, apiBaseURL, ...brand, development: { subject } };
  }
  if (mode === authenticated) {
    const authentication = value.authentication;
    const authMode = requiredString(authentication?.mode, 'authentication.mode');
    if (authMode === 'identity-token') {
      const authBaseURL = requiredString(authentication.authBaseURL, 'authentication.authBaseURL').replace(/\/$/, '');
      const mcpBaseURL = requiredString(value.mcpBaseURL, 'mcpBaseURL').replace(/\/$/, '');
      for (const [name, raw] of [['apiBaseURL', apiBaseURL], ['mcpBaseURL', mcpBaseURL], ['authentication.authBaseURL', authBaseURL]]) {
        const parsed = new URL(raw);
        if (parsed.protocol !== 'https:' && !localURL(raw)) throw new Error(`${name} must use HTTPS`);
      }
      if (globalThis.location?.origin && new URL(authBaseURL).origin !== globalThis.location.origin) {
        throw new Error('Studio UI and authentication must use the same origin');
      }
      return { mode, apiBaseURL, mcpBaseURL, ...brand,
        authentication: { mode: 'identity-token', authBaseURL,
          loginPath: '/v1/studio/auth/login', tokenPath: '/v1/studio/auth/token' } };
    }
    if (authMode !== 'bff') throw new Error(`unsupported Studio authentication mode ${authMode}`);
    return {
      mode,
      apiBaseURL,
      ...brand,
      authentication: {
        mode: authMode,
        mePath: authentication?.mePath || '/v1/studio/auth/me',
        loginPath: authentication?.loginPath || '/v1/studio/auth/login',
      },
    };
  }
  throw new Error(`unsupported Studio UI mode ${mode}`);
}

export async function loadConfig(url = '/studio-config.json', fetcher = globalThis.fetch) {
  const response = await fetcher(url, { headers: { Accept: 'application/json' } });
  if (!response.ok) throw new Error(`Studio UI configuration request failed (${response.status})`);
  return validateConfig(await response.json());
}

export function sessionURL(config, path) { return new URL(path, `${config.apiBaseURL}/`).toString(); }

// The browser fragment must be carried explicitly: it is never sent in HTTP.
export function loginURL(config, path, location = globalThis.location) {
  const endpoint = new URL(sessionURL(config, path));
  const target = `${location?.pathname || '/'}${location?.search || ''}${location?.hash || ''}`;
  if (target !== '/') endpoint.searchParams.set('returnTo', target);
  else endpoint.searchParams.delete('returnTo');
  return endpoint.toString();
}

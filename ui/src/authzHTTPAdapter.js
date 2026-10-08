import {accessMCPAdapter} from './accessMCPAdapter.js';
import {gateMCPAdapter} from './gateMCPAdapter.js';

const operationPaths = new Set([
  'authorization.check',
  'policies.get', 'policies.context', 'policies.create', 'policies.replace',
  'gates.check', 'gates.get', 'gates.context', 'gates.replace',
]);

// The embedding host owns this same-origin route and its authenticated session.
// Callers supply only resource/action inputs; no provider URL or identity facts
// are accepted by this browser adapter.
export function authzHTTPCall(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== 'function') throw new TypeError('fetch is required');
  return async (name, input) => {
    const operation = String(name || '').replace(/^authz\.sdk\./, '');
    if (!name?.startsWith('authz.sdk.') || !operationPaths.has(operation)) {
      throw new Error('Authorization operation is not supported');
    }
    const response = await fetchImpl(`/v1/authz/sdk/${operation}`, {
      method: 'POST',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: {'Content-Type': 'application/json', Accept: 'application/json'},
      body: JSON.stringify(input),
    });
    if (!response?.ok) {
      const error = new Error(`Authorization request failed (${response?.status || 'network'})`);
      error.status = response?.status;
      error.code = ({401: 'unauthorized', 403: 'forbidden', 409: 'conflict', 503: 'unavailable'})[response?.status];
      throw error;
    }
    const result = await response.json();
    if (!result || typeof result !== 'object') throw new Error('Authorization response is invalid');
    return result;
  };
}

export function accessHTTPAdapter(fetchImpl) {
  return accessMCPAdapter(authzHTTPCall(fetchImpl));
}

export function gateHTTPAdapter(fetchImpl) {
  return gateMCPAdapter(authzHTTPCall(fetchImpl));
}

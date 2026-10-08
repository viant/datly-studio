import test from 'node:test';
import assert from 'node:assert/strict';
import {authzHTTPCall, accessHTTPAdapter, gateHTTPAdapter} from './authzHTTPAdapter.js';

test('same-origin authz HTTP adapter sends only the selected SDK operation and input', async () => {
  const calls = [];
  const fetchImpl = async (url, options) => {
    calls.push({url, options});
    return {ok: true, json: async () => ({document: {revision: 3}})};
  };
  const resource = {kind: 'window', id: 'orders', version: '1', tenant: 'owner'};
  const document = await accessHTTPAdapter(fetchImpl).getResourceAccess(resource);
  assert.equal(document.revision, 3);
  assert.equal(calls[0].url, '/v1/authz/sdk/policies.get');
  assert.equal(calls[0].options.credentials, 'same-origin');
  assert.equal(calls[0].options.cache, 'no-store');
  assert.deepEqual(JSON.parse(calls[0].options.body), {resource});
  assert.equal(calls[0].options.headers.Authorization, undefined);
  await gateHTTPAdapter(fetchImpl).getGateRequirements(resource, 'execute');
  assert.equal(calls[1].url, '/v1/authz/sdk/gates.get');
  assert.deepEqual(JSON.parse(calls[1].options.body), {resource, action: 'execute'});
});

test('same-origin authz HTTP adapter rejects unknown routes and preserves denial status', async () => {
  let called = false;
  const invoke = authzHTTPCall(async () => { called = true; return {ok: false, status: 403}; });
  await assert.rejects(invoke('authz.sdk.unlisted.operation', {}));
  assert.equal(called, false);
  await assert.rejects(invoke('authz.sdk.policies.get', {}), error => error.status === 403);
  const conflict = authzHTTPCall(async () => ({ok: false, status: 409}));
  await assert.rejects(conflict('authz.sdk.gates.replace', {}), error => error.status === 409 && error.code === 'conflict');
});

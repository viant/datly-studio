import assert from 'node:assert/strict';
import test from 'node:test';
import { BrowserIdentity } from './browserIdentity.js';

const config = { authentication: { authBaseURL: 'https://studio.example.com', tokenPath: '/v1/studio/auth/token' } };

test('restores an ID token into memory using only the auth cookie', async () => {
  let request;
  const identity = new BrowserIdentity(config, async (url, init) => {
    request = { url, init };
    return { ok: true, json: async () => ({ id_token: 'signed-id', subject: 'alice' }) };
  });
  assert.equal(await identity.token(), 'signed-id');
  assert.equal(await identity.token(), 'signed-id');
  assert.equal(identity.subject, 'alice');
  assert.equal(request.url, 'https://studio.example.com/v1/studio/auth/token');
  assert.equal(request.init.credentials, 'include');
  assert.equal(request.init.method, 'POST');
  assert.equal(request.init.headers.Authorization, undefined);
});

test('deduplicates refreshes and clears the in-memory token on failure', async () => {
  let count = 0;
  const identity = new BrowserIdentity(config, async () => {
    count++;
    return count === 1
      ? { ok: true, json: async () => ({ id_token: 'signed-id', subject: 'alice' }) }
      : { ok: false, status: 401 };
  });
  await identity.token();
  await assert.rejects(() => identity.token(true), /Sign in required/);
  assert.equal(identity.idToken, '');
  assert.equal(identity.subject, '');
});

test('sign out revokes the cookie session before clearing the in-memory ID token', async () => {
  let revoke;
  const identity = new BrowserIdentity(config, async (url, init) => {
    revoke = { url, init };
    return { ok: true, status: 204 };
  });
  identity.idToken = 'signed-id';
  identity.subject = 'alice';
  await identity.signOut();
  assert.equal(revoke.url, 'https://studio.example.com/v1/studio/auth/session');
  assert.equal(revoke.init.method, 'DELETE');
  assert.equal(revoke.init.credentials, 'include');
  assert.equal(identity.idToken, '');
  assert.equal(identity.subject, '');
});

test('failed sign out leaves the current identity visible for explicit retry', async () => {
  const identity = new BrowserIdentity(config, async () => ({ ok: false, status: 503 }));
  identity.idToken = 'signed-id';
  identity.subject = 'alice';
  await assert.rejects(() => identity.signOut(), /Sign out failed \(503\)/);
  assert.equal(identity.idToken, 'signed-id');
  assert.equal(identity.subject, 'alice');
});

test('an in-flight refresh cannot restore a token after successful sign out', async () => {
  let finishRefresh;
  const identity = new BrowserIdentity(config, async (_url, init) => {
    if (init.method === 'DELETE') return { ok: true, status: 204 };
    return new Promise((resolve) => { finishRefresh = resolve; });
  });
  identity.idToken = 'old-id';
  identity.subject = 'alice';
  const refresh = identity.token(true);
  await identity.signOut();
  finishRefresh({ ok: true, json: async () => ({ id_token: 'late-id', subject: 'alice' }) });
  await assert.rejects(() => refresh, /Identity session changed/);
  assert.equal(identity.idToken, '');
  assert.equal(identity.subject, '');
});

import test from 'node:test';
import assert from 'node:assert/strict';
import {accessMCPAdapter} from './accessMCPAdapter.js';

test('shared policy editor adapter uses MCP context and revisioned replacement', async () => {
  const calls = [];
  const resource = {kind: 'window', id: 'orders', version: '1', tenant: 'tenant'};
  const document = {resource, revision: 3, policies: {execute: {mode: 'protected'}}};
  const adapter = accessMCPAdapter(async (name, input) => {
    calls.push({name, input});
    if (name.endsWith('.check')) return {decision: {Bounded: true, Entities: [{Type: 'customer', ID: '42'}]}};
    return name.endsWith('.context') ? {context: {canManage: false, choices: {role: [{id: 'reader'}]}}} : {document};
  });
  assert.equal((await adapter.getResourceAccess(resource)).revision, 3);
  assert.equal((await adapter.getResourceAccessContext(resource)).canManage, false);
  assert.equal((await adapter.replaceResourceAccess(document)).revision, 3);
  assert.deepEqual(await adapter.checkCurrentAccess(resource, 'execute'), {effect: 'allow', bounded: true, entities: [{Type: 'customer', ID: '42'}]});
  assert.deepEqual(calls.map(call => call.name), ['authz.sdk.policies.get', 'authz.sdk.policies.context', 'authz.sdk.policies.replace', 'authz.sdk.authorization.check']);
  assert.deepEqual(calls[2].input, {document});
  assert.deepEqual(calls[3].input, {resource, action: 'execute'});
});

test('current-principal preview distinguishes explicit denial from authority failure', async () => {
  const resource = {kind: 'window', id: 'orders', version: '1', tenant: 'tenant'};
  const denied = accessMCPAdapter(async () => { throw Object.assign(new Error('denied'), {status: 403}); });
  assert.deepEqual(await denied.checkCurrentAccess(resource, 'execute'), {effect: 'deny', bounded: false, entities: []});
  const unavailable = accessMCPAdapter(async () => { throw Object.assign(new Error('outage'), {status: 503}); });
  await assert.rejects(() => unavailable.checkCurrentAccess(resource, 'execute'), /outage/);
});

import test from 'node:test';
import assert from 'node:assert/strict';
import {gateMCPAdapter} from './gateMCPAdapter.js';

test('gate editor adapter uses exact shared MCP operations and CAS revision', async () => {
  const calls = [];
  const resource = {kind: 'window', id: 'orders', version: '1', tenant: 'tenant'};
  const adapter = gateMCPAdapter(async (name, input) => {
    calls.push({name, input});
    if (name.endsWith('.context')) return {context: {canManage: false, choices: {roles: [{id: 'reader'}]}}};
    if (name.endsWith('.check')) return {decision: {requestId: 'server-request', resourceKind: resource.kind, resourceId: resource.id, resourceVersion: resource.version, action: input.action, effect: 'allow', requirementsRevision: 'r1', validUntil: new Date(Date.now() + 60_000).toISOString()}};
    return {document: {revision: name.endsWith('.replace') ? 'r2' : 'r1', requirements: {schemaVersion: 1}}};
  });
  assert.equal((await adapter.getGateRequirements(resource, 'execute')).revision, 'r1');
  assert.equal((await adapter.getGateRequirementsContext(resource, 'execute')).canManage, false);
  assert.equal((await adapter.checkCurrentGate(resource, 'execute')).effect, 'allow');
  assert.equal((await adapter.replaceGateRequirements({resource, action: 'execute', expectedRevision: 'r1', requirements: {schemaVersion: 1}})).revision, 'r2');
  assert.deepEqual(calls.map(call => call.name), ['authz.sdk.gates.get', 'authz.sdk.gates.context', 'authz.sdk.gates.check', 'authz.sdk.gates.replace']);
  assert.deepEqual(Object.keys(calls[2].input).sort(), ['action', 'resource', 'selected']);
  assert.deepEqual(calls[3].input, {resource, action: 'execute', expectedRevision: 'r1', requirements: {schemaVersion: 1}});
});

import assert from 'node:assert/strict';
import test from 'node:test';
import { changedPolicies, describePolicy } from './resourceAccessReview.js';

test('review lists only structurally changed actions in deterministic order', () => {
  const current = { policies: { execute: { mode: 'protected', rule: { kind: 'role', value: 'reader' } }, describe: { mode: 'public' } } };
  const draft = { policies: { describe: { mode: 'public' }, execute: { rule: { value: 'analyst', kind: 'role' }, mode: 'protected' }, preview: { mode: 'protected', rule: { kind: 'subject', value: 'alice' } } } };
  assert.deepEqual(changedPolicies(current, draft).map(({ action }) => action), ['execute', 'preview']);
  assert.deepEqual(changedPolicies(current, { policies: { describe: { mode: 'public' }, execute: { rule: { value: 'reader', kind: 'role' }, mode: 'protected' } } }), []);
});

test('review describes authored rules without claiming an effective decision', () => {
  assert.equal(describePolicy(null), 'Denied · no policy');
  assert.equal(describePolicy({ mode: 'public' }), 'Public consumption');
  assert.equal(describePolicy({ mode: 'protected', entityType: 'project', rule: { kind: 'all', rules: [
    { kind: 'role', value: 'reviewer' }, { kind: 'exposure', value: 'analytics' },
  ] } }), 'Protected · Match all: Role reviewer; Feature exposure analytics · required project scope');
});

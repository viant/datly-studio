import assert from 'node:assert/strict';
import test from 'node:test';
import { environmentChecks, overviewFromSettled, overviewIssues } from './overviewModel.js';

test('overview derives actionable readiness without invented metrics', () => {
  const state = {
    reports: [{ id: 'draft', title: 'Draft reader', status: 'draft' }],
    connectors: [{ name: 'main', status: 'active', lastTestStatus: '' }],
    namespaces: [{ name: 'general' }],
    runtime: { host: { status: 'unavailable' } },
  };
  assert.deepEqual(overviewIssues(state).map((item) => item.key), ['runtime-host', 'connector:main', 'drafts']);
  assert.deepEqual(environmentChecks(state).map((item) => item.ready), [false, true, false]);
  assert.equal(overviewIssues(state)[0].title, 'Live runtime could not be verified');
  assert.equal(environmentChecks(state)[2].value, 'Check failed');
});

test('overview readiness is clean when the scoped environment is ready', () => {
  const state = { reports: [], connectors: [{ name: 'main', status: 'active', lastTestStatus: 'passed' }], namespaces: [{ name: 'general' }], runtime: { host: { status: 'ready', revision: 4 } } };
  assert.equal(overviewIssues(state).length, 0);
  assert.equal(environmentChecks(state).every((item) => item.ready), true);
});

test('overview preserves successful sections when one request fails', () => {
  const result = overviewFromSettled([
    { status: 'fulfilled', value: { items: [{ id: 'reader' }] } },
    { status: 'rejected', reason: new Error('connector catalog unavailable') },
    { status: 'fulfilled', value: { items: [{ name: 'general' }] } },
    { status: 'fulfilled', value: { host: { status: 'ready' } } },
  ]);
  assert.equal(result.reports.length, 1);
  assert.equal(result.namespaces.length, 1);
  assert.equal(result.runtime.host.status, 'ready');
  assert.equal(result.errors.connectors, 'connector catalog unavailable');
});

test('overview model accepts the initial null loading state', () => {
  assert.equal(Array.isArray(overviewIssues(null)), true);
  assert.equal(environmentChecks(null).length, 3);
});

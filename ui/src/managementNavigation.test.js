import test from 'node:test';
import assert from 'node:assert/strict';
import { managementSection } from './managementNavigation.js';
test('management deep links only resolve existing sections and registered extensions', () => {
  for (const section of ['overview', 'connectors', 'schema', 'namespaces', 'security', 'runtime', 'skills', 'components']) assert.equal(managementSection(`?section=${section}`), section);
  assert.equal(managementSection('?section=host-screen', [{ id: 'host-screen' }]), 'host-screen');
  for (const query of ['', '?section=unknown', '?section=https://other.example', '?section=builder']) assert.equal(managementSection(query), 'overview');
});

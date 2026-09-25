import assert from 'node:assert/strict';
import test from 'node:test';
import { createStudioSDK, defineStudioExtension } from './extensions.js';

test('third-party Studio workspaces register deterministically', () => {
  const sdk = createStudioSDK()
    .register(defineStudioExtension({ id: 'acme.audit', label: 'Audit', order: 20, render: () => null }))
    .register(defineStudioExtension({ id: 'acme.iam', label: 'IAM', order: 10, render: () => null }));
  assert.deepEqual(sdk.extensions().map((item) => item.id), ['acme.iam', 'acme.audit']);
});

test('third-party Studio workspace ids are unique and canonical', () => {
  const extension = defineStudioExtension({ id: 'acme.iam', label: 'IAM', render: () => null });
  const sdk = createStudioSDK().register(extension);
  assert.throws(() => sdk.register(extension), /Duplicate Studio extension/);
  assert.throws(() => defineStudioExtension({ id: 'Invalid name', label: 'Bad', render: () => null }), /canonical/);
  assert.throws(() => defineStudioExtension({ id: 'reports', label: 'Reports', render: () => null }), /reserved/);
});

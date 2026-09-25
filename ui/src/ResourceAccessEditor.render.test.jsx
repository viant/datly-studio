import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ResourceAccessEditor } from './ResourceAccessEditor.jsx';

const resource = { kind: 'skill', id: 'operations', version: '1', tenant: 'one' };
const document = { resource, revision: 4, policies: { retrieve: { mode: 'protected', rule: { kind: 'role', value: 'reader' } } } };
test('loads provider choices and respects server read-only capability', async () => {
  const api = {
    getResourceAccess: vi.fn().mockResolvedValue(document),
    getResourceAccessContext: vi.fn().mockResolvedValue({ source: 'provider-directory', canManage: false, choices: { role: [{ id: 'reader', label: 'Provider reader' }] } }),
  };
  render(<ResourceAccessEditor api={api} resource={resource} actions={['retrieve']}/>);
  await screen.findByText('You have read-only access.');
  expect(screen.getByRole('option', { name: 'Provider reader' })).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Review changes' }).disabled).toBe(true);
  expect(screen.getByLabelText('Access mode').disabled).toBe(true);
});
test('edits a provider role and preserves edits on revision conflict', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn().mockResolvedValue(document), replaceResourceAccess: vi.fn().mockRejectedValue(Object.assign(new Error('conflict'), { code: 'conflict' })) };
  render(<ResourceAccessEditor api={api} resource={resource} actions={['retrieve', 'manageAccess']} choices={{ role: [{ id: 'reader', label: 'Reader' }, { id: 'analyst', label: 'Analyst' }] }}/>);
  await screen.findByText('Policy revision 4');
  await user.selectOptions(screen.getByLabelText('rule value'), 'analyst');
  await user.click(screen.getByRole('button', { name: 'Review changes' }));
  expect(api.replaceResourceAccess).not.toHaveBeenCalled();
  expect(screen.getByText('Review permission changes')).toBeTruthy();
  await user.click(screen.getByRole('button', { name: 'Save permissions' }));
  expect((await screen.findByRole('alert')).textContent).toContain('Your edits are preserved');
  expect(screen.getByLabelText('rule value').value).toBe('analyst');
  expect(api.replaceResourceAccess.mock.calls[0][0].revision).toBe(4);
  expect(screen.getByRole('button', { name: 'Review changes' }).disabled).toBe(true);
  await user.click(screen.getByRole('button', { name: /manageAccess/ }));
  expect(screen.queryByRole('option', { name: 'Public consumption' })).toBeNull();
});

test('serializes entity type and id and mandatory scope independently', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn().mockResolvedValue(document), replaceResourceAccess: vi.fn(async d => ({ ...d, revision: 5 })) };
  render(<ResourceAccessEditor api={api} resource={resource} actions={['retrieve']} choices={{ entity: [{ entity: { type: 'project', id: '101' }, label: 'Operations' }], entityTypes: ['project'] }}/>);
  await screen.findByText('Policy revision 4');
  await user.selectOptions(screen.getByLabelText('rule condition'), 'entity');
  await user.selectOptions(screen.getByLabelText('rule value'), JSON.stringify({ type: 'project', id: '101' }));
  await user.selectOptions(screen.getByLabelText('Required entity scope'), 'project');
  await user.click(screen.getByRole('button', { name: 'Review changes' }));
  expect(api.replaceResourceAccess).not.toHaveBeenCalled();
  expect(screen.getByText(/policy revision 4/)).toBeTruthy();
  await user.click(screen.getByRole('button', { name: 'Save permissions' }));
  await screen.findByText('Permissions saved.');
  expect(api.replaceResourceAccess.mock.calls[0][0].policies.retrieve).toEqual({ mode: 'protected', rule: { kind: 'entity', entity: { type: 'project', id: '101' } }, entityType: 'project' });
});

test('closing the review preserves the draft and performs no mutation', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn().mockResolvedValue(document), replaceResourceAccess: vi.fn() };
  render(<ResourceAccessEditor api={api} resource={resource} actions={['retrieve']}/>);
  await screen.findByText('Policy revision 4');
  await user.selectOptions(screen.getByLabelText('Access mode'), '');
  await user.click(screen.getByRole('button', { name: 'Review changes' }));
  expect(screen.getByRole('region', { name: 'retrieve change' }).textContent).toContain('Denied · no policy');
  await user.click(screen.getByRole('button', { name: 'Back to editor' }));
  expect(api.replaceResourceAccess).not.toHaveBeenCalled();
  expect(screen.getByText('Unsaved permission changes')).toBeTruthy();
});

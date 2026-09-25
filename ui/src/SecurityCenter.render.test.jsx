import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SecurityCenter } from './SecurityCenter.jsx';

test('Permissions links to the ACL UX review in a safe new tab', () => {
  render(<SecurityCenter api={{}}/>);

  const link = screen.getByRole('link', { name: 'Open ACL UX review' });
  expect(link.getAttribute('href')).toMatch(/(?:^|\/)acl-review\.html$/);
  expect(link.target).toBe('_blank');
  expect(link.rel).toContain('noopener');
  expect(link.rel).toContain('noreferrer');
});

test('loads trimmed skill resource values and offers retrieve instead of execute', async () => {
  const user = userEvent.setup();
  const resourceDocument = { resource: {}, revision: 1, policies: { retrieve: { mode: 'public' } } };
  const api = {
    getResourceAccess: vi.fn().mockResolvedValue(resourceDocument),
  };
  render(<SecurityCenter api={api}/>);

  await user.selectOptions(screen.getByLabelText('Resource type'), 'skill');
  await user.type(screen.getByLabelText('Resource ID'), '  deploy-check  ');
  await user.type(screen.getByLabelText('Tenant'), '  platform  ');
  await user.clear(screen.getByLabelText('Version'));
  await user.type(screen.getByLabelText('Version'), '  2.4  ');
  await user.click(screen.getByRole('button', { name: 'Load permissions' }));

  expect(await screen.findByText('Policy revision 1')).toBeTruthy();
  expect(api.getResourceAccess).toHaveBeenCalledWith({
    kind: 'skill', id: 'deploy-check', tenant: 'platform', version: '2.4',
  });
  expect(screen.getByRole('button', { name: /retrieve/ })).toBeTruthy();
  expect(screen.queryByRole('button', { name: /execute/ })).toBeNull();
});

test('report permissions expose preview and execute as separate actions', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn().mockResolvedValue({ resource: { kind: 'report', id: 'overview', tenant: 'one', version: '2' }, revision: 3,
    policies: { preview: { mode: 'protected', rule: { kind: 'role', value: 'reviewer' } }, execute: { mode: 'protected', rule: { kind: 'role', value: 'reader' } } } }) };
  render(<SecurityCenter api={api}/>);
  await user.selectOptions(screen.getByLabelText('Resource type'), 'report');
  await user.type(screen.getByLabelText('Resource ID'), 'overview');
  await user.type(screen.getByLabelText('Tenant'), 'one');
  await user.click(screen.getByRole('button', { name: 'Load permissions' }));
  await screen.findByText('Policy revision 3');
  expect(api.getResourceAccess).toHaveBeenCalledWith({ kind: 'report', id: 'overview', tenant: 'one', version: '1' });
  expect(screen.getByRole('button', { name: /preview Protected/ })).toBeTruthy();
  expect(screen.getByRole('button', { name: /execute Protected/ })).toBeTruthy();
  expect(screen.queryByRole('button', { name: /retrieve/ })).toBeNull();
  await user.click(screen.getByRole('button', { name: /preview Protected/ }));
  expect(screen.queryByRole('option', { name: 'Public consumption' })).toBeNull();
});

test('custom resource kinds need an explicit action contract', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn() };
  render(<SecurityCenter api={api} kinds={['dataset']}/>);
  await user.type(screen.getByLabelText('Resource ID'), 'dataset-1');
  await user.type(screen.getByLabelText('Tenant'), 'one');
  await user.click(screen.getByRole('button', { name: 'Load permissions' }));
  expect(screen.getByText(/Configure the permission actions for resource type/)).toBeTruthy();
  expect(api.getResourceAccess).not.toHaveBeenCalled();
});

test('embedding app can supply a distinct action contract for another resource kind', async () => {
  const user = userEvent.setup();
  const api = { getResourceAccess: vi.fn().mockResolvedValue({ resource: { kind: 'dataset', id: 'facts', tenant: 'one', version: '5' }, revision: 1,
    policies: { execute: { mode: 'protected', rule: { kind: 'role', value: 'reader' } } } }) };
  render(<SecurityCenter api={api} kinds={['dataset']} actionsByKind={{ dataset: ['discover', 'execute'] }}/>);
  await user.type(screen.getByLabelText('Resource ID'), 'facts');
  await user.type(screen.getByLabelText('Tenant'), 'one');
  await user.click(screen.getByRole('button', { name: 'Load permissions' }));
  await screen.findByText('Policy revision 1');
  expect(api.getResourceAccess).toHaveBeenCalledWith({ kind: 'dataset', id: 'facts', tenant: 'one', version: '1' });
  expect(screen.getByRole('button', { name: /execute Protected/ })).toBeTruthy();
  expect(screen.queryByRole('button', { name: /edit/ })).toBeNull();
});

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

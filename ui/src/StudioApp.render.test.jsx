import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { StudioApp } from './StudioApp.jsx';

function catalogAPI() {
  return {
    listComponents: vi.fn().mockResolvedValue({ items: [] }),
    listNamespaces: vi.fn().mockResolvedValue({ items: [] }),
    getRuntimeStatus: vi.fn().mockResolvedValue({}),
    listConnectors: vi.fn(async (input = {}) => {
      if (input.query === 'missing' || input.status === 'disabled') return { items: [] };
      return { items: [{ name: 'reporting', driver: 'sqlite', status: 'active', ownerId: 'owner', lastTestStatus: 'passed', etag: 2 }] };
    }),
    disableConnector: vi.fn(),
  };
}

describe('Studio connector catalog', () => {
  test('allows keyboard activation of sidebar navigation', async () => {
    const user = userEvent.setup();
    render(<StudioApp api={catalogAPI()} mode="development" subject="owner" />);
    const target = screen.getByRole('button', { name: 'Connectors', exact: true });
    target.focus();
    await user.keyboard('{Enter}');
    expect(await screen.findByRole('heading', { name: 'Connectors', level: 1 })).toBeTruthy();
  });
  test('uses one current namespace selector for the component catalog', async () => {
    localStorage.clear();
    const user = userEvent.setup();
    const api = catalogAPI();
    api.config = { apiBaseURL: 'http://namespace-ui.test' };
    api.setNamespace = vi.fn();
    api.setNamespaceBlocked = vi.fn();
    api.listNamespaces.mockResolvedValue({ items: [
      { name: 'alpha', title: 'Alpha', ownerId: 'owner', status: 'active', namespaceId: 'a'.repeat(64) },
      { name: 'beta', title: 'Beta', ownerId: 'owner', status: 'active', namespaceId: 'b'.repeat(64) },
    ] });
    render(<StudioApp api={api} mode="development" subject="owner" />);
    await waitFor(() => expect(api.setNamespace).toHaveBeenLastCalledWith('a'.repeat(64)));
    await user.click(screen.getAllByText('Components').find((item) => item.closest('.bp6-tree-node')));
    const selector = screen.getByRole('combobox', { name: 'Current namespace' });
    expect(screen.queryByRole('combobox', { name: 'Filter components by namespace' })).toBeNull();
    await user.selectOptions(selector, 'b'.repeat(64));
    await waitFor(() => expect(api.setNamespace).toHaveBeenLastCalledWith('b'.repeat(64)));
    expect(localStorage.getItem('studio:namespace:http://namespace-ui.test:owner')).toBe('b'.repeat(64));
  });
  test('keeps namespace deletion visible until its request finishes', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    let finish;
    api.deleteNamespace = vi.fn(() => new Promise((resolve) => { finish = resolve; }));
    api.listNamespaces.mockResolvedValue({ items: [{ name: 'owned', title: 'Owned', ownerId: 'owner', status: 'active', canManage: true, etag: 3 }] });
    render(<StudioApp api={api} mode="development" subject="owner" />);
    await user.click(await screen.findByText('Namespaces'));
    await user.click(await screen.findByRole('button', { name: 'Delete owned' }));
    await user.click(await screen.findByRole('button', { name: 'Delete namespace' }));
    await user.keyboard('{Escape}');
    expect(screen.getByRole('button', { name: 'Delete namespace' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Cancel' }).disabled).toBe(true);
    expect(api.deleteNamespace).toHaveBeenCalledTimes(1);
    finish();
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Delete namespace' })).toBeNull());
  });
  test('shows view-only namespace actions for viewers and management actions for owners', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    api.listNamespaces.mockResolvedValue({ items: [
      { name: 'shared', title: 'Shared', ownerId: 'other', status: 'active', canManage: false },
      { name: 'owned', title: 'Owned', ownerId: 'owner', status: 'active', canManage: true },
    ] });
    render(<StudioApp api={api} mode="development" subject="owner" />);
    await user.click(await screen.findByText('Namespaces'));
    expect(await screen.findByRole('button', { name: 'View shared' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Delete shared' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Edit owned' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Delete owned' })).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'View shared' }));
    expect(await screen.findByRole('dialog', { name: 'Namespace · shared' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Save namespace' })).toBeNull();
  });
  test('offers authenticated sign-out and preserves a retry on revocation failure', async () => {
    const user=userEvent.setup();
    const onSignOut=vi.fn().mockRejectedValueOnce(new Error('Identity service unavailable')).mockResolvedValueOnce();
    render(<StudioApp api={catalogAPI()} mode="authenticated" subject="owner" onSignOut={onSignOut}/>);
    await user.click(screen.getByRole('button',{name:'Sign out'}));
    expect((await screen.findByRole('alert')).textContent).toContain('Identity service unavailable');
    await user.click(screen.getByRole('button',{name:'Try again'}));
    await waitFor(()=>expect(onSignOut).toHaveBeenCalledTimes(2));
  });
  test('submits server-side search and renders a filter-aware empty state', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    render(<StudioApp api={api} mode="development" subject="owner" />);

    await user.click(await screen.findByText('Connectors'));
    const selectedNav=screen.getAllByText('Connectors').find((item)=>item.closest('.bp6-tree-node'));
    expect(selectedNav.closest('.bp6-tree-node').className).toContain('bp6-tree-node-selected');
    const search = await screen.findByRole('textbox', { name: 'Search connectors' });
    await user.type(search, 'missing');
    await user.click(screen.getByRole('button', { name: 'Search' }));

    expect(await screen.findByRole('heading', { name: 'No matching connectors' })).toBeTruthy();
    expect(screen.getByText(/Adjust the search or lifecycle filter/i)).toBeTruthy();
    expect(api.listConnectors).toHaveBeenCalledWith(expect.objectContaining({ query: 'missing', limit: 26, offset: 0 }));

    await user.click(screen.getByRole('button', { name: 'Clear connector search' }));
    await waitFor(() => expect(screen.getByText('reporting')).toBeTruthy());
  });

  test('filters lifecycle state through the catalog operation', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    render(<StudioApp api={api} mode="development" subject="owner" />);

    await user.click(await screen.findByText('Connectors'));
    await user.selectOptions(await screen.findByRole('combobox', { name: 'Filter connectors by status' }), 'disabled');

    expect(await screen.findByRole('heading', { name: 'No matching connectors' })).toBeTruthy();
    expect(api.listConnectors).toHaveBeenCalledWith(expect.objectContaining({ status: 'disabled', limit: 26, offset: 0 }));
  });

  test('keeps a stale lifecycle change unapplied and offers catalog refresh', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    api.disableConnector.mockRejectedValue(Object.assign(new Error('connector etag does not match'), { code: 'conflict' }));
    render(<StudioApp api={api} mode="development" subject="owner" />);

    await user.click(await screen.findByText('Connectors'));
    await user.click(await screen.findByRole('button', { name: 'Disable connector' }));

    const warning = await screen.findByRole('alert');
    expect(warning.textContent).toContain('Connector changed elsewhere');
    expect(warning.textContent).toContain('No change was applied');
    expect(api.disableConnector).toHaveBeenCalledWith('reporting', 2);
    const callsBeforeRefresh = api.listConnectors.mock.calls.length;
    await user.click(screen.getByRole('button', { name: 'Refresh catalog' }));
    await waitFor(() => expect(api.listConnectors.mock.calls.length).toBeGreaterThan(callsBeforeRefresh));
  });
});

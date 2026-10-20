import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {managementResourceFixture} from './managementWindowFixture.js';
import { StudioApp } from './StudioApp.jsx';

// Transform the lazy runtime as test setup, outside interaction deadlines.
import './ManagementWindowRuntime.jsx';

function catalogAPI() {
  const api = {
    readManagementResource: vi.fn(async(uri)=>managementResourceFixture(uri.split("/").at(-1))),
    listComponents: vi.fn().mockResolvedValue({ items: [] }),
    listNamespaces: vi.fn().mockResolvedValue({ items: [] }),
    getRuntimeStatus: vi.fn().mockResolvedValue({}),
    listConnectors: vi.fn(async (input = {}) => {
      if (input.query === 'missing' || input.status === 'disabled') return { items: [] };
      return { items: [{ name: 'reporting', driver: 'sqlite', status: 'active', ownerId: 'owner', lastTestStatus: 'passed', etag: 2 }] };
    }),
    disableConnector: vi.fn(),
  };
  api.invoke=vi.fn((operation,input)=>{
    if(operation==='namespaces.list')return api.listNamespaces(input);
    if(operation==='namespaces.delete')return api.deleteNamespace(input.name,input.etag);
    if(operation==='namespaces.get')return api.listNamespaces().then(page=>page.items.find(item=>item.name===input.name));
    if(operation==='connectors.list')return api.listConnectors(input);
    if(operation==='components.list')return api.listComponents(input);
    if(operation==='components.create')return api.createComponent(input);
    if(operation==='versions.list')return api.listVersions(input.reportId,input.input);
    if(operation==='runtime.status')return api.getRuntimeStatus();
    if(operation==='connectors.disable')return api.disableConnector(input.name,input.etag);
    throw new Error('Unexpected operation '+operation);
  });
  return api;
}

describe('Studio connector catalog', () => {
  test('opens the management section selected by a deep link', async () => {
    window.history.replaceState({}, '', '/?section=connectors');
    const api = catalogAPI();
    try {
      render(<StudioApp api={api} mode="development" subject="owner" />);
      expect(await screen.findByRole('button', { name: 'New connector', exact:true })).toBeTruthy();
      await waitFor(() => expect(api.listConnectors).toHaveBeenCalled());
    } finally {
      window.history.replaceState({}, '', '/');
    }
  });
  test('opens the created component directly in root SQL', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    api.listNamespaces.mockResolvedValue({items:[{name:'general',title:'General',status:'active'}]});
    api.createComponent = vi.fn().mockResolvedValue({id:'created',title:'New reader',slug:'new-reader',defaultConnectorName:'reporting'});
    api.listVersions = vi.fn().mockResolvedValue({items:[]});
    render(<StudioApp api={api} mode="development" subject="owner"/>);
    await user.click(screen.getAllByRole('button',{name:'Components',exact:true}).find((item)=>item.closest('.bp6-tree-node')));
    await screen.findByRole('heading',{name:'No components yet'});
    await user.click(screen.getAllByRole('button',{name:'New component'})[0]);
    await user.type(screen.getByLabelText('Component title'),'New reader');
    await user.click(await screen.findByRole('button',{name:'Active connector',exact:true}));
    await user.click(await screen.findByRole('menuitem',{name:'reporting · sqlite',exact:true}));
    await user.click(screen.getByRole('button',{name:'Create component'}));
    expect(await screen.findByRole('heading',{name:'Root SQL'})).toBeTruthy();
    expect(screen.getByRole('main',{name:'Component editor'})).toBeTruthy();
    await waitFor(()=>expect(screen.queryByRole('dialog',{name:'New component'})).toBeNull());
    expect(api.listVersions).toHaveBeenCalledWith('created',{limit:100});
    expect(api.createComponent).toHaveBeenCalledTimes(1);
    await user.clear(screen.getByLabelText('Root view identifier'));
    await user.type(screen.getByLabelText('Root view identifier'),'customers');
    await user.selectOptions(screen.getByLabelText('Component visibility'),'internal');
    await waitFor(()=>expect(screen.getByLabelText('Root view identifier').value).toBe('customers'));
    expect(screen.getByLabelText('Component visibility').value).toBe('internal');
    expect(api.listVersions).toHaveBeenCalledTimes(1);
  });

  test('allows keyboard activation of sidebar navigation', async () => {
    const user = userEvent.setup();
    render(<StudioApp api={catalogAPI()} mode="development" subject="owner" />);
    const target = screen.getByRole('button', { name: 'Connectors', exact: true });
    target.focus();
    await user.keyboard('{Enter}');
    expect(await screen.findByRole('button', { name: 'New connector', exact:true })).toBeTruthy();
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
    await user.click(await screen.findByRole('button', { name: 'Delete namespace',exact:true }));
    await user.click(await screen.findByRole('button', { name: 'Confirm deletion' }));
    await user.keyboard('{Escape}');
    expect(screen.getByRole('button', { name: 'Confirm deletion' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Cancel' }).disabled).toBe(true);
    expect(api.deleteNamespace).toHaveBeenCalledTimes(1);
    finish();
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Confirm deletion' })).toBeNull());
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
    const row=await screen.findByText('Shared');
    if(!row.closest('tr').classList.contains('selected-row'))await user.click(row);
    await user.click(screen.getByRole('button',{name:'Edit namespace',exact:true}));
    expect(await screen.findByRole('dialog',{name:'Namespace settings'})).toBeTruthy();
    expect(screen.queryByRole('button',{name:'Save',exact:true})).toBeNull();
    expect(screen.getByLabelText('Display title',{exact:true}).readOnly).toBe(true);
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
    const search = await screen.findByRole('textbox', { name: 'Find connector, driver, or owner' });
    await user.type(search, 'missing');
    await user.click(screen.getByRole('button', { name: 'Search' }));

    await waitFor(()=>expect(screen.queryByText('reporting')).toBeNull());
    expect(api.listConnectors).toHaveBeenCalledWith(expect.objectContaining({ query: 'missing', limit: 25, offset: 0 }));

    await user.clear(search);await user.click(screen.getByRole('button',{name:'Search',exact:true}));
    await waitFor(() => expect(screen.getByText('reporting')).toBeTruthy());
  });

  test('filters lifecycle state through the catalog operation', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    render(<StudioApp api={api} mode="development" subject="owner" />);

    await user.click(await screen.findByText('Connectors'));
    await user.click(await screen.findByRole('button',{name:'Status',exact:true}));
    await user.click(await screen.findByText('Disabled',{exact:true}));
    await user.click(screen.getByRole('button',{name:'Search',exact:true}));

    await waitFor(()=>expect(screen.queryByText('reporting')).toBeNull());
    expect(api.listConnectors).toHaveBeenCalledWith(expect.objectContaining({ status: 'disabled', limit: 25, offset: 0 }));
  });

  test('keeps a stale lifecycle change unapplied and offers catalog refresh', async () => {
    const user = userEvent.setup();
    const api = catalogAPI();
    api.disableConnector.mockRejectedValue(Object.assign(new Error('connector etag does not match'), { code: 'conflict' }));
    render(<StudioApp api={api} mode="development" subject="owner" />);

    await user.click(await screen.findByText('Connectors'));
    await screen.findByText('reporting');
    await user.click(await screen.findByRole('button', { name: 'Activate or disable',exact:true }));

    expect(await screen.findByText('connector etag does not match')).toBeTruthy();
    expect(api.disableConnector).toHaveBeenCalledWith('reporting', 2);
    const callsBeforeRefresh = api.listConnectors.mock.calls.length;
    await user.click(screen.getByRole('button', { name: 'Refresh',exact:true }));
    await waitFor(() => expect(api.listConnectors.mock.calls.length).toBeGreaterThan(callsBeforeRefresh));
  });
});

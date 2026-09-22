import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { StudioApp } from './StudioApp.jsx';

function catalogAPI() {
  return {
    listReports: vi.fn().mockResolvedValue({ items: [] }),
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

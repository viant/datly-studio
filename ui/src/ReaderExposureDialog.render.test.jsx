import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderExposureDialog } from './ReaderExposureDialog.jsx';

const structure = {
  component: {
    name: 'reader', description: 'Read vendors',
    rootView: { name: 'reader', namespace: 'vendor' },
    routes: [{ method: 'GET', path: '/vendors', mcp: [] }],
    settings: {},
  },
};

describe('ReaderExposureDialog', () => {
  test('keeps denied projection until the author explicitly enables it', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    const scoped = { ...structure, component: { ...structure.component, settings: { report: { enabled: true } }, rootView: { name: 'reader', namespace: 'vendor', selector: { allowFields: false } } }, functions: [{ name: 'selector_fields', args: ['vendor', 'false'], occurrence: 0 }] };
    const api = { listConnectors: vi.fn().mockResolvedValue({items:[{name:'main',driver:'sqlite'}]}), getComponent: vi.fn().mockResolvedValue({id:'vendor',title:'Vendor Catalog',description:''}) };
    render(<ReaderExposureDialog isOpen api={api} structure={scoped} report={{id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main',ownerPackage:'alice'}} onClose={vi.fn()} onApply={onApply}/>);
    expect(screen.getByRole('checkbox',{name:'Allow field selection'}).checked).toBe(false);
    expect(screen.getByText(/This also permits field selection on the source reader/)).toBeTruthy();
    await user.click(screen.getByRole('checkbox',{name:'Allow field selection'}));
    await user.click(screen.getByRole('button',{name:'Save component'}));
    expect(onApply.mock.calls[0][0].operations.at(-1)).toEqual({type:'updateFunction',function:{name:'selector_fields',args:['vendor','true'],occurrence:0,expectedArgs:['vendor','false']}});
  });

  test('persists cube, composition, and MCP controls in one atomic operation', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    const onClose = vi.fn();
    const api = { listConnectors: vi.fn().mockResolvedValue({items:[{name:'main',driver:'sqlite'}]}), getComponent: vi.fn().mockResolvedValue({id:'vendor',title:'Vendor Catalog',description:'',etag:2}), updateComponent: vi.fn() };
    render(<ReaderExposureDialog
      isOpen api={api} structure={structure} version={{ compileStatus: 'valid' }}
      report={{ id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main',ownerPackage: 'alice' }} onClose={onClose} onApply={onApply}
    />);

    await user.click(screen.getByRole('checkbox', { name: 'Enable cube' }));
    expect(screen.getByRole('checkbox', { name: 'Expose cube as an MCP tool' }).checked).toBe(true);
    await user.click(screen.getByRole('checkbox', { name: 'Enable cube composition' }));
    await user.click(screen.getByRole('checkbox', { name: 'Expose composition as an MCP tool' }));
    await user.clear(screen.getByLabelText('Maximum cubes'));
    await user.type(screen.getByLabelText('Maximum cubes'), '12');
    await user.clear(screen.getByLabelText('Maximum result rows'));
    await user.type(screen.getByLabelText('Maximum result rows'), '250');
    await user.clear(screen.getByLabelText('Timeout (ms)'));
    await user.type(screen.getByLabelText('Timeout (ms)'), '45000');
    await user.click(screen.getByRole('checkbox', { name: 'Enable MCP tool' }));
    expect(screen.getByLabelText('Canonical MCP tool name').value).toBe('alice.vendors.read');
    await user.clear(screen.getByLabelText('Description resource'));
    await user.type(screen.getByLabelText('Description resource'), 'docs/vendors.md');
    await user.click(screen.getByRole('button', { name: 'Save component' }));

    expect(onApply).toHaveBeenCalledWith({
      type: 'batch',
      operations: [
        { type: 'setSetting', setting: { name: 'connector', args: ['"main"'] } },
        { type: 'setSetting', setting: { name: 'cube', args: [] } },
        { type: 'setSetting', setting: { name: 'cubeCompose', args: ['true', 'true', '12', '250', '45000'] } },
        { type: 'setSetting', setting: { name: 'mcp', args: ['"alice.vendors.read"', '"Read vendors"', '"docs/vendors.md"'] } },
        { type: 'setSetting', setting: { name: 'mcpOnly', remove: true } },
      ],
    });
    expect(onClose).toHaveBeenCalledOnce();
  });

  test('rejects an MCP name outside the owner namespace', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    const api = { listConnectors: vi.fn().mockResolvedValue({items:[{name:'main',driver:'sqlite'}]}) };
    render(<ReaderExposureDialog
      isOpen api={api} structure={structure} version={{ compileStatus: 'valid' }}
      report={{ id:'vendor',title:'Vendor Catalog',defaultConnectorName:'main',ownerPackage: 'alice' }} onClose={vi.fn()} onApply={onApply}
    />);

    await user.click(screen.getByRole('checkbox', { name: 'Enable MCP tool' }));
    await user.clear(screen.getByLabelText('Canonical MCP tool name'));
    await user.type(screen.getByLabelText('Canonical MCP tool name'), 'vendor.read');
    await user.click(screen.getByRole('button', { name: 'Save component' }));

    expect((await screen.findByRole('alert')).textContent).toContain('owner prefix alice.');
    expect(onApply).not.toHaveBeenCalled();
  });
});

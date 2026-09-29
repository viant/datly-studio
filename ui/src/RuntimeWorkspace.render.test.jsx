import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RuntimeWorkspace } from './RuntimeWorkspace.jsx';

const status = {
  status: 'active', activeGeneration: 12, reportCount: 1,
  host: { status: 'ready', revision: 12 },
  readers: [{
    reportId: 'vendor', title: 'Vendor Catalog', namespace: 'general', ownerPackage: 'alice', connectorName: 'reporting', versionNo: 2, status: 'active',
    mcpExposures: [{ kind: 'tool', name: 'alice.vendor.read', component: 'reader', method: 'GET', path: '/vendors', enabled: true }, { kind: 'tool', name: 'VendorCube', component: 'VendorCube', method: 'POST', path: '/vendors/cube', enabled: true }, { kind: 'tool', name: 'VendorCubeCompose', component: 'VendorCubeCompose', method: 'POST', path: '/vendors/cube/compose', enabled: true }],
    mcpResources: [{ namespace: 'alice.docs', rootPath: 'guide', uriPrefix: 'skill://alice-guide/' }],
    skills: [{ skillId: 'guide', skillRoot: '.', uriPrefix: 'skill://alice-guide/' }],
  }],
};

describe('Runtime catalogs', () => {
  test('keeps live MCP discovery inside Runtime tabs', async () => {
    const user=userEvent.setup();
    const api={listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read',description:'Read vendors',inputSchema:{type:'object',properties:{limit:{type:'integer'}}},outputSchema:{type:'object',properties:{vendors:{type:'array'}}}}])};
    const onOpen=vi.fn();
    render(<RuntimeWorkspace api={api} mode="runtime" status={status} loading={false} error="" onRefresh={vi.fn()} onOpenComponent={onOpen}/>);
    expect(screen.getByRole('heading', { name: 'Runtime' })).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Open Vendor Catalog'}));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({id:'vendor',ownerPackage:'alice',versionNo:2}));
    await user.click(screen.getByRole('tab', { name: /MCP tools/ }));
    expect(await screen.findByText('alice.vendor.read')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await waitFor(() => expect(api.listMCPTools).toHaveBeenCalledTimes(2));
    expect(screen.getAllByText('1 fields')).toHaveLength(2);
    await user.click(screen.getByRole('button',{name:'Inspect alice.vendor.read schema'}));
    const dialog = await screen.findByRole('dialog',{name:'alice.vendor.read contract'});
    expect(screen.getAllByText(/vendors/).length).toBeGreaterThan(1);
    await user.click(within(dialog).getAllByRole('button', { name: 'Close' }).at(-1));
    await user.click(screen.getByRole('tab', { name: 'Resources (1)' }));
    expect(screen.getByText('skill://alice-guide/')).toBeTruthy();
  });

  test('links a published skill to its versioned component editor', async () => {
    const user=userEvent.setup();
    const onOpen=vi.fn();
    render(<RuntimeWorkspace api={{listMCPSkills:vi.fn().mockResolvedValue([{uri:'skill://alice-guide/SKILL.md',frontmatter:{name:'alice-guide',description:'Use vendors','allowed-tools':'alice.vendor.read'}}]),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'}])}} mode="skills" status={status} loading={false} error="" onRefresh={vi.fn()} onOpenComponent={onOpen} />);
    expect(screen.getByRole('heading', { name: 'Skills', level: 1 })).toBeTruthy();
    expect(await screen.findByText('alice-guide')).toBeTruthy();
    expect(screen.getByRole('button',{name:'Open MCP tool alice.vendor.read component'})).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Open MCP tool alice.vendor.read component'}));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({id:'vendor',versionNo:2}));
    onOpen.mockClear();
    await user.click(screen.getByRole('button', { name: 'Edit in component' }));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: 'vendor', title: 'Vendor Catalog', ownerPackage:'alice', versionNo:2, openResources: true }));
  });

  test('keeps live skill inventory read-only until the owning component is opened',async()=>{
    const user=userEvent.setup();
    const api={listMCPSkills:vi.fn().mockResolvedValue([{uri:'skill://alice-guide/SKILL.md',frontmatter:{name:'alice-guide',description:'Guide','allowed-tools':'alice.vendor.read'}}]),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'}]),upsertResourceFile:vi.fn(),deleteSkillRoot:vi.fn(),publishReader:vi.fn()};
    const onOpen=vi.fn();
    render(<RuntimeWorkspace api={api} mode="skills" status={status} loading={false} error="" onRefresh={vi.fn()} onOpenComponent={onOpen}/>);
    await screen.findByText('alice-guide');
    expect(screen.getByRole('radio',{name:'Select alice-guide skill'}).getAttribute('name')).toBe('published-skill');
    expect(screen.queryByRole('button',{name:'Assign'})).toBeNull();
    expect(screen.queryByRole('button',{name:'Remove alice.vendor.read from skill'})).toBeNull();
    await user.click(screen.getByRole('button',{name:'Edit in component'}));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({id:'vendor',openResources:true}));
    expect(api.upsertResourceFile).not.toHaveBeenCalled();
    expect(api.deleteSkillRoot).not.toHaveBeenCalled();
    expect(api.publishReader).not.toHaveBeenCalled();
  });
});

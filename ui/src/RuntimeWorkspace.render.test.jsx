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
    render(<RuntimeWorkspace api={api} mode="runtime" status={status} loading={false} error="" onRefresh={vi.fn()} />);
    expect(screen.getByRole('heading', { name: 'Runtime' })).toBeTruthy();
    await user.click(screen.getByRole('tab', { name: 'MCP tools (0)' }));
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
    expect(screen.getByRole('heading', { name: 'Skills & Resources' })).toBeTruthy();
    expect(await screen.findByText('alice-guide')).toBeTruthy();
    expect(screen.getByText('alice.vendor.read')).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Open MCP tool alice.vendor.read component'}));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({id:'vendor',versionNo:2}));
    onOpen.mockClear();
    await user.click(screen.getByRole('button', { name: 'Edit' }));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: 'vendor', title: 'Vendor Catalog', ownerPackage:'alice', versionNo:2, openResources: true }));
  });

  test('assigns a published MCP tool from the skill toolbar and republishes',async()=>{
    const user=userEvent.setup();
    const snapshot={version:{versionNo:2,sourceRevision:4},files:[{resourceId:'file',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'---\nname: alice-guide\ndescription: Guide\nallowed-tools: alice.vendor.read\n---\nGuide'}],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[{skillId:'guide',folderId:'folder',skillRoot:'.'}]};
    const next={...snapshot,version:{versionNo:2,sourceRevision:5}};
    const api={listMCPSkills:vi.fn().mockResolvedValue([{uri:'skill://alice-guide/SKILL.md',frontmatter:{name:'alice-guide',description:'Guide','allowed-tools':'alice.vendor.read'}}]),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'},{name:'extra.tool'}]),getResources:vi.fn().mockResolvedValue(snapshot),upsertResourceFile:vi.fn().mockResolvedValue(next),validateVersion:vi.fn().mockResolvedValue({valid:true}),publishReader:vi.fn().mockResolvedValue({status:'active'})};
    const onRefresh=vi.fn().mockResolvedValue();
    render(<RuntimeWorkspace api={api} mode="skills" status={status} loading={false} error="" onRefresh={onRefresh} onOpenComponent={vi.fn()}/>);
    await screen.findByText('alice-guide');
    await user.selectOptions(screen.getByLabelText('Assign published MCP tool'),'extra.tool');
    await user.click(screen.getByRole('button',{name:'Assign'}));
    expect(api.upsertResourceFile).toHaveBeenCalledWith(expect.objectContaining({content:expect.stringContaining('allowed-tools: "alice.vendor.read extra.tool"')}));
    expect(api.publishReader).toHaveBeenCalledWith('vendor',2,5,expect.stringContaining('Update allowed MCP tools'));
  });

  test('deletes the selected skill declaration and document from the toolbar',async()=>{
    const user=userEvent.setup();
    const snapshot={version:{versionNo:2,sourceRevision:4},files:[{resourceId:'file',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'---\nname: alice-guide\ndescription: Guide\nallowed-tools: alice.vendor.read\n---\nGuide'}],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[{skillId:'guide',folderId:'folder',skillRoot:'.'}]};
    const withoutRoot={...snapshot,version:{versionNo:2,sourceRevision:5},skills:[]};
    const empty={...withoutRoot,version:{versionNo:2,sourceRevision:6},files:[]};
    const api={listMCPSkills:vi.fn().mockResolvedValue([{uri:'skill://alice-guide/SKILL.md',frontmatter:{name:'alice-guide',description:'Guide','allowed-tools':'alice.vendor.read'}}]),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'}]),getResources:vi.fn().mockResolvedValue(snapshot),deleteSkillRoot:vi.fn().mockResolvedValue(withoutRoot),deleteResourceFile:vi.fn().mockResolvedValue(empty),validateVersion:vi.fn().mockResolvedValue({valid:true}),publishReader:vi.fn().mockResolvedValue({status:'active'})};
    render(<RuntimeWorkspace api={api} mode="skills" status={status} loading={false} error="" onRefresh={vi.fn().mockResolvedValue()} onOpenComponent={vi.fn()}/>);
    await screen.findByText('alice-guide');
    await user.click(screen.getByRole('button',{name:'Delete'}));
    await user.click(await screen.findByRole('button',{name:'Delete skill'}));
    expect(api.deleteSkillRoot).toHaveBeenCalledWith('vendor',2,'guide',4);
    expect(api.deleteResourceFile).toHaveBeenCalledWith('vendor',2,'file',5);
    expect(api.publishReader).toHaveBeenCalledWith('vendor',2,6,expect.stringContaining('Remove skill'));
  });
});

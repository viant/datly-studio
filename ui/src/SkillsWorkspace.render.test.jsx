import React from 'react';
import { expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SkillsWorkspace, componentSkills } from './SkillsWorkspace.jsx';
import { StudioApp } from './StudioApp.jsx';

const apiFor = () => ({
  listNamespaceSkills: vi.fn().mockResolvedValue({items:[]}),
  listComponents: vi.fn().mockResolvedValue({items:[{id:'one',title:'Example',currentDraftVersion:3}]}),
  getResources: vi.fn().mockResolvedValue({skills:[{skillId:'guide',skillRoot:'guide'}]}),
  listMCPSkills: vi.fn().mockRejectedValue(new Error('runtime offline')),
  listMCPTools: vi.fn().mockRejectedValue(new Error('runtime offline')),
});

test('component skills can be edited and created with the MCP runtime offline', async () => {
  const api=apiFor(), open=vi.fn(), user=userEvent.setup();
  render(<SkillsWorkspace api={api} namespace={{name:'example'}} onOpenComponent={open}/>);
  await user.click(screen.getByRole('tab',{name:'Component-owned'}));
  await screen.findByText('guide');
  expect(api.getResources).toHaveBeenCalledWith('one',3);
  await user.click(screen.getByRole('button',{name:'Edit skill'}));
  expect(open).toHaveBeenCalledWith(expect.objectContaining({id:'one',versionNo:3,resourceSkillId:'guide',openResources:true}));
  await user.selectOptions(screen.getByLabelText('Component'),'one');
  await user.click(screen.getByRole('button',{name:'New component skill'}));
  expect(open).toHaveBeenLastCalledWith(expect.objectContaining({id:'one',resourceAction:'newSkill'}));
  expect(api.listMCPSkills).not.toHaveBeenCalled();
  expect(api.listMCPTools).not.toHaveBeenCalled();
});

test('namespace-owned skills use the host provider without a component or runtime', async () => {
  const api=apiFor(), namespace={name:'alpha',namespaceId:'a'.repeat(64)}, user=userEvent.setup();
  const renderer=vi.fn(({namespace,onDirtyChange})=><div><p>{namespace.name} skills</p><button onClick={()=>onDirtyChange(true)}>Edit namespace skill</button></div>);
  render(<SkillsWorkspace api={api} namespace={namespace} renderNamespaceSkills={renderer}/>);
  await screen.findByText('alpha skills');
  expect(api.listComponents).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button',{name:'Edit namespace skill'}));
  expect(screen.getByRole('tab',{name:'Component-owned'}).getAttribute('aria-disabled')).toBe('true');
});

test('the Skills navigation does not request runtime status', async () => {
  const api={...apiFor(),listNamespaces:vi.fn().mockResolvedValue({items:[]}),listConnectors:vi.fn().mockResolvedValue({items:[]}),getRuntimeStatus:vi.fn().mockResolvedValue({host:{status:'unavailable'}})};
  render(<StudioApp api={api} mode="development" subject="alice"/>);
  await waitFor(()=>expect(api.getRuntimeStatus).toHaveBeenCalledOnce());
  api.getRuntimeStatus.mockClear();
  await userEvent.setup().click(screen.getByRole('button',{name:'Skills',exact:true}));
  await screen.findByText(/No namespace skills found/);
  expect(api.getRuntimeStatus).not.toHaveBeenCalled();
});

test('component inventory preserves visible rows and reports denied snapshots', async () => {
  const api=apiFor();
  api.listComponents.mockResolvedValue({items:[{id:'one',title:'Example',currentDraftVersion:3},{id:'denied',title:'Restricted',currentDraftVersion:1}]});
  api.getResources.mockImplementation(async id=>{if(id==='denied')throw new Error('forbidden');return {skills:[{skillId:'guide'}]};});
  const catalog=await componentSkills(api);
  expect(catalog.skills).toHaveLength(1);
  expect(catalog.failures).toEqual(['Restricted: forbidden']);
});

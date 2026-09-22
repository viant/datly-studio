import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./LazyEditor.jsx',()=>({LazyEditor:({value,onChange,ariaLabel})=><textarea aria-label={ariaLabel} value={value} onChange={(event)=>onChange(event.target.value)}/> }));
import { ReaderResourcesDialog } from './ReaderResourcesDialog.jsx';
import { skillToolNames, withSkillTools } from './skillFrontmatter.js';

describe('ReaderResourcesDialog',()=>{
  test('keeps a stale resource write unapplied and reloads the exact revision',async()=>{
    const user=userEvent.setup();
    const stale=Object.assign(new Error('resource revision does not match'),{code:'conflict'});
    const empty={version:{versionNo:2,sourceRevision:4},files:[],folders:[],skills:[]};
    const current={version:{versionNo:2,sourceRevision:5},files:[{resourceId:'guide',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'current',contentSize:7,contentSha256:'abcdef12'}],folders:[],skills:[]};
    const api={getResources:vi.fn().mockResolvedValueOnce(empty).mockResolvedValueOnce(current),upsertResourceFile:vi.fn().mockRejectedValue(stale)};
    const onChanged=vi.fn();
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:2,sourceRevision:4}} onClose={vi.fn()} onChanged={onChanged}/>);

    expect(await screen.findByText('Revision 4')).toBeTruthy();
    await user.click(screen.getByRole('tab',{name:'Files (0)'}));
    await user.clear(screen.getByLabelText('Skill markup'));
    await user.type(screen.getByLabelText('Skill markup'),'my unsaved skill');
    await user.click(screen.getByRole('button',{name:'Save file'}));
    expect((await screen.findByRole('alert')).textContent).toContain('No resource change was applied');
    expect(screen.getByLabelText('Skill markup').value).toBe('my unsaved skill');
    expect(onChanged).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button',{name:'Reload resources'}));
    expect(await screen.findByText('Revision 5')).toBeTruthy();
    expect(screen.getByText('alice.docs:guide/SKILL.md')).toBeTruthy();
    expect(api.getResources).toHaveBeenCalledTimes(2);
  });

  test('opens a declared skill in the versioned markup editor',async()=>{
    const user=userEvent.setup();
    const snapshot={version:{versionNo:3,sourceRevision:4},files:[{resourceId:'guide',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'---\nname: guide\n---\nUse it.',contentSize:28,contentSha256:'abcdef12'}],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[{skillId:'skill',folderId:'folder',skillRoot:'.'}]};
    const api={getResources:vi.fn().mockResolvedValue(snapshot)};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={vi.fn()}/>);
    await user.click(await screen.findByRole('button',{name:/\.\/SKILL\.md/i}));
    const editor=screen.getByRole('textbox',{name:'Skill markup'});
    expect(editor.value).toContain('name: guide');
  });

  test('declares every referenced live MCP tool in skill frontmatter',async()=>{
    const user=userEvent.setup();
    const snapshot={version:{versionNo:3,sourceRevision:4},files:[{resourceId:'guide',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'---\nname: guide\ndescription: Guide\nallowed-tools: ""\n---\nUse it.',contentSize:54,contentSha256:'abcdef12'}],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[{skillId:'skill',folderId:'folder',skillRoot:'.'}]};
    const api={getResources:vi.fn().mockResolvedValue(snapshot),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'}])};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={vi.fn()}/>);
    await user.click(await screen.findByRole('button',{name:/\.\/SKILL\.md/i}));
    await user.selectOptions(screen.getByLabelText('Add MCP tool to skill'),'alice.vendor.read');
    await user.click(screen.getByRole('button',{name:'Add'}));
    expect(screen.getByLabelText('Skill markup').value).toContain('allowed-tools: "alice.vendor.read"');
  });

  test('replaces a tools frontmatter block without touching skill prose',()=>{
    const source='---\nname: guide\ndescription: Guide\nallowed-tools: old.tool\n---\nUse it.';
    const updated=withSkillTools(source,['one.tool','two.tool']);
    expect(skillToolNames(updated)).toEqual(['one.tool','two.tool']);
    expect(updated).toContain('---\nUse it.');
  });

  test('blocks a skill save when frontmatter names an unknown live tool',async()=>{
    const snapshot={version:{versionNo:3,sourceRevision:4},files:[],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[]};
    const api={getResources:vi.fn().mockResolvedValue(snapshot),listMCPTools:vi.fn().mockResolvedValue([{name:'known.tool'}]),upsertResourceFile:vi.fn()};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={vi.fn()}/>);
    const editor=await screen.findByRole('textbox',{name:'Skill markup'});
    await userEvent.setup().clear(editor);
    await userEvent.setup().type(editor,'---\nname: guide\ndescription: Guide\nallowed-tools: missing.tool\n---\nUse it.');
    expect((await screen.findByRole('alert')).textContent).toContain('Tool not found in the live MCP catalog');
    expect(screen.getByRole('button',{name:'Create skill'}).disabled).toBe(true);
  });

  test('creates a skill document and declaration from one action',async()=>{
    const user=userEvent.setup();
    const base={version:{versionNo:3,sourceRevision:4},files:[],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'skills',uriPrefix:'skill://alice-skills/'}],skills:[]};
    const withFile={...base,version:{versionNo:3,sourceRevision:5},files:[{resourceId:'file',namespace:'alice.docs',resourcePath:'skills/skill-1/SKILL.md',content:'skill'}]};
    const complete={...withFile,version:{versionNo:3,sourceRevision:6},skills:[{skillId:'skill',folderId:'folder',skillRoot:'skill-1'}]};
    const api={getResources:vi.fn().mockResolvedValue(base),listMCPTools:vi.fn().mockResolvedValue([{name:'alice.vendor.read'}]),upsertResourceFile:vi.fn().mockResolvedValue(withFile),upsertSkillRoot:vi.fn().mockResolvedValue(complete)};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={vi.fn()}/>);
    await user.click(await screen.findByRole('button',{name:'New skill'}));
    await user.selectOptions(screen.getByLabelText('Add MCP tool to skill'),'alice.vendor.read');
    await user.click(screen.getByRole('button',{name:'Add'}));
    await user.click(screen.getByRole('button',{name:'Create skill'}));
    expect(api.upsertResourceFile).toHaveBeenCalledWith(expect.objectContaining({resourcePath:'skills/skill-1/SKILL.md',expectedSourceRevision:4}));
    expect(api.upsertSkillRoot).toHaveBeenCalledWith(expect.objectContaining({skillRoot:'skill-1',expectedSourceRevision:5}));
  });
});

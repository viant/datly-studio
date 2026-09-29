import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./LazyEditor.jsx',()=>({LazyEditor:({value,onChange,ariaLabel})=><textarea aria-label={ariaLabel} value={value} onChange={(event)=>onChange(event.target.value)}/> }));
import { ReaderResourcesDialog } from './ReaderResourcesDialog.jsx';
import { skillToolNames, withSkillTools } from './skillFrontmatter.js';

describe('ReaderResourcesDialog',()=>{
  const sourceSnapshot = () => ({version:{versionNo:3,sourceRevision:4},files:[
    {resourceId:'skill-file',namespace:'alice.docs',resourcePath:'guide/SKILL.md',content:'---\nname: guide\n---\nUse it.'},
    {resourceId:'one',namespace:'alice.docs',resourcePath:'sql/one.sql',content:'SELECT 1',contentSize:8},
    {resourceId:'two',namespace:'alice.docs',resourcePath:'sql/two.sql',content:'SELECT 2',contentSize:8}],
    folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[{skillId:'skill',folderId:'folder',skillRoot:'.'}]});

  test('source resources opens a visible SQL file instead of skill markup',async()=>{
    const api={getResources:vi.fn().mockResolvedValue(sourceSnapshot())};
    render(<ReaderResourcesDialog isOpen mode="resources" api={api} report={{id:'vendor',ownerPackage:'alice'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()}/>);
    expect((await screen.findByLabelText('Text content')).value).toBe('SELECT 1');
    expect(screen.getByLabelText('Resource path').value).toBe('sql/one.sql');
    expect(screen.queryByRole('textbox',{name:'Skill markup'})).toBeNull();
    expect(screen.queryByRole('tab',{name:/Skills/})).toBeNull();
  });

  test('protects resource drafts when switching files or closing',async()=>{
    const user=userEvent.setup();const onClose=vi.fn();
    const api={getResources:vi.fn().mockResolvedValue(sourceSnapshot())};
    render(<ReaderResourcesDialog isOpen mode="resources" api={api} report={{id:'vendor',ownerPackage:'alice'}} version={{versionNo:3,sourceRevision:4}} onClose={onClose}/>);
    const editor=await screen.findByLabelText('Text content');
    await user.clear(editor);await user.type(editor,'SELECT 3');
    await user.click(screen.getByRole('button',{name:'Edit sql/two.sql'}));
    await user.click(screen.getByRole('button',{name:'Keep editing'}));
    expect(editor.value).toBe('SELECT 3');
    expect(screen.getByLabelText('Resource path').value).toBe('sql/one.sql');
    await user.click(screen.getByRole('button',{name:'Edit sql/two.sql'}));
    await user.click(screen.getByRole('button',{name:'Discard changes'}));
    expect(screen.getByLabelText('Text content').value).toBe('SELECT 2');
    await user.type(screen.getByLabelText('Text content'),' -- draft');
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    expect(onClose).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button',{name:'Keep editing'}));
    expect(screen.getByLabelText('Text content').value).toBe('SELECT 2 -- draft');
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    await user.click(screen.getByRole('button',{name:'Discard changes'}));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test('a successful save clears the discard guard',async()=>{
    const user=userEvent.setup();const snapshot=sourceSnapshot();const onClose=vi.fn();
    const api={getResources:vi.fn().mockResolvedValue(snapshot),upsertResourceFile:vi.fn().mockResolvedValue({...snapshot,version:{versionNo:3,sourceRevision:5}})};
    render(<ReaderResourcesDialog isOpen mode="resources" api={api} report={{id:'vendor',ownerPackage:'alice'}} version={{versionNo:3,sourceRevision:4}} onClose={onClose}/>);
    await user.type(await screen.findByLabelText('Text content'),' -- saved');
    await user.click(screen.getByRole('button',{name:'Update file'}));
    await screen.findByText('Revision 5');
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    expect(onClose).toHaveBeenCalledTimes(1);
    await waitFor(()=>expect(screen.queryByText('Discard unsaved resource changes?')).toBeNull());
  });

  test('saving a folder does not clear an unsaved file draft',async()=>{
    const user=userEvent.setup();const snapshot=sourceSnapshot();const onClose=vi.fn();
    const api={getResources:vi.fn().mockResolvedValue(snapshot),upsertResourceFolder:vi.fn().mockResolvedValue({...snapshot,version:{versionNo:3,sourceRevision:5}})};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice'}} version={{versionNo:3,sourceRevision:4}} onClose={onClose}/>);
    await screen.findByText('Revision 4');
    await user.click(screen.getByRole('tab',{name:'Files (3)'}));
    await user.click(screen.getByRole('button',{name:'Edit sql/one.sql'}));
    await user.type(screen.getByLabelText('Text content'),' -- unsaved');
    await user.click(screen.getByRole('tab',{name:'Published folders (1)'}));
    await user.type(screen.getByLabelText('Folder root'),'-new');
    await user.click(screen.getByRole('button',{name:'Update folder'}));
    await screen.findByText('Revision 5');
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    expect(onClose).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button',{name:'Keep editing'}));
    await user.click(screen.getByRole('tab',{name:'Files (3)'}));
    expect(screen.getByLabelText('Text content').value).toBe('SELECT 1 -- unsaved');
  });

  test('discarded folder edits do not reappear after reopening',async()=>{
    const user=userEvent.setup();const onClose=vi.fn();
    const api={getResources:vi.fn().mockResolvedValue(sourceSnapshot())};
    const props={api,report:{id:'vendor',ownerPackage:'alice'},version:{versionNo:3,sourceRevision:4},onClose};
    const {rerender}=render(<ReaderResourcesDialog {...props} isOpen/>);
    await screen.findByText('Revision 4');
    await user.click(screen.getByRole('tab',{name:'Published folders (1)'}));
    await user.type(screen.getByLabelText('Folder root'),'-discarded');
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    await user.click(screen.getByRole('button',{name:'Discard changes'}));
    expect(onClose).toHaveBeenCalledTimes(1);
    rerender(<ReaderResourcesDialog {...props} isOpen={false}/>);
    rerender(<ReaderResourcesDialog {...props} isOpen/>);
    await screen.findByRole('tab',{name:'Published folders (1)'});
    await user.click(screen.getByRole('tab',{name:'Published folders (1)'}));
    await waitFor(()=>expect(screen.getByLabelText('Folder root').value).toBe('guide'));
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    expect(onClose).toHaveBeenCalledTimes(2);
    await waitFor(()=>expect(screen.queryByText('Discard unsaved resource changes?')).toBeNull());
  });
  test('keeps skill markup savable while live MCP discovery is down', async () => {
    const user = userEvent.setup();
    const api = {
      getResources: vi.fn().mockResolvedValue({ version: { versionNo: 3, sourceRevision: 4 }, files: [], folders: [{ folderId: 'folder', namespace: 'alice.docs', rootPath: 'skills', uriPrefix: 'skill://alice-skills/' }], skills: [] }),
      listMCPTools: vi.fn().mockRejectedValue(new Error('runtime offline')),
    };
    render(<ReaderResourcesDialog isOpen api={api} report={{ id: 'vendor', ownerPackage: 'alice', title: 'Vendor' }} version={{ versionNo: 3, sourceRevision: 4, state: 'draft' }} onClose={vi.fn()} />);
    expect((await screen.findByRole('alert')).textContent).toContain('Save is available');
    await user.click(screen.getByRole('tab', { name: 'Files (0)' }));
    expect(screen.getByRole('button', { name: 'Save file' }).disabled).toBe(false);
  });

  test('creates an editable draft from a published skill version', async () => {
    const user = userEvent.setup();
    const snapshot = { version: { versionNo: 4, sourceRevision: 7 }, files: [], folders: [], skills: [] };
    const api = {
      getResources: vi.fn().mockResolvedValue(snapshot),
      getVersion: vi.fn().mockResolvedValue({ sourceRevision: 7, authoringMode: 'dql', authoredDql: 'SELECT 1' }),
      cloneVersion: vi.fn().mockResolvedValue({ versionNo: 8, sourceRevision: 1 }),
      listMCPTools: vi.fn().mockResolvedValue([]),
    };
    const onDraftCreated = vi.fn();
    render(<ReaderResourcesDialog isOpen api={api} report={{ id: 'vendor', ownerPackage: 'alice', title: 'Vendor' }} version={{ versionNo: 4, sourceRevision: 7, state: 'published' }} onClose={vi.fn()} onDraftCreated={onDraftCreated} />);
    await user.click(await screen.findByRole('button', { name: 'Create editable draft' }));
    await waitFor(() => expect(onDraftCreated).toHaveBeenCalledWith({ versionNo: 8, sourceRevision: 1 }));
    expect(api.cloneVersion).toHaveBeenCalledWith('vendor', 4, 7);
  });

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
    await user.click(screen.getByRole('button',{name:'Discard changes'}));
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
    expect((await screen.findByRole('alert')).textContent).toContain('Tool is neither published nor declared by this component');
    expect(screen.getByRole('button',{name:'Create skill'}).disabled).toBe(true);
  });

  test('allows a skill to reference its own unpublished declared tool',async()=>{
    const user=userEvent.setup();
    const snapshot={version:{versionNo:3,sourceRevision:4},files:[],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'guide',uriPrefix:'skill://alice-guide/'}],skills:[]};
    const api={getResources:vi.fn().mockResolvedValue(snapshot),listMCPTools:vi.fn().mockResolvedValue([])};
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} structure={{component:{routes:[{mcp:[{kind:'tool',name:'alice.vendor.draft'}]}]}}} onClose={vi.fn()}/>);
    const editor=await screen.findByRole('textbox',{name:'Skill markup'});
    await user.clear(editor);await user.type(editor,'---\nname: guide\ndescription: Guide\nallowed-tools: alice.vendor.draft\n---\nUse it.');
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.getByRole('button',{name:'Create skill'}).disabled).toBe(false);
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

  test('reports a saved draft revision when the second skill save fails',async()=>{
    const user=userEvent.setup();
    const base={version:{versionNo:3,sourceRevision:4},files:[],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'skills',uriPrefix:'skill://alice-skills/'}],skills:[]};
    const withFile={...base,version:{versionNo:3,sourceRevision:5},files:[{resourceId:'file',namespace:'alice.docs',resourcePath:'skills/skill-1/SKILL.md',content:'---\nname: skill-1\nallowed-tools: ""\n---\nGuide'}]};
    const stale=Object.assign(new Error('skill declaration conflicted'),{code:'conflict'});
    const api={getResources:vi.fn().mockResolvedValueOnce(base).mockResolvedValueOnce(withFile),listMCPTools:vi.fn().mockResolvedValue([]),upsertResourceFile:vi.fn().mockResolvedValue(withFile),upsertSkillRoot:vi.fn().mockRejectedValue(stale)};
    const onChanged=vi.fn();
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={onChanged}/>);
    await user.click(await screen.findByRole('button',{name:'New skill'}));
    await user.click(screen.getByRole('button',{name:'Create skill'}));
    const alert=await screen.findByRole('alert');
    expect(alert.textContent).toContain('source revision 5');
    expect(alert.textContent).toContain('serving reader was not republished');
    expect(alert.textContent).not.toContain('No resource change was applied');
    expect(onChanged).toHaveBeenCalledWith(withFile.version);
    await user.click(screen.getByRole('button',{name:'Reload resources'}));
    expect(await screen.findByText('Revision 5')).toBeTruthy();
  });

  test('reports a saved draft revision when the second skill deletion fails',async()=>{
    const user=userEvent.setup();
    const base={version:{versionNo:3,sourceRevision:4},files:[{resourceId:'file',namespace:'alice.docs',resourcePath:'skills/SKILL.md',content:'---\nname: guide\n---\nGuide'}],folders:[{folderId:'folder',namespace:'alice.docs',rootPath:'skills',uriPrefix:'skill://alice-skills/'}],skills:[{skillId:'guide',folderId:'folder',skillRoot:'.'}]};
    const withoutRoot={...base,version:{versionNo:3,sourceRevision:5},skills:[]};
    const api={getResources:vi.fn().mockResolvedValue(base),listMCPTools:vi.fn().mockResolvedValue([]),deleteSkillRoot:vi.fn().mockResolvedValue(withoutRoot),deleteResourceFile:vi.fn().mockRejectedValue(new Error('file deletion failed'))};
    const onChanged=vi.fn();
    render(<ReaderResourcesDialog isOpen api={api} report={{id:'vendor',ownerPackage:'alice',title:'Vendor'}} version={{versionNo:3,sourceRevision:4}} onClose={vi.fn()} onChanged={onChanged}/>);
    await user.click(await screen.findByRole('button',{name:'Delete skill .'}));
    await user.click(screen.getByRole('button',{name:'Delete skill'}));
    const alert=await screen.findByRole('alert');
    expect(alert.textContent).toContain('source revision 5');
    expect(alert.textContent).toContain('file deletion failed');
    expect(onChanged).toHaveBeenCalledWith(withoutRoot.version);
  });
});

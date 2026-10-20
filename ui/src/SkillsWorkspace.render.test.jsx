import React from 'react';
import {EditorView} from '@codemirror/view';
import {act,render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {test,expect,vi} from 'vitest';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';
let serial=0;
const namespace={namespaceId:'a'.repeat(64),name:'sales',title:'Sales',canManage:true,mcpEnabled:false};
const content='---\nname: sales-guide\ndescription: Query sales records.\n---\nUse sales tools.';
const skill={skillId:'guide',name:'sales-guide',title:'Sales guide',description:'Query sales records.',latestRevision:1,publishedRevision:0,etag:1};
function mount({items=[],scope=namespace,conflict=false,slots,failResources=false}={}) {
 const navigate=vi.fn(),dirty=vi.fn();
 const api={readManagementResource:async()=>managementResourceFixture('skills'),fetchMCPDataSource:vi.fn(async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.namespace_skills.list':return {items};
   case 'studio.sdk.namespace_skills.get':return {skill,revision:{revision:1,bundle:{title:'Sales guide',content,files:[{path:'binary.dat',content:'AAEC',isBinary:true}]}}};
   case 'studio.sdk.namespace_skills.create':return {skill,revision:{revision:1,bundle:{content}}};
   case 'studio.sdk.namespace_skills.update':if(conflict)throw Object.assign(new Error('expected revision changed'),{code:'conflict'});return {skill:{...skill,etag:2,latestRevision:2}};
   case 'studio.sdk.namespace_skills.publish':return {skill:{...skill,etag:2,publishedRevision:1}};
   case 'studio.sdk.namespace_skills.unpublish':return {skill};
   case 'studio.sdk.namespace_skills.delete':return {};
   case 'studio.sdk.namespace_skills.revisions':return {items:[{revision:1,createdBy:'owner',bundle:{content}}]};
   case 'studio.sdk.components.list':return {items:[{id:'one',title:'Example',currentDraftVersion:3},...(failResources?[{id:'denied',title:'Restricted',currentDraftVersion:1}]:[])]};
   case 'studio.sdk.resources.get':if(input.reportId==='denied')throw new Error('forbidden');return {skills:[{skillId:'guide',skillRoot:'guide'}]};
   default:throw new Error('Unexpected '+source.backend.method);
  }
 })};
 const view=render(<ManagementWindowRuntime api={api} uri="window://datastudio/skills" instanceKey={'forge-skills-'+(++serial)} parameters={{currentNamespace:scope,hasNamespaceSkillsRenderer:Boolean(slots)}} renderSlots={slots} onNavigate={navigate} onDirtyChange={dirty}/>);
 return {api,navigate,dirty,view};
}
async function editText(label,text){const node=await screen.findByRole('textbox',{name:label,exact:true});const editor=EditorView.findFromDOM(node.closest('.cm-editor'));await act(async()=>editor.dispatch({changes:{from:0,to:editor.state.doc.length,insert:text}}));return node;}
test('metadata namespace skills create independent drafts and supporting files with MCP publication disabled',async()=>{
 const user=userEvent.setup(),{api}=mount();await user.click(await screen.findByRole('button',{name:'New namespace skill',exact:true}));
 await user.type(screen.getByLabelText('Display title',{exact:true}),'Sales guide');await editText('SKILL.md',content);
 await user.click(screen.getByRole('tab',{name:'Supporting files',exact:true}));await user.click(screen.getByRole('button',{name:'Add supporting file',exact:true}));
 await user.type(await screen.findByLabelText('Relative path',{exact:true}),'references/query.md');await user.type(screen.getByLabelText('Content',{exact:true}),'Use namespace tools.');await user.click(screen.getByRole('button',{name:'Save supporting file',exact:true}));await user.click(screen.getByRole('button',{name:'Save namespace skill',exact:true}));
 expect(await screen.findByText('Saved sales-guide, revision 1.')).toBeTruthy();
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespace_skills.create')[1]).toEqual({bundle:{title:'Sales guide',content,files:[{path:'references/query.md',content:'Use namespace tools.'}]}});
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespace_skills.publish')).toBe(false);
});
test('metadata namespace skill CAS conflict preserves edits and binary support files until explicit reload',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[skill],conflict:true});await user.click(await screen.findByRole('button',{name:'Edit namespace skill',exact:true}));await editText('SKILL.md',content+'\nKeep my changes.');
 expect(screen.getByRole('tab',{name:'Component-owned',exact:true}).getAttribute('aria-disabled')).toBe('true');
 await user.click(screen.getByRole('button',{name:'Save namespace skill',exact:true}));await screen.findByText('Skill changed elsewhere: expected revision changed');expect(screen.getByRole('textbox',{name:'SKILL.md',exact:true}).textContent).toContain('Keep my changes.');expect(screen.getByRole('button',{name:'Save namespace skill',exact:true}).disabled).toBe(true);
 expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespace_skills.update')[1].bundle.files).toEqual([{path:'binary.dat',content:'AAEC',isBinary:true}]);
 await user.click(screen.getByRole('button',{name:'Discard edits and reload',exact:true}));await waitFor(()=>expect(screen.getByRole('textbox',{name:'SKILL.md',exact:true}).textContent).not.toContain('Keep my changes.'));
});
test('metadata namespace publication uses the saved revision and etag and respects namespace MCP settings',async()=>{
 const user=userEvent.setup(),first=mount({items:[skill]});expect((await screen.findByRole('button',{name:'Publish namespace skill',exact:true})).disabled).toBe(true);first.view.unmount();
 const {api}=mount({items:[skill],scope:{...namespace,mcpEnabled:true}});await user.click(await screen.findByRole('button',{name:'Publish namespace skill',exact:true}));await screen.findByText('Published sales-guide, revision 1.');expect(api.fetchMCPDataSource.mock.calls.find(([source])=>source.backend.method==='studio.sdk.namespace_skills.publish')[1]).toEqual({skillId:'guide',etag:1,revision:1});
});
test('metadata component skills use the authorized version catalog and retain healthy siblings on failures',async()=>{
 const user=userEvent.setup(),{api,navigate}=mount({failResources:true});await screen.findByRole('button',{name:'New namespace skill',exact:true});await user.click(screen.getByRole('tab',{name:'Component-owned',exact:true}));await screen.findByText('guide');await screen.findByText(/Restricted: forbidden/);
 await user.click(screen.getByRole('button',{name:'Edit component skill',exact:true}));expect(navigate).toHaveBeenCalledWith('builder',{report:expect.objectContaining({id:'one',versionNo:3,resourceSkillId:'guide',openResources:true})});
 await user.click(screen.getByRole('button',{name:'Component',exact:true}));await user.click(await screen.findByRole('menuitem',{name:'Example',exact:true}));await user.click(screen.getByRole('button',{name:'New component skill',exact:true}));expect(navigate).toHaveBeenLastCalledWith('builder',{report:expect.objectContaining({id:'one',resourceAction:'newSkill'})});
 expect(api.fetchMCPDataSource.mock.calls.every(([source])=>!source.backend.method.startsWith('studio.sdk.runtime'))).toBe(true);
});
test('metadata namespace skills keep the existing optional host renderer and edit guard',async()=>{
 const renderer=vi.fn(props=><div>Sales host skills<button onClick={()=>props.onDirtyChange(true)}>Edit host skill</button></div>),user=userEvent.setup(),{api,dirty}=mount({slots:{'namespace-skills':renderer}});
 await user.click(await screen.findByRole('button',{name:'Edit host skill'}));expect(dirty).toHaveBeenCalledWith(true);expect(screen.getByRole('tab',{name:'Component-owned',exact:true}).getAttribute('aria-disabled')).toBe('true');expect(api.fetchMCPDataSource).not.toHaveBeenCalled();
});
test('metadata namespace skills do not discover or offer mutations to a namespace viewer',async()=>{
 const {api}=mount({scope:{...namespace,canManage:false}});await screen.findByText('Namespace skills are managed by the namespace owner and namespace administrators.');expect(screen.queryByRole('button',{name:'New namespace skill',exact:true})).toBeNull();expect(api.fetchMCPDataSource).not.toHaveBeenCalled();
});

test('metadata namespace skills keep bounded server pages and revision inspection',async()=>{
 const user=userEvent.setup(),{api}=mount({items:Array.from({length:21},(_,index)=>({...skill,skillId:'skill-'+index,name:'guide-'+index,title:'Guide '+index}))});await screen.findByText('Guide 0');expect(screen.queryByText('Guide 20')).toBeNull();
 await user.click(screen.getByRole('button',{name:'Next',exact:true}));await waitFor(()=>expect(api.fetchMCPDataSource.mock.calls.some(([source,input])=>source.backend.method==='studio.sdk.namespace_skills.list'&&input.limit===21&&input.offset===20)).toBe(true));
});
test('metadata namespace skill history is a read-only saved revision',async()=>{
 const user=userEvent.setup(),{api}=mount({items:[skill]});await user.click(await screen.findByRole('button',{name:'Skill revision history',exact:true}));
 await user.click(await screen.findByRole('button',{name:'View skill revision',exact:true}));expect((await screen.findByRole('textbox',{name:'Revision content',exact:true})).textContent).toContain('Use sales tools.');
 expect(api.fetchMCPDataSource.mock.calls.some(([source])=>source.backend.method==='studio.sdk.namespace_skills.update'||source.backend.method==='studio.sdk.namespace_skills.publish')).toBe(false);
});

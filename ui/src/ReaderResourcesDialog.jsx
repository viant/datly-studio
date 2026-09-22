import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Button, ButtonGroup, Callout, Code, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tab, Tabs, Tag, TextArea } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';
import { skillToolNames, withSkillTools } from './skillFrontmatter.js';

export function ReaderResourcesDialog({ isOpen, initialAction = '', api, report, version, onClose, onChanged }) {
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const namespace = `${report?.ownerPackage || 'user'}.docs`;
  const [file, setFile] = useState({ namespace, resourcePath: 'guide/SKILL.md', content: skillTemplate(report) });
  const [folder, setFolder] = useState({ namespace, rootPath: 'guide', uriPrefix: `skill://${report?.ownerPackage || 'user'}-guide/` });
  const [skill, setSkill] = useState({ folderId: '', skillRoot: '.' });
  const [pendingDelete, setPendingDelete] = useState(null);
  const [conflict, setConflict] = useState(null);
  const [mcpTools, setMCPTools] = useState([]);
  const [toolToAdd, setToolToAdd] = useState('');
  const [toolsLoading, setToolsLoading] = useState(false);
  const [toolsError, setToolsError] = useState('');
  const [activeTab, setActiveTab] = useState('skills');

  const selectSkillFrom = (item, folderItems, fileItems) => {
    setSkill({skillId:item.skillId,folderId:item.folderId,skillRoot:item.skillRoot,ordinal:item.ordinal||0});
    const parent=folderItems.find((folderItem)=>folderItem.folderId===item.folderId);
    if(!parent)return;
    const relative=item.skillRoot==='.'?'':`${item.skillRoot}/`;
    const resourcePath=`${parent.rootPath}/${relative}SKILL.md`;
    const existing=fileItems.find((fileItem)=>fileItem.namespace===parent.namespace&&fileItem.resourcePath===resourcePath);
    setFile(existing?{resourceId:existing.resourceId,namespace:existing.namespace,resourcePath:existing.resourcePath,mediaType:existing.mediaType||'',content:existing.content||''}:{namespace:parent.namespace,resourcePath,content:skillTemplate(report,item.skillRoot==='.'?parent.rootPath:item.skillRoot)});
  };

  const load = async () => {
    if (!report || !version) return;
    setLoading(true); setError(''); setConflict(null);
    try {
      const next = await api.getResources(report.id, version.versionNo);
      setSnapshot(next);
      if(initialAction==='newSkill'&&next?.folders?.[0]){
        const parent=next.folders[0];let index=(next.skills?.length??0)+1;let name=`skill-${index}`;const used=new Set((next.skills??[]).map((item)=>item.skillRoot));while(used.has(name)){index++;name=`skill-${index}`;}
        setSkill({folderId:parent.folderId,skillRoot:name,ordinal:next.skills?.length??0});setFile({namespace:parent.namespace,resourcePath:`${parent.rootPath}/${name}/SKILL.md`,content:skillTemplate(report,name)});
      } else if(next?.skills?.[0])selectSkillFrom(next.skills[0],next.folders??[],next.files??[]);
      else if (!skill.folderId && next?.folders?.[0]?.folderId) setSkill((current) => ({ ...current, folderId: next.folders[0].folderId }));
    } catch (cause) { if(cause?.code==='conflict')setConflict(cause);else setError(cause.message); }
    finally { setLoading(false); }
  };
  useEffect(() => { if (isOpen) { setPendingDelete(null); setActiveTab('skills'); load(); } }, [isOpen, report?.id, version?.versionNo]);
  useEffect(() => { setFile((current) => ({ ...current, namespace })); setFolder((current) => ({ ...current, namespace, uriPrefix: `skill://${report?.ownerPackage || 'user'}-guide/` })); }, [namespace, report?.ownerPackage]);
  const loadMCPTools = () => {
    if (!api?.listMCPTools) return;
    let cancelled = false;
    setToolsLoading(true); setToolsError('');
    api.listMCPTools().then((tools) => { if (!cancelled) setMCPTools(tools); }).catch((cause) => { if (!cancelled) setToolsError(cause.message || 'Unable to load live MCP tools.'); }).finally(() => { if (!cancelled) setToolsLoading(false); });
    return () => { cancelled = true; };
  };
  useEffect(() => { if (!isOpen) return; return loadMCPTools(); }, [api, isOpen]);

  const apply = async (action) => {
    setSaving(true); setError('');
    try {
      const next = await action();
      setSnapshot(next);
      onChanged?.(next.version);
      if (!skill.folderId && next?.folders?.[0]?.folderId) setSkill((current) => ({ ...current, folderId: next.folders[0].folderId }));
    } catch (cause) { if(cause?.code==='conflict')setConflict(cause);else setError(cause.message); }
    finally { setSaving(false); }
  };
  const folders = snapshot?.folders ?? [];
  const files = snapshot?.files ?? [];
  const skills = snapshot?.skills ?? [];
  const revision = snapshot?.version?.sourceRevision ?? version?.sourceRevision;
  const resetFile=()=>{setFile({namespace,resourcePath:'docs/note.md',content:''});setActiveTab('files');};
  const resetFolder=()=>{setFolder({namespace,rootPath:'guide',uriPrefix:`skill://${report?.ownerPackage||'user'}-guide/`});setActiveTab('folders');};
  const resetSkill=()=>{setSkill({folderId:folders[0]?.folderId||'',skillRoot:'.'});setActiveTab('skills');};
  const skillFileFor=(item)=>{
    const parent=folders.find((folderItem)=>folderItem.folderId===item.folderId);
    if(!parent)return null;
    const relative=[parent.rootPath,item.skillRoot,'.'].filter((part,index)=>index===0||part!=='.').join('/').replace(/\/\.+\//g,'/');
    const resourcePath=`${relative.replace(/\/$/,'')}/SKILL.md`;
    return files.find((fileItem)=>fileItem.namespace===parent.namespace&&fileItem.resourcePath===resourcePath)||{namespace:parent.namespace,resourcePath,content:skillTemplate(report,item.skillRoot==='.'?parent.rootPath:item.skillRoot)};
  };
  const newSkill=()=>{
    const parent=folders[0];
    if(!parent){setActiveTab('folders');return;}
    let index=skills.length+1;
    let name=`skill-${index}`;
    const used=new Set(skills.map((item)=>item.skillRoot));
    while(used.has(name)){index++;name=`skill-${index}`;}
    setSkill({folderId:parent.folderId,skillRoot:name,ordinal:skills.length});
    setFile({namespace:parent.namespace,resourcePath:`${parent.rootPath}/${name}/SKILL.md`,content:skillTemplate(report,name)});
    setActiveTab('skills');
  };
  const remove=async()=>{
    if(!pendingDelete)return;
    const target=pendingDelete;
    await apply(async()=>{
      let next;
      if(target.kind==='file')next=await api.deleteResourceFile(report.id,version.versionNo,target.item.resourceId,revision);
      else if(target.kind==='folder')next=await api.deleteResourceFolder(report.id,version.versionNo,target.item.folderId,revision);
      else {
        const associated=skillFileFor(target.item);
        next=await api.deleteSkillRoot(report.id,version.versionNo,target.item.skillId,revision);
        if(associated?.resourceId)next=await api.deleteResourceFile(report.id,version.versionNo,associated.resourceId,next.version.sourceRevision);
      }
      setPendingDelete(null);
      if(target.kind==='file'&&file.resourceId===target.item.resourceId)resetFile();
      if(target.kind==='folder'&&folder.folderId===target.item.folderId)resetFolder();
      if(target.kind==='skill'&&skill.skillId===target.item.skillId)resetSkill();
      return next;
    });
  };
  const openSkill=(item)=>{
	setActiveTab('skills');
    selectSkillFrom(item,folders,files);
  };
  const skillMarkup=/(^|\/)SKILL\.md$/i.test(file.resourcePath);
  const skillTools = skillMarkup ? skillToolNames(file.content) : [];
  const duplicateTools = skillTools.filter((name, index) => skillTools.indexOf(name) !== index);
  const unknownTools = !toolsLoading && !toolsError ? skillTools.filter((name) => !mcpTools.some((item) => item.name === name)) : [];
  const skillToolError = duplicateTools.length ? `Tool list has duplicates: ${[...new Set(duplicateTools)].join(', ')}.` : unknownTools.length ? `Tool not found in the live MCP catalog: ${unknownTools.join(', ')}.` : '';
  const canSaveFile = !saving && (!skillMarkup || (!toolsLoading && !toolsError && !skillToolError));
  const addSkillTool = () => {
    if (!toolToAdd || skillTools.includes(toolToAdd)) return;
    setFile({ ...file, content: withSkillTools(file.content, [...skillTools, toolToAdd]) });
    setToolToAdd('');
  };
  const removeSkillTool = (name) => setFile({ ...file, content: withSkillTools(file.content, skillTools.filter((item) => item !== name)) });
  const updateSkillLocation=(folderId,skillRoot)=>{
    const parent=folders.find((item)=>item.folderId===folderId);
    const root=skillRoot||'.';
    setSkill({...skill,folderId,skillRoot:root});
    if(parent){const relative=root==='.'?'':`${root}/`;setFile({...file,namespace:parent.namespace,resourcePath:`${parent.rootPath}/${relative}SKILL.md`});}
  };
  const saveSkill=()=>apply(async()=>{
    const withFile=await api.upsertResourceFile({reportId:report.id,versionNo:version.versionNo,expectedSourceRevision:revision,...file});
    return api.upsertSkillRoot({reportId:report.id,versionNo:version.versionNo,expectedSourceRevision:withFile.version.sourceRevision,...skill});
  });

  return <Dialog className="studio-connector-dialog studio-resources-dialog" isOpen={isOpen} onClose={onClose} title="Resources & skills" icon="folder-open" canOutsideClickClose={!saving}>
    <DialogBody className="studio-connector-dialog-body">
      <div className="studio-validation-revision"><span>Revision {revision ?? '—'}</span><Tag minimal>{files.length} files · {folders.length} folders · {skills.length} skills</Tag></div>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      {conflict&&<Callout intent="warning" title="Resources changed elsewhere" role="alert">No resource change was applied. Reload the exact version before reviewing and retrying your edit.<Button small minimal intent="warning" icon="refresh" onClick={load}>Reload resources</Button></Callout>}
      {loading && <div className="studio-muted">Loading versioned resources…</div>}
      {!loading && <>
        <Tabs id="resource-tabs" selectedTabId={activeTab} onChange={setActiveTab} className="studio-resource-tabs"><Tab id="skills" title={`Skills (${skills.length})`}/><Tab id="files" title={`Files (${files.length})`}/><Tab id="folders" title={`Published folders (${folders.length})`}/></Tabs>
        {activeTab==='files'&&<section className="studio-resource-section">
          <div className="studio-resource-section-heading"><h3>Files</h3>{file.resourceId&&<Button small icon="add" onClick={resetFile}>New file</Button>}</div>
          <FormGroup label="Namespace" labelFor="resource-namespace" helperText={`Must begin with ${report?.ownerPackage || 'user'}. and is reserved to this reader.`}><InputGroup id="resource-namespace" value={file.namespace} onChange={(event) => setFile({ ...file, namespace: event.target.value })} disabled={saving}/></FormGroup>
          <FormGroup label="Resource path" labelFor="resource-path"><InputGroup id="resource-path" value={file.resourcePath} onChange={(event) => setFile({ ...file, resourcePath: event.target.value })} disabled={saving}/></FormGroup>
          <FormGroup label={skillMarkup?'Skill markup':'Text content'} labelFor={skillMarkup?'skill-markup-editor':'resource-content'}><div className={skillMarkup?'studio-code-editor studio-skill-editor':''}>{skillMarkup?<Editor id="skill-markup-editor" ariaLabel="Skill markup" value={file.content} onChange={(content)=>setFile({...file,content})} language="yaml" height="260px" readOnly={saving}/>:<TextArea id="resource-content" value={file.content} onChange={(event) => setFile({ ...file, content: event.target.value })} fill rows={7} className="studio-code-input" disabled={saving}/>}</div></FormGroup>
          <Button icon="floppy-disk" intent="primary" loading={saving} disabled={!canSaveFile} onClick={() => apply(() => api.upsertResourceFile({ reportId: report.id, versionNo: version.versionNo, expectedSourceRevision: revision, ...file }))}>{file.resourceId?'Update file':'Save file'}</Button>
          {files.length > 0 && <div className="studio-resource-list">{files.map((item) => <div key={item.resourceId}><button type="button" onClick={()=>setFile({resourceId:item.resourceId,namespace:item.namespace,resourcePath:item.resourcePath,mediaType:item.mediaType||'',content:item.content||''})}><Code>{item.namespace}:{item.resourcePath}</Code><small>{item.contentSize} B</small></button><ButtonGroup minimal><Tag minimal>{item.contentSha256?.slice(0,8)||'stored'}</Tag><Button small icon="edit" aria-label={`Edit ${item.resourcePath}`} onClick={()=>setFile({resourceId:item.resourceId,namespace:item.namespace,resourcePath:item.resourcePath,mediaType:item.mediaType||'',content:item.content||''})}/><Button small icon="trash" intent="danger" aria-label={`Delete ${item.resourcePath}`} onClick={()=>setPendingDelete({kind:'file',item})}/></ButtonGroup></div>)}</div>}
        </section>}
        {activeTab==='folders'&&<section className="studio-resource-section">
          <div className="studio-resource-section-heading"><h3>Published folders</h3>{folder.folderId&&<Button small icon="add" onClick={resetFolder}>New folder</Button>}</div>
          <FormGroup label="Folder root" labelFor="resource-root"><InputGroup id="resource-root" value={folder.rootPath} onChange={(event) => setFolder({ ...folder, rootPath: event.target.value })} disabled={saving}/></FormGroup>
          <FormGroup label="URI prefix" labelFor="resource-uri"><InputGroup id="resource-uri" value={folder.uriPrefix} onChange={(event) => setFolder({ ...folder, uriPrefix: event.target.value })} disabled={saving}/></FormGroup>
          <Button icon="folder-new" loading={saving} onClick={() => apply(() => api.upsertResourceFolder({ reportId: report.id, versionNo: version.versionNo, expectedSourceRevision: revision, ...folder }))}>{folder.folderId?'Update folder':'Save folder'}</Button>
          {folders.length > 0 && <div className="studio-resource-list">{folders.map((item) => <div key={item.folderId}><button type="button" onClick={()=>setFolder({folderId:item.folderId,namespace:item.namespace,rootPath:item.rootPath,uriPrefix:item.uriPrefix,ordinal:item.ordinal||0})}><Code>{item.namespace}:{item.rootPath}</Code><small>{item.uriPrefix}</small></button><ButtonGroup minimal><Button small icon="edit" aria-label={`Edit folder ${item.rootPath}`} onClick={()=>setFolder({folderId:item.folderId,namespace:item.namespace,rootPath:item.rootPath,uriPrefix:item.uriPrefix,ordinal:item.ordinal||0})}/><Button small icon="trash" intent="danger" aria-label={`Delete folder ${item.rootPath}`} onClick={()=>setPendingDelete({kind:'folder',item})}/></ButtonGroup></div>)}</div>}
        </section>}
        {activeTab==='skills'&&<section className="studio-resource-section">
          <div className="studio-resource-section-heading"><h3>Skills</h3><Button small icon="add" onClick={newSkill}>New skill</Button></div>
          {folders.length===0?<Callout intent="primary">Create a published folder before adding a skill.<Button small minimal onClick={()=>setActiveTab('folders')}>Open folders</Button></Callout>:<>
          {skills.length > 0 && <div className="studio-resource-list studio-skill-list">{skills.map((item) => <div key={item.skillId}><button type="button" onClick={()=>openSkill(item)}><Code>{item.skillRoot}/SKILL.md</Code><small>{skillFileFor(item)?.resourcePath||'SKILL.md'}</small></button><ButtonGroup minimal><Button small icon="edit" aria-label={`Edit skill ${item.skillRoot}`} onClick={()=>openSkill(item)}/><Button small icon="trash" intent="danger" aria-label={`Delete skill ${item.skillRoot}`} onClick={()=>setPendingDelete({kind:'skill',item})}/></ButtonGroup></div>)}</div>}
          <div className="studio-skill-editor-panel">
          <div className="studio-form-grid"><FormGroup label="Published folder" labelFor="skill-folder"><HTMLSelect id="skill-folder" value={skill.folderId} onChange={(event)=>updateSkillLocation(event.target.value,skill.skillRoot)} fill disabled={saving}><option value="">Select a folder</option>{folders.map((item) => <option key={item.folderId} value={item.folderId}>{item.rootPath}</option>)}</HTMLSelect></FormGroup><FormGroup label="Skill root" labelFor="skill-root"><InputGroup id="skill-root" value={skill.skillRoot} onChange={(event)=>updateSkillLocation(skill.folderId,event.target.value)} disabled={saving}/></FormGroup></div>
          <FormGroup label="SKILL.md" labelFor="skill-markup-editor"><div className="studio-code-editor studio-skill-editor"><Editor id="skill-markup-editor" ariaLabel="Skill markup" value={file.content} onChange={(content)=>setFile({...file,content})} language="yaml" height="260px" readOnly={saving}/></div></FormGroup>
          <section className="studio-skill-tools" aria-labelledby="skill-tools-title"><div><h4 id="skill-tools-title">Allowed MCP tools</h4></div>{toolsError&&<Callout intent="danger" role="alert" title="Live MCP tools unavailable">{toolsError}<Button small minimal intent="danger" onClick={loadMCPTools}>Retry</Button></Callout>}{skillToolError&&<Callout intent="danger" role="alert">{skillToolError}</Callout>}<div className="studio-skill-tool-picker"><HTMLSelect aria-label="Add MCP tool to skill" value={toolToAdd} onChange={(event)=>setToolToAdd(event.target.value)} disabled={saving||toolsLoading||Boolean(toolsError)}><option value="">{toolsLoading?'Loading MCP tools…':'Choose an MCP tool'}</option>{mcpTools.filter((item)=>!skillTools.includes(item.name)).map((item)=><option key={item.name} value={item.name}>{item.name}</option>)}</HTMLSelect><Button icon="add" onClick={addSkillTool} disabled={!toolToAdd||saving||toolsLoading||Boolean(toolsError)}>Add</Button></div><div className="studio-skill-tool-list">{skillTools.length?skillTools.map((name)=><Tag key={name} interactive rightIcon="small-cross" onRemove={()=>removeSkillTool(name)}>{name}</Tag>):<span className="studio-muted">No tools assigned</span>}</div></section>
          <Button icon="floppy-disk" intent="primary" loading={saving} disabled={!skill.folderId||!canSaveFile} onClick={saveSkill}>{skill.skillId?'Save skill':'Create skill'}</Button>
          </div></>}
        </section>}
      </>}
    </DialogBody>
    <DialogFooter actions={<Button onClick={onClose} disabled={saving}>Close</Button>} />
    <Alert isOpen={Boolean(pendingDelete)} intent="danger" icon="trash" confirmButtonText={`Delete ${pendingDelete?.kind||'resource'}`} cancelButtonText="Cancel" loading={saving} onCancel={()=>setPendingDelete(null)} onConfirm={remove} canEscapeKeyCancel canOutsideClickCancel><p>Delete <strong>{resourceDeleteLabel(pendingDelete)}</strong> from this exact reader version?</p><p className="studio-muted">Deleting a folder is blocked while a declared skill still depends on it. Every successful change invalidates validation for this source revision.</p></Alert>
  </Dialog>;
}

function resourceDeleteLabel(target){if(!target)return'';if(target.kind==='file')return `${target.item.namespace}:${target.item.resourcePath}`;if(target.kind==='folder')return `${target.item.namespace}:${target.item.rootPath}`;return `${target.item.skillRoot}/SKILL.md`;}

function skillTemplate(report, requestedName = '') {
  const name = (requestedName || `${report?.ownerPackage || 'user'}-guide`).toLowerCase().replace(/[^a-z0-9-]+/g,'-').replace(/^-+|-+$/g,'');
  return `---\nname: ${name}\ndescription: Describe how to use this reader.\nallowed-tools: ""\n---\n\nUse the declared MCP tools and resources.`;
}

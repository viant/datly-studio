import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Button, ButtonGroup, Callout, Card, Code, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, HTMLTable, InputGroup, Spinner, Tab, Tabs, Tag } from '@blueprintjs/core';
import { skillToolNames, withSkillTools } from './skillFrontmatter.js';

const pageSize = 10;

export function RuntimeWorkspace({ api, mode = 'runtime', status, loading, error, onRefresh, onOpenComponent }) {
  const [activeTab, setActiveTab] = useState('components');
  const [queries, setQueries] = useState({ components: '', tools: '', resources: '' });
  const [pages, setPages] = useState({ components: 1, tools: 1, resources: 1 });
  const [mcpTools, setMCPTools] = useState([]);
  const [mcpLoading, setMCPLoading] = useState(false);
  const [mcpError, setMCPError] = useState('');
  const [mcpSkills, setMCPSkills] = useState([]);
  const [skillsError, setSkillsError] = useState('');
  const [selectedTool, setSelectedTool] = useState(null);
  const [catalogRefresh, setCatalogRefresh] = useState(0);
  const [createSkillOpen, setCreateSkillOpen] = useState(false);
  const [skillOwnerID, setSkillOwnerID] = useState('');
  const readers = status?.readers ?? [];
  const resources = readers.flatMap((reader) => (reader.mcpResources ?? []).map((resource) => ({ ...resource, reader })));
  const skills = readers.flatMap((reader) => (reader.skills ?? []).map((skill) => ({ ...skill, reader })));
  const skillsMode = mode === 'skills';
  const hostReady = status?.host?.status === 'ready';
  const componentItems = useMemo(() => search(readers, queries.components, (item) => `${item.title} ${item.namespace} ${item.connectorName} ${item.runtimeRevision}`), [readers, queries.components]);
  const toolItems = useMemo(() => search(mcpTools, queries.tools, (item) => `${item.name} ${item.description ?? ''}`), [mcpTools, queries.tools]);
  const resourceItems = useMemo(() => search(resources, queries.resources, (item) => `${item.uriPrefix} ${item.namespace} ${item.rootPath} ${item.reader.title}`), [resources, queries.resources]);

  useEffect(() => {
    if (!api) return;
    let cancelled = false;
    setMCPLoading(true); setMCPError('');
    if (skillsMode) {
      setSkillsError('');
      Promise.all([api.listMCPSkills(),api.listMCPTools()]).then(([items,tools]) => { if (!cancelled) { setMCPSkills(items); setMCPTools(tools); } }).catch((cause) => !cancelled && setSkillsError(cause.message)).finally(() => !cancelled && setMCPLoading(false));
    } else api.listMCPTools().then((tools) => { if (!cancelled) setMCPTools(tools); }).catch((cause) => !cancelled && setMCPError(cause.message)).finally(() => !cancelled && setMCPLoading(false));
    return () => { cancelled = true; };
  }, [api, skillsMode, status?.activeGeneration, catalogRefresh]);

  const setQuery = (tab, value) => { setQueries((current) => ({ ...current, [tab]: value })); setPages((current) => ({ ...current, [tab]: 1 })); };
  const setPage = (tab, value) => setPages((current) => ({ ...current, [tab]: value }));
  const title = skillsMode ? 'Skills & Resources' : 'Runtime';
  const description = skillsMode ? 'Create and publish reusable skills for MCP hosts.' : 'Inspect the active generation and discover the live Datly contracts it serves.';
  const refresh = async () => { await onRefresh?.(); setCatalogRefresh((current) => current + 1); };
  const createSkill = () => {
    const owner = readers.find((item) => item.reportId === skillOwnerID) ?? readers[0];
    if (!owner) return;
    setCreateSkillOpen(false);
    onOpenComponent?.({id:owner.reportId,title:owner.title,namespace:owner.namespace,ownerPackage:owner.ownerPackage,defaultConnectorName:owner.connectorName,versionNo:owner.versionNo,openResources:true,resourceAction:'newSkill'});
  };

  return <main className="studio-workspace studio-runtime-workspace">
    <div className="studio-runtime-header"><div><h1 className="studio-page-heading">{title}</h1><p className="studio-page-description">{description}</p></div><Button icon="refresh" loading={loading} onClick={refresh}>Refresh</Button></div>
    {error && <Callout intent="danger" title="Runtime status unavailable" role="alert">{error}<Button small minimal intent="danger" onClick={refresh}>Retry</Button></Callout>}
    {loading && !status ? <div className="studio-loading"><Spinner size={28}/></div> : status && <>
      {!skillsMode&&<section className="studio-generation-strip" aria-label="Recorded runtime generation"><Tag intent={status.status === 'active' ? 'success' : 'warning'}>{status.status} record</Tag><strong>{status.activeGeneration ? `Generation ${status.activeGeneration}` : 'No active generation'}</strong><span>{status.reportCount ?? 0} deployed components</span><span>{status.activatedAt ? `Activated ${formatDate(status.activatedAt)}` : 'Not activated'}</span><Tag minimal intent={hostReady ? 'success' : status.host?.status === 'unavailable' ? 'danger' : 'warning'}>{hostReady ? `Dynamic host ready · revision ${status.host.revision}` : status.host?.status === 'unavailable' ? 'Dynamic host unavailable' : 'Dynamic host status unknown'}</Tag></section>}
      {!skillsMode&&(status.diagnostics ?? []).map((diagnostic, index) => <Callout key={`${diagnostic.code}-${index}`} intent={diagnostic.severity === 'error' ? 'danger' : 'warning'} title={diagnostic.code || 'Runtime diagnostic'}>{diagnostic.message}</Callout>)}
      {skillsMode ? <SkillsCatalog api={api} skills={mcpSkills} tools={mcpTools} readers={readers} storageSkills={skills} loading={mcpLoading} error={skillsError} onOpen={onOpenComponent} onRefresh={refresh} onCreate={(suggestedOwner)=>{setSkillOwnerID(suggestedOwner||readers[0]?.reportId||'');setCreateSkillOpen(true);}}/> : <Tabs className="studio-runtime-tabs" id="runtime-tabs" selectedTabId={activeTab} onChange={setActiveTab} renderActiveTabPanelOnly>
        <Tab id="components" title={`Components (${readers.length})`} panel={<Components items={componentItems} query={queries.components} page={pages.components} onQuery={(value) => setQuery('components', value)} onPage={(value) => setPage('components', value)} onOpen={onOpenComponent}/>}/>
        <Tab id="tools" title={`MCP tools (${mcpTools.length})`} panel={<Tools items={toolItems} query={queries.tools} page={pages.tools} loading={mcpLoading} error={mcpError} onQuery={(value) => setQuery('tools', value)} onPage={(value) => setPage('tools', value)} onInspect={setSelectedTool}/>}/>
        <Tab id="resources" title={`Resources (${resources.length})`} panel={<Resources items={resourceItems} query={queries.resources} page={pages.resources} onQuery={(value) => setQuery('resources', value)} onPage={(value) => setPage('resources', value)}/>}/>
      </Tabs>}
    </>}
    <MCPToolSchemaDialog tool={selectedTool} onClose={() => setSelectedTool(null)}/>
    <Dialog isOpen={createSkillOpen} onClose={()=>setCreateSkillOpen(false)} title="New skill" icon="learning"><DialogBody><FormGroup label="Store with component" labelFor="new-skill-component"><HTMLSelect id="new-skill-component" fill value={skillOwnerID} onChange={(event)=>setSkillOwnerID(event.target.value)}>{readers.map((item)=><option key={item.reportId} value={item.reportId}>{item.title} · v{item.versionNo}</option>)}</HTMLSelect></FormGroup></DialogBody><DialogFooter actions={<><Button onClick={()=>setCreateSkillOpen(false)}>Cancel</Button><Button intent="primary" icon="add" disabled={!skillOwnerID&&readers.length===0} onClick={createSkill}>Continue</Button></>}/></Dialog>
  </main>;
}

function Components({ items, query, page, onQuery, onPage, onOpen }) { const current = pageOf(items, page); return <section className="studio-runtime-section" aria-labelledby="deployed-components-title"><Heading id="deployed-components-title" title="Deployed components" description="Serving version, connector, generation revision, and exposed-contract count." count={items.length} query={query} onQuery={onQuery} placeholder="Find component, namespace, connector, or revision"/><Card className="studio-card" elevation={0}>{current.items.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table studio-runtime-table" striped><thead><tr><th>Component</th><th>Version</th><th>Connector</th><th>Runtime revision</th><th>State</th><th>MCP</th><th>Actions</th></tr></thead><tbody>{current.items.map((reader) => <tr key={reader.reportId}><td><strong>{reader.title}</strong><small>{reader.namespace} · {reader.reportId}</small></td><td>v{reader.versionNo}</td><td><code>{reader.connectorName}</code></td><td><code>{reader.runtimeRevision || '—'}</code></td><td><Tag minimal intent={reader.status === 'active' ? 'success' : 'warning'}>{reader.status}</Tag></td><td>{reader.mcpExposures?.length ? `${reader.mcpExposures.length} exposed` : 'None'}</td><td><Button small minimal icon="application" aria-label={`Open ${reader.title}`} onClick={() => onOpen?.({ id: reader.reportId, title: reader.title, namespace: reader.namespace, defaultConnectorName: reader.connectorName })}>Open</Button></td></tr>)}</tbody></HTMLTable></div> : <Empty query={query} text="No deployed components are visible with your publication permissions."/>}<Pager {...current} onPage={onPage}/></Card></section>; }
function Tools({ items, query, page, loading, error, onQuery, onPage, onInspect }) { const current = pageOf(items, page); return <section className="studio-runtime-section" aria-labelledby="mcp-tools-title"><Heading id="mcp-tools-title" title="MCP tools" description={<>Live Datly <Code>tools/list</Code> contracts from the dedicated MCP server.</>} count={items.length} query={query} onQuery={onQuery} placeholder="Find MCP tool name or description"/>{error && <Callout intent="danger" role="alert" title="Datly MCP catalog unavailable">{error}</Callout>}<Card className="studio-card" elevation={0}>{loading ? <div className="studio-loading"><Spinner size={22}/></div> : current.items.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table studio-runtime-table" striped><thead><tr><th>Tool</th><th>Description</th><th>Input</th><th>Output</th><th>Actions</th></tr></thead><tbody>{current.items.map((tool) => <tr key={tool.name}><td><code>{tool.name}</code></td><td>{tool.description || '—'}</td><td>{schemaSummary(tool.inputSchema)}</td><td>{schemaSummary(tool.outputSchema)}</td><td><Button small minimal icon="search" aria-label={`Inspect ${tool.name} schema`} onClick={() => onInspect(tool)}>Inspect schema</Button></td></tr>)}</tbody></HTMLTable></div> : <Empty query={query} text="No MCP tools are active. Enable and publish a reader, cube, or cube-compose tool from Edit component."/>}<Pager {...current} onPage={onPage}/></Card></section>; }
function Resources({ items, query, page, onQuery, onPage }) { const current = pageOf(items, page); return <section className="studio-runtime-section" aria-labelledby="mcp-resources-title"><Heading id="mcp-resources-title" title="Resources" description="Published folder roots addressable through MCP resource URIs." count={items.length} query={query} onQuery={onQuery} placeholder="Find URI, namespace, folder, or component"/><Card className="studio-card" elevation={0}>{current.items.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table studio-runtime-table" striped><thead><tr><th>URI prefix</th><th>Component</th><th>Namespace</th><th>Folder</th></tr></thead><tbody>{current.items.map((resource) => <tr key={`${resource.reader.reportId}:${resource.uriPrefix}`}><td><code>{resource.uriPrefix}</code></td><td>{resource.reader.title}</td><td><code>{resource.namespace}</code></td><td><code>{resource.rootPath}</code></td></tr>)}</tbody></HTMLTable></div> : <Empty query={query} text="No MCP resource folders are published in this runtime generation."/>}<Pager {...current} onPage={onPage}/></Card></section>; }
function SkillsCatalog({ api, skills, tools: liveTools, readers, storageSkills, loading, error, onOpen, onRefresh, onCreate }) {
  const [query,setQuery]=useState('');
  const [page,setPage]=useState(1);
  const [selectedURI,setSelectedURI]=useState('');
  const [toolToAssign,setToolToAssign]=useState('');
  const [mutating,setMutating]=useState(false);
  const [mutationError,setMutationError]=useState('');
  const [pendingDelete,setPendingDelete]=useState(null);
  useEffect(()=>{if(!skills.some((item)=>item.uri===selectedURI))setSelectedURI(skills[0]?.uri??'');},[skills,selectedURI]);
  const filtered=search(skills,query,(skill)=>`${skill.frontmatter?.name??''} ${skill.frontmatter?.description??''} ${skill.frontmatter?.['allowed-tools']??''} ${skill.uri}`);
  const current=pageOf(filtered,page);
  const selected=skills.find((item)=>item.uri===selectedURI);
  const owner=selected&&storageSkills.find((item)=>selected.uri.startsWith(item.uriPrefix));
  const allowed=selected&&typeof selected.frontmatter?.['allowed-tools']==='string'?selected.frontmatter['allowed-tools'].split(/\s+/).filter(Boolean):[];
  const openOwner=(resources=false)=>owner&&onOpen?.({id:owner.reader.reportId,title:owner.reader.title,namespace:owner.reader.namespace,ownerPackage:owner.reader.ownerPackage,defaultConnectorName:owner.reader.connectorName,versionNo:owner.reader.versionNo,openResources:resources});
  const storedSkill=async()=>{
    if(!owner)throw new Error('The selected skill has no versioned Studio owner.');
    const snapshot=await api.getResources(owner.reader.reportId,owner.reader.versionNo);
    const root=snapshot.skills.find((item)=>item.skillId===owner.skillId);
    const folder=root&&snapshot.folders.find((item)=>item.folderId===root.folderId);
    if(!root||!folder)throw new Error('The selected skill storage contract is unavailable.');
    const relative=root.skillRoot==='.'?'':`${root.skillRoot}/`;
    const path=`${folder.rootPath}/${relative}SKILL.md`;
    const file=snapshot.files.find((item)=>item.namespace===folder.namespace&&item.resourcePath===path);
    if(!file)throw new Error('The selected SKILL.md file is unavailable.');
    return {snapshot,root,file};
  };
  const publish=async(next,reason)=>{
    const validation=await api.validateVersion(owner.reader.reportId,owner.reader.versionNo);
    if(!validation?.valid)throw new Error(validation?.diagnostics?.[0]?.message||'Skill publication validation failed.');
    await api.publishReader(owner.reader.reportId,owner.reader.versionNo,next.version.sourceRevision,reason);
    await onRefresh?.();
  };
  const replaceTools=async(nextTools)=>{
    setMutating(true);setMutationError('');
    try{const stored=await storedSkill();const next=await api.upsertResourceFile({reportId:owner.reader.reportId,versionNo:owner.reader.versionNo,expectedSourceRevision:stored.snapshot.version.sourceRevision,...stored.file,content:withSkillTools(stored.file.content,nextTools)});await publish(next,`Update allowed MCP tools for ${selected.frontmatter?.name}.`);setToolToAssign('');}
    catch(cause){setMutationError(cause.message);}finally{setMutating(false);}
  };
  const removeSkill=async()=>{
    setMutating(true);setMutationError('');
    try{const stored=await storedSkill();let next=await api.deleteSkillRoot(owner.reader.reportId,owner.reader.versionNo,stored.root.skillId,stored.snapshot.version.sourceRevision);next=await api.deleteResourceFile(owner.reader.reportId,owner.reader.versionNo,stored.file.resourceId,next.version.sourceRevision);await publish(next,`Remove skill ${selected.frontmatter?.name}.`);setPendingDelete(null);}
    catch(cause){setMutationError(cause.message);}finally{setMutating(false);}
  };
  const assignable=liveTools.filter((item)=>!allowed.includes(item.name));
  return <section className="studio-runtime-section" aria-labelledby="skills-catalog-title">
    <div className="studio-runtime-section-heading"><div><h2 id="skills-catalog-title">Skills</h2><p>Published through the live MCP skills catalog.</p></div><div className="studio-runtime-controls"><Tag minimal>{filtered.length} {filtered.length===1?'skill':'skills'}</Tag><InputGroup leftIcon="search" aria-label="Search skills" placeholder="Find skill or MCP tool" value={query} onChange={(event)=>{setQuery(event.target.value);setPage(1);}}/></div></div>
    <div className="studio-skill-catalog-toolbar" role="toolbar" aria-label="Skill actions"><ButtonGroup><Button icon="add" intent="primary" onClick={()=>onCreate(owner?.reader.reportId)}>New</Button><Button icon="edit" disabled={!owner||mutating} onClick={()=>openOwner(true)}>Edit</Button><Button icon="trash" intent="danger" disabled={!owner||mutating} onClick={()=>setPendingDelete(selected)}>Delete</Button></ButtonGroup><HTMLSelect aria-label="Assign published MCP tool" value={toolToAssign} onChange={(event)=>setToolToAssign(event.target.value)} disabled={!owner||mutating}><option value="">Select published MCP tool</option>{assignable.map((item)=><option key={item.name} value={item.name}>{item.name}</option>)}</HTMLSelect><Button icon="add" disabled={!toolToAssign||mutating} loading={mutating} onClick={()=>replaceTools([...allowed,toolToAssign])}>Assign</Button></div>
    {error && <Callout intent="danger" role="alert" title="Datly skills catalog unavailable"><p>{error}</p><Button small icon="refresh" intent="danger" onClick={onRefresh}>Retry catalog</Button></Callout>}{mutationError&&<Callout intent="danger" role="alert">{mutationError}</Callout>}
    {!error && <Card className="studio-card" elevation={0}>{loading ? <div className="studio-loading"><Spinner size={22}/></div> : current.items.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table studio-runtime-table studio-skills-table" striped><thead><tr><th aria-label="Select skill"/><th>Skill</th><th>Description</th><th>Allowed MCP tools</th><th>Resource URI</th></tr></thead><tbody>{current.items.map((skill) => { const portable=skill.frontmatter?.['allowed-tools']; const rowTools=typeof portable==='string'?portable.split(/\s+/).filter(Boolean):[]; return <tr key={skill.uri} className={selectedURI===skill.uri?'studio-selected-row':''} onClick={()=>setSelectedURI(skill.uri)}><td><input type="radio" aria-label={`Select ${skill.frontmatter?.name} skill`} checked={selectedURI===skill.uri} onChange={()=>setSelectedURI(skill.uri)}/></td><td><strong>{skill.frontmatter?.name}</strong></td><td>{skill.frontmatter?.description||'—'}</td><td><div className="studio-skill-tool-list">{rowTools.length ? rowTools.map((name)=>{const tool=liveTools.find((item)=>item.name===name);const provider=readers.find((reader)=>(reader.mcpExposures??[]).some((item)=>item.name===name));return tool&&provider?<span className="studio-skill-tool-assignment" key={name}><Button small minimal className="studio-skill-tool-link" aria-label={`Open MCP tool ${name} component`} onClick={(event)=>{event.stopPropagation();onOpen?.({id:provider.reportId,title:provider.title,namespace:provider.namespace,ownerPackage:provider.ownerPackage,defaultConnectorName:provider.connectorName,versionNo:provider.versionNo});}}>{name}</Button>{selectedURI===skill.uri&&<Button small minimal icon="small-cross" aria-label={`Remove ${name} from skill`} disabled={mutating} onClick={(event)=>{event.stopPropagation();replaceTools(rowTools.filter((item)=>item!==name));}}/>}</span>:<Tag key={name} minimal intent="danger">{name}</Tag>;}) : '—'}</div></td><td><code>{skill.uri}</code></td></tr>; })}</tbody></HTMLTable></div> : <Empty query={query} text="No skills are published yet."/>}<Pager {...current} onPage={setPage}/></Card>}
    <Alert isOpen={Boolean(pendingDelete)} intent="danger" icon="trash" confirmButtonText="Delete skill" cancelButtonText="Cancel" loading={mutating} onCancel={()=>setPendingDelete(null)} onConfirm={removeSkill}><p>Remove <strong>{pendingDelete?.frontmatter?.name}</strong> and its SKILL.md from the published catalog?</p></Alert>
  </section>;
}
function Heading({ id, title, description, count, query, onQuery, placeholder }) { return <div className="studio-runtime-section-heading"><div><h2 id={id}>{title}</h2><p>{description}</p></div><div className="studio-runtime-controls"><Tag minimal>{count} available</Tag><InputGroup leftIcon="search" aria-label={`Search ${title.toLowerCase()}`} placeholder={placeholder} value={query} onChange={(event) => onQuery(event.target.value)}/></div></div>; }
function Empty({ query, text }) { return <div className="studio-empty-compact">{query ? 'No items match this search.' : text}</div>; }
function Pager({ page, pages, onPage }) { if (pages < 2) return null; return <div className="studio-runtime-pager" aria-label="Pagination"><span>Page {page} of {pages}</span><Button small disabled={page === 1} onClick={() => onPage(page - 1)}>Previous</Button><Button small disabled={page === pages} onClick={() => onPage(page + 1)}>Next</Button></div>; }
function MCPToolSchemaDialog({ tool, onClose }) { return <Dialog className="studio-contract-dialog" isOpen={Boolean(tool)} onClose={onClose} title={tool ? `${tool.name} contract` : 'MCP tool contract'} icon="manual"><DialogBody><p className="studio-dialog-lead">Live contract returned by Datly’s dedicated MCP server through <Code>tools/list</Code>.</p><SchemaPanel title="Input" schema={tool?.inputSchema}/><SchemaPanel title="Output" schema={tool?.outputSchema} empty="Datly did not declare an output schema for this tool."/></DialogBody><DialogFooter actions={<Button onClick={onClose}>Close</Button>}/></Dialog>; }
function SchemaPanel({ title, schema, empty }) { const properties = schema?.properties ?? {}; const required = new Set(schema?.required ?? []); return <section className="studio-contract-panel"><div className="studio-contract-panel-heading"><h3>{title}</h3>{schema && <Tag minimal>{Object.keys(properties).length} fields</Tag>}</div>{!schema ? <p className="studio-empty-compact">{empty}</p> : <>{Object.keys(properties).length ? <div className="studio-contract-fields">{Object.entries(properties).map(([name, property]) => <div className="studio-contract-field" key={name}><div><Code>{name}</Code>{required.has(name) && <Tag minimal intent="primary">required</Tag>}</div><span>{schemaType(property)}</span>{property?.description && <p>{property.description}</p>}</div>)}</div> : <p className="studio-empty-compact">No named fields.</p>}<details className="studio-contract-raw"><summary>Raw JSON schema</summary><pre className="studio-preview-source">{JSON.stringify(schema, null, 2)}</pre></details></>}</section>; }
function schemaType(property) { const value = property?.type; if (Array.isArray(value)) return value.join(' | '); if (value) return value; if (property?.items) return `array<${schemaType(property.items)}>`; return 'object'; }
function search(items, query, stringify) { const value = query.trim().toLowerCase(); return value ? items.filter((item) => stringify(item).toLowerCase().includes(value)) : items; }
function pageOf(items, requested) { const pages = Math.max(1, Math.ceil(items.length / pageSize)); const page = Math.min(requested, pages); return { items: items.slice((page - 1) * pageSize, page * pageSize), page, pages }; }
function schemaSummary(schema) { const properties = schema?.properties; return properties ? `${Object.keys(properties).length} fields` : schema ? 'defined' : 'not declared'; }
function formatDate(value) { const date = new Date(value); if (Number.isNaN(date.valueOf())) return value; return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(date); }

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, ButtonGroup, Callout, Card, Code, Divider, InputGroup, Menu, MenuItem, Popover, PopoverInteractionKind, Spinner, Tag } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';
import { ReaderParameterDialog } from './ReaderParameterDialog.jsx';
import { ReaderPredicateDialog } from './ReaderPredicateDialog.jsx';
import { DownloadComponentButton } from './ComponentTransfer.jsx';
import { ReaderSubviewDialog } from './ReaderSubviewDialog.jsx';
import { ReaderRemoveViewDialog } from './ReaderRemoveViewDialog.jsx';
import { ReaderAnalyticsDialog } from './ReaderAnalyticsDialog.jsx';
import { ReaderCacheDialog } from './ReaderCacheDialog.jsx';
import { ResultPreview } from './NestedDataGrid.jsx';
import { ReaderRelationDialog } from './ReaderRelationDialog.jsx';
import { ReaderFieldDialog } from './ReaderFieldDialog.jsx';
import { ReaderExposureDialog } from './ReaderExposureDialog.jsx';
import { ReaderValidationDialog } from './ReaderValidationDialog.jsx';
import { ReaderPublicationDialog } from './ReaderPublicationDialog.jsx';
import { ReaderResourcesDialog } from './ReaderResourcesDialog.jsx';
import { ReaderACLDialog } from './ReaderACLDialog.jsx';
import { ReaderConflictDialog } from './ReaderConflictDialog.jsx';

// ReaderBuilder renders the canonical structure returned by the Studio SDK's
// Datly reader-builder bridge. It does not parse DQL or invent a browser graph.
export function ReaderBuilder({ api, report, openResources = false, resourceAction = '', onReportUpdated, onBack }) {
  const headingRef = useRef(null);
  const [inspection, setInspection] = useState(null);
  const [version, setVersion] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [parameterDialogOpen, setParameterDialogOpen] = useState(false);
  const [predicateDialogOpen, setPredicateDialogOpen] = useState(false);
  const [subviewDialogOpen, setSubviewDialogOpen] = useState(false);
  const [removeViewDialogOpen, setRemoveViewDialogOpen] = useState(false);
  const [analyticsDialogOpen, setAnalyticsDialogOpen] = useState(false);
  const [cacheDialogOpen, setCacheDialogOpen] = useState(false);
  const [exposureDialogOpen, setExposureDialogOpen] = useState(false);
  const [validationDialogOpen, setValidationDialogOpen] = useState(false);
  const [publicationDialogOpen, setPublicationDialogOpen] = useState(false);
  const [resourcesDialogOpen, setResourcesDialogOpen] = useState(false);
  useEffect(() => { if (openResources) setResourcesDialogOpen(true); }, [openResources, report?.id]);
  const [aclDialogOpen, setACLDialogOpen] = useState(false);
  const [conflict, setConflict] = useState(null);
  const [conflictReloading, setConflictReloading] = useState(false);
  const [preview, setPreview] = useState(null);
  const [previewTarget, setPreviewTarget] = useState(null);
  const [previewing, setPreviewing] = useState(false);
  const [testedView, setTestedView] = useState(null);
  const [testedRelation, setTestedRelation] = useState(null);
  const [testingView, setTestingView] = useState('');
  const [testingRelation, setTestingRelation] = useState('');
  const [selectedNode, setSelectedNode] = useState(null);
  const [editRelation, setEditRelation] = useState(null);
  const [editFields, setEditFields] = useState(null);
  const [activeTab, setActiveTab] = useState('component');
  const [viewTabs, setViewTabs] = useState([]);
  const [showRawDQL, setShowRawDQL] = useState(false);
  const [dirtyTabs,setDirtyTabs]=useState(()=>new Set());
  const [pendingTabAction,setPendingTabAction]=useState(null);

  const load = async () => {
    if (!report) return;
    setLoading(true); setError('');
    try {
      if (report.versionNo) {
        const selected = await api.inspectVersion(report.id, report.versionNo);
        setVersion(selected.version);
        setInspection(selected);
        return;
      }
      const page = await api.listVersions(report.id, { limit: 1 });
      const current = page?.items?.[0];
      if (!current) { setVersion(null); setInspection(null); return; }
      setVersion(current);
      setInspection(await api.inspectVersion(report.id,current.versionNo));
    } catch (cause) { setError(cause.message); }
    finally { setLoading(false); }
  };
  useEffect(() => { load(); }, [api, report?.id, report?.versionNo]);

  const rootView = inspection?.structure?.component?.rootView;
  const diagnostics = inspection?.diagnostics ?? [];
  const source = inspection?.dql ?? '';
  const capabilities = inspection?.capabilities ?? {};
  const canEdit = capabilities.canEdit !== false;
  const canRun = capabilities.canRun !== false;
  const canPublish = capabilities.canPublish !== false;
  const canUseDQL = capabilities.canUseDql === true;
  const applyCommand = async (operation) => {
    if (!inspection?.version) throw new Error('Reader inspection is unavailable.');
    let result;
    try {
      result = await api.applyReaderCommand(report.id, inspection.version.versionNo, {
        expectedSourceRevision: inspection.version.sourceRevision,
        operation,
      });
    } catch (cause) {
      if (cause?.code === 'conflict') setConflict(cause);
      throw cause;
    }
    if (!result?.applied) {
      const message = result?.inspection?.diagnostics?.[0]?.message || 'Datly rejected the reader-builder command.';
      throw new Error(message);
    }
    setInspection(result.inspection);
    setVersion(result.inspection.version);
    setTestedRelation(null);
    return result;
  };
  const runPreview = async (target = null) => {
    if (!inspection?.version) return;
    setPreviewing(true); setError(''); setPreviewTarget(target);
    try { setPreview(await api.previewReader(report.id, inspection.version.versionNo, {}, 50, inspection?.structure?.component?.routes?.[0]?.path)); }
    catch (cause) { setError(cause.message); }
    finally { setPreviewing(false); }
  };
  const viewNames = useMemo(() => collectViews(inspection?.structure), [inspection]);
  const runViewTest = async (view) => {
    if (!inspection?.version || !view) return;
    setTestingView(view); setError('');
    try { setTestedView(await api.testReaderView(report.id, inspection.version.versionNo, view)); }
    catch (cause) { setError(cause.message); }
    finally { setTestingView(''); }
  };
  const runWarmup = async () => {
    if (!inspection?.version) throw new Error('Reader inspection is unavailable.');
    setError('');
    return api.warmupReader(report.id, inspection.version.versionNo);
  };
  const runCompose = async (input) => {
    if (!inspection?.version) throw new Error('Reader inspection is unavailable.');
    return api.testCubeCompose(report.id, inspection.version.versionNo, input);
  };
  const validateDraft = async () => {
    if (!inspection?.version) throw new Error('Reader inspection is unavailable.');
    const result = await api.validateVersion(report.id, inspection.version.versionNo);
    setVersion(result.version);
    setInspection((current) => current ? { ...current, version: result.version } : current);
    return result;
  };
  const publishDraft = async (reason) => {
    if (!inspection?.version) throw new Error('Reader inspection is unavailable.');
    let result;
    try { result = await api.publishReader(report.id, inspection.version.versionNo, inspection.version.sourceRevision, reason); }
    catch (cause) { if (cause?.code === 'conflict') setConflict(cause); throw cause; }
    await load();
    return result;
  };
  const unpublishDraft = async (reason) => {
    if (!report) throw new Error('Reader report is unavailable.');
    const result = await api.unpublishReader(report.id, reason);
    await load();
    return result;
  };
  const rollbackDraft = async (historicalVersion, reason) => {
    if (!report || !historicalVersion) throw new Error('Historical version is unavailable.');
    let result;
    try { result = await api.rollbackReader(report.id, historicalVersion.versionNo, historicalVersion.sourceRevision, reason); }
    catch (cause) { if (cause?.code === 'conflict') setConflict(cause); throw cause; }
    await load();
    return result;
  };
  const resourceChanged = (updated) => {
    if (!updated) return;
    setVersion(updated);
    setInspection((current) => current ? { ...current, version: updated } : current);
  };
  const openSQLTab = (node) => {
    const target = selectedViewNode(node);
    if (!target) return;
    setSelectedNode(node);
    setViewTabs((current) => current.some((item) => item.name === target.name) ? current : [...current, { ...target, mode: 'sql' }]);
    setActiveTab(`sql:${target.name}`);
  };
  const closeViewTabNow = (name, mode) => {
    setViewTabs((current) => current.filter((item) => item.name !== name || item.mode !== mode));
    setDirtyTabs((current)=>{const next=new Set(current);next.delete(`${mode}:${name}`);return next;});
    if (activeTab === `${mode}:${name}`) setActiveTab('component');
  };
  const closeViewTab=(name,mode)=>{const key=`${mode}:${name}`;if(dirtyTabs.has(key)){setPendingTabAction({kind:'close',name,mode});return;}closeViewTabNow(name,mode);};
  const selectBuilderTab=(next)=>{if(next===activeTab)return;if(dirtyTabs.has(activeTab)){setPendingTabAction({kind:'select',next});return;}setActiveTab(next);};
  const discardPendingTab=()=>{const action=pendingTabAction;if(!action)return;if(action.kind==='close')closeViewTabNow(action.name,action.mode);else if(action.kind==='back'){setDirtyTabs(new Set());onBack();}else{setDirtyTabs((current)=>{const next=new Set(current);next.delete(activeTab);return next;});setActiveTab(action.next);}setPendingTabAction(null);};
  const requestBack=()=>{if(dirtyTabs.has(activeTab)){setPendingTabAction({kind:'back'});return;}onBack();};
  const reloadAfterConflict = async () => {
    setConflictReloading(true);
    setParameterDialogOpen(false); setPredicateDialogOpen(false); setSubviewDialogOpen(false); setRemoveViewDialogOpen(false);
    setEditRelation(null); setEditFields(null); setAnalyticsDialogOpen(false); setCacheDialogOpen(false);
    setExposureDialogOpen(false); setValidationDialogOpen(false); setPublicationDialogOpen(false); setResourcesDialogOpen(false);
    try { await load(); setConflict(null); requestAnimationFrame(()=>headingRef.current?.focus()); }
    finally { setConflictReloading(false); }
  };
  const selectView = (node) => setSelectedNode(node);
  const testSelectedView = (node) => {
    const target = selectedViewNode(node);
    if (!target) return;
    runViewTest(target.name);
  };
  const testSelectedRelation = async (node) => {
    if (!inspection?.version || node?.type !== 'relation') return;
    const identity = `${node.parentName}->${node.child?.name || node.relation?.name}`;
    setTestingRelation(node.key); setError('');
    try { setTestedRelation(await api.testReaderRelation(report.id, inspection.version.versionNo, identity, {}, 50)); }
    catch (cause) { setError(cause.message); }
    finally { setTestingRelation(''); }
  };
  const removeSelectedView = (node) => {
    const target = selectedViewNode(node);
    if (!target) return;
    setSelectedNode(target);
    setRemoveViewDialogOpen(true);
  };
  const activeViewName = activeTab.startsWith('sql:') ? activeTab.slice(activeTab.indexOf(':') + 1) : '';
  const activeView = activeViewName ? findView(rootView, activeViewName) : null;

  return <main className="studio-workspace studio-builder-workspace">
    <div className="studio-builder-header">
      {version&&<DownloadComponentButton api={api} report={report} versionNo={version.versionNo} disabled={!canUseDQL}/>}
      <div><Button minimal icon="arrow-left" onClick={requestBack}>Components</Button><h1 ref={headingRef} tabIndex={-1} className="studio-page-heading">{report?.title ?? 'Reader Builder'}</h1>
        <p className="studio-page-description">Versioned Datly reader graph for {report?.defaultConnectorName}.</p></div>
      {version && <div className="studio-builder-header-actions"><div className="studio-builder-status" aria-label="Component revision status"><Tag minimal>{report?.namespace ?? 'general'}</Tag><Tag minimal>{version.state==='published'?'Published':'Draft'} v{version.versionNo}</Tag><Tag minimal>rev {version.sourceRevision}</Tag><Tag intent={version.compileStatus === 'valid' ? 'success' : version.compileStatus === 'invalid' ? 'danger' : 'warning'} minimal>{version.compileStatus}</Tag></div><Button icon="edit" disabled={!canEdit} onClick={()=>setExposureDialogOpen(true)}>Edit component</Button></div>}
    </div>
    {version && <div className="studio-builder-commandbar" role="toolbar" aria-label="Component authoring commands">
      <div className="studio-command-group"><span>Author</span><ButtonGroup minimal><Button small icon="add" disabled={!canEdit} onClick={() => setParameterDialogOpen(true)}>Inputs</Button><Button small icon="filter" disabled={!canEdit} onClick={() => setPredicateDialogOpen(true)}>Predicates</Button><Button small icon="heatmap" disabled={!canRun} onClick={() => setAnalyticsDialogOpen(true)}>Composition lab</Button><Button small icon="database" disabled={!canEdit} onClick={() => setCacheDialogOpen(true)}>Cache & warmup</Button></ButtonGroup></div>
      <div className="studio-command-group"><span>Govern</span><ButtonGroup minimal><Button small icon="folder-open" disabled={!canEdit} onClick={()=>setResourcesDialogOpen(true)}>Skills & resources</Button><Button small icon="lock" disabled={!canPublish} onClick={()=>setACLDialogOpen(true)}>Permissions</Button></ButtonGroup></div>
      <div className="studio-command-group studio-command-release"><span>Release</span><ButtonGroup minimal><Button small icon="endorsed" disabled={!canEdit} onClick={()=>setValidationDialogOpen(true)}>Validate</Button><Button small icon="play" disabled={!canRun} loading={previewing&&!previewTarget} onClick={()=>runPreview(null)}>Preview</Button><Button small intent="primary" icon="rocket-slant" disabled={!canPublish || version.compileStatus !== 'valid'} title={!canPublish ? 'Publish permission is required' : version.compileStatus === 'valid' ? 'Publish this validated revision' : 'Validate this exact revision before publishing'} onClick={()=>setPublicationDialogOpen(true)}>Publish</Button><Button small icon="refresh" aria-label="Refresh component" title="Refresh component" onClick={load}/></ButtonGroup></div>
    </div>}
    {error && <Callout intent="danger" title="Reader Builder request failed" role="alert">{error}<Button small minimal intent="danger" onClick={load}>Retry</Button></Callout>}
    {loading && <div className="studio-loading"><Spinner size={28} /></div>}
    {!loading && !version && <Card className="studio-card studio-empty" elevation={0}><h2>No reader definition yet</h2><span>Create an initial reader version from this catalog entry before composing views.</span></Card>}
    {!loading && version && <><BuilderTabs componentTitle={report?.title} activeTab={activeTab} viewTabs={viewTabs} canUseDQL={canUseDQL} onSelect={selectBuilderTab} onClose={closeViewTab}/><div className="studio-builder-grid" role="tabpanel" id={workspacePanelId(activeTab)} aria-labelledby={workspaceTabId(activeTab)} tabIndex={0}>
      {activeTab === 'component' ? <>
        <Card className="studio-card studio-graph-panel" elevation={0}>
          <div className="studio-panel-heading"><div><h2>Component graph</h2><p className="studio-muted">Open a view to inspect its fields. Select a relation to inspect its join.</p></div><Code>{rootView ? `${displayViewName(rootView)} tree` : 'empty'}</Code></div>
          {rootView && <ComponentGraph root={rootView} selected={selectedNode} canEdit={canEdit} canRun={canRun} onSelect={selectView} onOpen={selectView} onOpenSQL={openSQLTab} onAdd={(node)=>{setSelectedNode(node);setSubviewDialogOpen(true);}} onEdit={selectView} onTest={testSelectedView} onRemove={removeSelectedView} />}
        </Card>
        <SelectionWorkspace api={api} selection={selectedNode} root={rootView} connector={report?.defaultConnectorName} functions={inspection?.structure?.functions ?? []} version={version} relationTest={testedRelation} canEdit={canEdit} canRun={canRun} testingView={testingView} testingRelation={testingRelation} onClear={() => setSelectedNode(null)} onOpenSQL={openSQLTab} onAdd={(node) => { setSelectedNode(node); setSubviewDialogOpen(true); }} onEditRelation={setEditRelation} onFields={(node) => setEditFields(node)} onTest={testSelectedView} onRunRelation={testSelectedRelation} onPreviewRelation={(node) => runPreview({type:'relation',label:`${displayNodeName(node.parentName, rootView)} → ${displayViewName(node.child?.view)}`})} onRemove={removeSelectedView}/>
      </> : activeView ? <ViewWorkspace mode="sql" view={activeView} viewName={activeViewName} root={rootView} sql={viewSourceSQL(inspection, activeViewName)} canEdit={canEdit} canRun={canRun} onDirtyChange={(dirty)=>setDirtyTabs((current)=>{const next=new Set(current);if(dirty)next.add(activeTab);else next.delete(activeTab);return next;})} onApply={applyCommand} onTest={() => runViewTest(activeViewName)} testing={testingView === activeViewName}/> : null}
      {activeTab === 'advanced' && canUseDQL && <Card className="studio-card studio-advanced-panel" elevation={0}><div className="studio-panel-heading"><div><h2>Advanced component source</h2><p className="studio-muted">Component DQL is structural. View SQL opens in a dedicated SQL tab.</p></div><Button small minimal icon={showRawDQL ? 'eye-off' : 'eye-open'} onClick={() => setShowRawDQL((current) => !current)}>{showRawDQL ? 'Hide raw DQL' : 'Show raw DQL'}</Button></div>{showRawDQL ? <pre className="studio-dql-source">{source || 'No DQL source is defined.'}</pre> : <><div className="studio-dql-resource-list">{collectViewEntries(rootView).map((entry) => <Button key={entry.view.name} minimal icon="code" onClick={() => openSQLTab({ type: 'view', name: entry.view.name, label: entry.view.name, view: entry.view })}>{`embed:sql/${viewResourcePath(entry.view, rootView)}`}</Button>)}</div><Divider/><h3>Diagnostics</h3>{diagnostics.length === 0 ? <div className="studio-diagnostic-ok">Datly inspection completed without diagnostics.</div> : diagnostics.map((item, index) => <Callout key={`${item.code}-${index}`} intent={item.severity === 'error' ? 'danger' : 'warning'} title={item.code || item.severity}>{item.message}</Callout>)}</>}</Card>}
      {preview && <Card className="studio-card studio-preview-panel" elevation={0}>
        <div className="studio-panel-heading"><div><h2>{previewTarget?.type==='relation'?`Graph preview · ${previewTarget.label}`:'Reader preview'}</h2><p className="studio-muted">{previewTarget?.type==='relation'?'Full exact-version graph preview with the selected relation in context; relation-specific assertions are reported only by a dedicated relation test.':'Exact draft-version execution through the authorized Studio SDK.'}</p></div><Code>{formatDuration(preview.duration)}</Code></div>
        <ExecutionEvidence evidence={preview.evidence}/>
        <ResultPreview data={preview.data}/>
      </Card>}
      {testedView && <Card className="studio-card studio-preview-panel" elevation={0}>
        <div className="studio-panel-heading"><div><h2>View test · {testedView.view}</h2><p className="studio-muted">A dynamically generated Datly reader for this view only.</p></div><Code>{formatDuration(testedView.duration)}</Code></div>
        <ExecutionEvidence evidence={testedView.evidence}/>
        <ResultPreview data={testedView.data}/>
      </Card>}
    </div></>}
    <ReaderParameterDialog isOpen={parameterDialogOpen} structure={inspection?.structure} onClose={() => setParameterDialogOpen(false)} onApply={applyCommand} />
    <ReaderPredicateDialog api={api} isOpen={predicateDialogOpen} structure={inspection?.structure} onClose={() => setPredicateDialogOpen(false)} onApply={applyCommand} />
    <ReaderSubviewDialog isOpen={subviewDialogOpen} structure={inspection?.structure} initialParent={selectedNode?.type==='view'?selectedNode.name:''} onClose={() => setSubviewDialogOpen(false)} onApply={applyCommand} />
    <ReaderRemoveViewDialog isOpen={removeViewDialogOpen} structure={inspection?.structure} initialView={selectedNode?.type==='view'?selectedNode.name:''} onClose={() => setRemoveViewDialogOpen(false)} onApply={applyCommand} />
    <ReaderRelationDialog isOpen={Boolean(editRelation)} node={editRelation} structure={inspection?.structure} onClose={()=>setEditRelation(null)} onApply={async(operation)=>{await applyCommand(operation);setSelectedNode(null);}}/>
    <ReaderFieldDialog isOpen={Boolean(editFields)} node={editFields} columnContracts={inspection?.structure?.columnContracts ?? []} onClose={()=>setEditFields(null)} onApply={async(operation)=>{await applyCommand(operation);setSelectedNode(null);}}/>
    <ReaderAnalyticsDialog isOpen={analyticsDialogOpen} structure={inspection?.structure} onClose={() => setAnalyticsDialogOpen(false)} onTestCompose={runCompose} />
    <ReaderCacheDialog api={api} report={report} version={version} canWarmup={canPublish} isOpen={cacheDialogOpen} structure={inspection?.structure} onClose={() => setCacheDialogOpen(false)} onApply={applyCommand} onWarmup={runWarmup} />
    <ReaderExposureDialog isOpen={exposureDialogOpen} api={api} structure={inspection?.structure} version={version} report={report} onReportUpdated={onReportUpdated} onClose={()=>setExposureDialogOpen(false)} onApply={applyCommand}/>
    <ReaderValidationDialog isOpen={validationDialogOpen} version={version} canUseDQL={canUseDQL} onClose={()=>setValidationDialogOpen(false)} onValidate={validateDraft} onOpenSource={()=>{setValidationDialogOpen(false);setActiveTab('advanced');setShowRawDQL(true);}}/>
    <ReaderPublicationDialog isOpen={publicationDialogOpen} api={api} report={report} version={version} inspection={inspection} onClose={()=>setPublicationDialogOpen(false)} onPublish={publishDraft} onUnpublish={unpublishDraft} onRollback={rollbackDraft}/>
    <ReaderResourcesDialog isOpen={resourcesDialogOpen} initialAction={resourceAction} api={api} report={report} version={version} onClose={()=>setResourcesDialogOpen(false)} onChanged={resourceChanged}/>
    <ReaderACLDialog isOpen={aclDialogOpen} api={api} report={report} onClose={()=>setACLDialogOpen(false)}/>
    <ReaderConflictDialog conflict={conflict} reloading={conflictReloading} onReview={()=>setConflict(null)} onReload={reloadAfterConflict}/>
    <Alert isOpen={Boolean(pendingTabAction)} intent="warning" icon="warning-sign" confirmButtonText="Discard changes" cancelButtonText="Keep editing" onCancel={()=>setPendingTabAction(null)} onConfirm={discardPendingTab} canEscapeKeyCancel canOutsideClickCancel><p>This SQL tab has unsaved changes. Leaving it now discards the draft; the saved Datly view remains unchanged.</p></Alert>
  </main>;
}

function BuilderTabs({ componentTitle, activeTab, viewTabs, canUseDQL, onSelect, onClose }) {
  const tabs = ['component', ...viewTabs.map((view) => `${view.mode}:${view.name}`), ...(canUseDQL ? ['advanced'] : [])];
  const onKeyDown = (event, current) => {
    const index = tabs.indexOf(current);
    let next = -1;
    if (event.key === 'ArrowRight') next = (index + 1) % tabs.length;
    if (event.key === 'ArrowLeft') next = (index - 1 + tabs.length) % tabs.length;
    if (event.key === 'Home') next = 0;
    if (event.key === 'End') next = tabs.length - 1;
    if (next < 0) return;
    event.preventDefault();
    onSelect(tabs[next]);
    requestAnimationFrame(() => document.getElementById(workspaceTabId(tabs[next]))?.focus());
  };
  return <div className="studio-builder-tabs" role="tablist" aria-label="Component workspace tabs">
    <Button small icon="application" role="tab" id={workspaceTabId('component')} aria-controls={workspacePanelId('component')} tabIndex={activeTab === 'component' ? 0 : -1} active={activeTab === 'component'} aria-selected={activeTab === 'component'} onKeyDown={(event)=>onKeyDown(event,'component')} onClick={() => onSelect('component')}>{`Component ${componentTitle || ''}`}</Button>
    {viewTabs.map((view) => { const key=`${view.mode}:${view.name}`; return <div role="presentation" className={`studio-builder-tab-view ${activeTab === key ? 'active' : ''}`} key={key}><Button small role="tab" id={workspaceTabId(key)} aria-controls={workspacePanelId(key)} tabIndex={activeTab === key ? 0 : -1} icon={view.mode === 'sql' ? 'code' : 'properties'} active={activeTab === key} aria-selected={activeTab === key} onKeyDown={(event)=>onKeyDown(event,key)} onClick={() => onSelect(key)}>{view.mode === 'sql' ? `${displayViewName(view.view || view)} SQL` : displayViewName(view.view || view)}</Button><Button small minimal icon="cross" aria-label={`Close ${displayViewName(view.view || view)} ${view.mode === 'sql' ? 'SQL' : 'view'} tab`} onClick={() => onClose(view.name, view.mode)}/></div>;})}
    {canUseDQL && <Button small minimal icon="cog" role="tab" id={workspaceTabId('advanced')} aria-controls={workspacePanelId('advanced')} tabIndex={activeTab === 'advanced' ? 0 : -1} active={activeTab === 'advanced'} aria-selected={activeTab === 'advanced'} aria-label="Advanced component DQL" title="Advanced component DQL" onKeyDown={(event)=>onKeyDown(event,'advanced')} onClick={() => onSelect('advanced')}/>}
  </div>;
}

function workspaceToken(value) { return String(value || 'component').replace(/[^A-Za-z0-9_-]+/g, '-'); }
function workspaceTabId(value) { return `component-workspace-tab-${workspaceToken(value)}`; }
function workspacePanelId(value) { return `component-workspace-panel-${workspaceToken(value)}`; }

function ViewWorkspace({ view, viewName, root, sql, canEdit, canRun, onDirtyChange, onApply, onTest, testing }) {
  const [draft, setDraft] = useState(sql || '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [editorSize, setEditorSize] = useState('balanced');
  useEffect(() => { setDraft(sql || ''); setError(''); setSaving(false); setEditorSize('balanced'); onDirtyChange?.(false); }, [view?.name, sql]);
  const lineage = viewLineage(root, view?.name).map(displayViewName).join(' / ');
  const save = async () => { if (!draft.trim()) { setError('View SQL is required.'); return; } setSaving(true); setError(''); try { await onApply({ type: 'updateView', view: { name: viewName || view.name, sql: draft.trim() } }); } catch (cause) { setError(cause.message); } finally { setSaving(false); } };
  const dirty = draft !== (sql || '');
  useEffect(()=>()=>onDirtyChange?.(false),[view?.name]);
  useEffect(()=>{const warn=(event)=>{if(!dirty)return;event.preventDefault();event.returnValue='';};window.addEventListener('beforeunload',warn);return()=>window.removeEventListener('beforeunload',warn);},[dirty]);
  const relation=findRelationContext(root,view?.name);
  return <Card className="studio-card studio-view-workspace" elevation={0}><div className="studio-sql-header"><div><h2><span className="studio-view-kind-icon">◇</span>{displayViewName(view)} SQL</h2><p className="studio-muted">{lineage || displayViewName(view)} · <code>{`embed:sql/${viewResourcePath(view, root)}`}</code></p></div><div className="studio-sql-actions"><ButtonGroup minimal aria-label="SQL workspace size"><Button small icon="minimize" aria-label="Minimize SQL editor" aria-pressed={editorSize==='compact'} onClick={()=>setEditorSize('compact')}/><Button small icon="vertical-distribution" aria-label="Balance SQL editor and preview" aria-pressed={editorSize==='balanced'} onClick={()=>setEditorSize('balanced')}/><Button small icon="maximize" aria-label="Maximize SQL editor" aria-pressed={editorSize==='expanded'} onClick={()=>setEditorSize('expanded')}/></ButtonGroup><Tag minimal intent={dirty ? 'warning' : 'success'}>{dirty ? 'Unsaved changes' : 'Saved'}</Tag><Button icon="play" disabled={!canRun} loading={testing} onClick={onTest}>Test view</Button><Button intent="primary" icon="floppy-disk" disabled={!canEdit || !dirty} loading={saving} onClick={save}>Save SQL</Button></div></div>{relation&&<div className="studio-sql-relation-keys"><span>Join keys</span>{(relation.relation.on??[]).map((item,index)=><code key={`${item.parentColumn}-${item.childColumn}-${index}`}>{item.parentColumn} → {item.childColumn}</code>)}</div>}{error && <Callout intent="danger" role="alert">{error}</Callout>}<div className={`studio-code-editor studio-code-editor-resizable ${editorSize}`} aria-label="Resizable SQL pane"><Editor ariaLabel={`${displayViewName(view)} SQL source`} value={draft} onChange={(value)=>{setDraft(value);onDirtyChange?.(value!==(sql||''));}} language="sql" height="100%" readOnly={saving || !canEdit}/></div></Card>;
}

function SelectionWorkspace({ api, selection, root, connector, functions, version, relationTest, canEdit, canRun, testingView, testingRelation, onClear, onOpenSQL, onAdd, onEditRelation, onFields, onTest, onRunRelation, onPreviewRelation, onRemove }) {
  const target = selectedViewNode(selection);
  const view = target?.view;
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [discoveredColumns, setDiscoveredColumns] = useState([]);
  const [metadataError, setMetadataError] = useState('');
  const [metadataLoading, setMetadataLoading] = useState(false);
  const [metadataReload, setMetadataReload] = useState(0);
  useEffect(() => { setQuery(''); setPage(0); }, [target?.name]);
  useEffect(() => {
    let cancelled = false;
    if (!view || !api || !connector) { setDiscoveredColumns([]); setMetadataError(''); setMetadataLoading(false); return () => { cancelled = true; }; }
    const reference = tableReference(view?.source?.table);
    if (!reference.table) { setDiscoveredColumns([]); setMetadataError(''); setMetadataLoading(false); return () => { cancelled = true; }; }
    setMetadataError(''); setDiscoveredColumns([]); setMetadataLoading(true);
    api.getTable(connector, reference).then((detail) => {
      if (!cancelled) setDiscoveredColumns((detail?.columns ?? []).map((column) => ({ name: column.name, source: column.name, databaseType: column.type, type: { name: column.type }, groupable: false, schema: column })));
    }).catch((cause) => { if (!cancelled) setMetadataError(cause.message); }).finally(() => { if (!cancelled) setMetadataLoading(false); });
    return () => { cancelled = true; };
  }, [api, connector, view?.name, view?.source?.table, view?.columns, metadataReload]);
  if (!target || !view) return <Card className="studio-card studio-view-inspector studio-view-inspector-empty" elevation={0}><span>Select a view or relation to inspect its settings and columns.</span></Card>;
  const resolvedView = { ...view, columns: mergeViewColumns(view?.columns, discoveredColumns) };
  const columns = (resolvedView.columns ?? []).filter((column) => `${column.name} ${column.source || ''} ${column.type?.name || column.databaseType || ''}`.toLowerCase().includes(query.trim().toLowerCase()));
  const pageSize = 20;
  const pages = Math.max(1, Math.ceil(columns.length / pageSize));
  const currentPage = Math.min(page, pages - 1);
  const visibleColumns = columns.slice(currentPage * pageSize, currentPage * pageSize + pageSize);
  const relation = selection?.type === 'relation';
  const derived = selection?.relationKind === 'derived';
  const relationCurrent = relation && relationTest && normalizedIdentity(relationTest.parentView) === normalizedIdentity(selection.parentName) && normalizedIdentity(relationTest.childView) === normalizedIdentity(target.name) && relationTest.evidence?.versionNo === version?.versionNo && relationTest.evidence?.sourceRevision === version?.sourceRevision;
  const parentLabel = relation ? displayNodeName(selection.parentName, root) : '';
  const viewFunctions = (functions ?? []).filter((fn) => String(fn.args?.[0] || '').split('.')[0] === view?.name && !['tag', 'cast'].includes(String(fn.name || '').toLowerCase()));
  const sourceLabel = view?.source?.table || `embed:sql/${viewResourcePath(view, root)}`;
  return <>
    <Card className="studio-card studio-view-inspector" elevation={0}>
      <div className="studio-inspector-heading"><div><span>{relation ? 'Relation' : target.root ? 'Root view' : 'View settings'}</span><h2>{relation ? `${parentLabel} → ${displayViewName(view)}` : displayViewName(view)}</h2></div><Button minimal small icon="cross" aria-label="Clear graph selection" onClick={onClear}/></div>
      {relation&&<div className="studio-relation-classification"><Tag minimal intent="primary">{derived?'derived output':String(selection.relationKind||'subview')}</Tag><Tag minimal>{String(selection.relation?.cardinality||'many')}</Tag></div>}
      <dl className="studio-view-settings"><div><dt>Source</dt><dd><code>{sourceLabel}</code></dd></div><div><dt>Connector</dt><dd>{connector || 'Inherited'}</dd></div><div><dt>Contract</dt><dd><code>{view.name || view.namespace}</code></dd></div>{relation && <div><dt>{derived ? 'Output' : 'Keys'}</dt><dd>{derived?<code>Independent output; no parent attachment</code>:<div className="studio-relation-keys">{(selection.relation?.on??[]).map((key,index)=><code key={`${key.parentColumn}-${key.childColumn}-${index}`}>{key.parentColumn} → {key.childColumn}</code>)}</div>}</dd></div>}</dl>
      {viewFunctions.length > 0 && <div className="studio-inspector-policies"><span>Policies</span><div>{viewFunctions.map((fn) => <Tag key={`${fn.name}:${fn.occurrence}`} minimal>{formatFunction(fn)}</Tag>)}</div></div>}
      <div className="studio-inspector-actions" role="toolbar" aria-label="Selected view actions"><Button small icon="code" onClick={() => onOpenSQL(selection)}>{relation?'Child SQL':'Open SQL'}</Button><Button small icon="play" disabled={!canRun} loading={testingView === target.name} onClick={() => onTest(selection)}>{relation?'Test child':'Test view'}</Button>{relation && <Button small icon="diagram-tree" disabled={!canRun} onClick={() => onPreviewRelation(selection)}>Preview full graph</Button>}{!relation && !derived && <Button small icon="git-branch" disabled={!canEdit} onClick={() => onAdd(target)}>Add child</Button>}{relation && !derived && <Button small icon="flows" disabled={!canEdit} onClick={() => onEditRelation(selection)}>Edit relation</Button>}<Button small icon="trash" intent="danger" disabled={!canEdit || target.root} onClick={() => onRemove(selection)}>{relation?'Remove child':'Remove view'}</Button></div>
      {relation&&<section className="studio-relation-evidence" aria-label="Relation test evidence"><div className="studio-panel-heading"><div><h3>Relation test</h3><p>{derived?'Not applicable to independent derived output.':'Attachment evidence for this exact component revision.'}</p></div>{!derived&&<Button small intent="primary" icon="endorsed" loading={testingRelation===selection.key} disabled={!canRun} onClick={()=>onRunRelation(selection)}>Run relation test</Button>}</div>{!derived&&(relationCurrent?<RelationEvidence result={relationTest}/>:<div className="studio-empty-compact">No relation test has been run for this exact revision.</div>)}</section>}
    </Card>
    <Card className="studio-card studio-columns-panel" elevation={0}>
      <div className="studio-columns-heading"><div><h2>Columns</h2><span>{columns.length} available</span></div><div className="studio-column-tools"><InputGroup leftIcon="search" aria-label="Search view columns" placeholder="Find column, source, or type" value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); }}/><Button icon="properties" disabled={!canEdit} onClick={() => onFields({ ...target, view: resolvedView })}>Manage columns</Button></div></div>
      {metadataError && <Callout intent="warning" title="Connector metadata unavailable">{metadataError}<Button small minimal intent="warning" icon="refresh" onClick={() => setMetadataReload((value) => value + 1)}>Retry metadata</Button></Callout>}
      {!metadataError && visibleColumns.length === 0 ? <div className="studio-empty-compact">{metadataLoading ? 'Loading column metadata…' : query ? 'No columns match this search.' : 'No columns are exposed by this view.'}</div> : <div className="studio-view-column-list">{visibleColumns.map((column) => <button type="button" disabled={!canEdit} key={columnIdentity(column)} onClick={() => onFields({ ...target, view: resolvedView, column })}><span><strong>{column.name}</strong><small>{column.source || column.name}</small></span><span>{column.type?.name || column.databaseType || 'inferred'}</span><Tag minimal intent={column.groupable ? 'primary' : 'none'}>{column.groupable ? 'dimension' : 'field'}</Tag><span className="studio-column-edit">Edit</span></button>)}</div>}
      {columns.length > pageSize && <ButtonGroup minimal className="studio-column-pages"><Button small icon="chevron-left" aria-label="Previous column page" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}/><span>{currentPage + 1} / {pages}</span><Button small icon="chevron-right" aria-label="Next column page" disabled={currentPage + 1 >= pages} onClick={() => setPage(currentPage + 1)}/></ButtonGroup>}
    </Card>
  </>;
}

function ComponentGraph({root,selected,canEdit,canRun,onSelect,onOpen,onOpenSQL,onAdd,onEdit,onTest,onRemove}) {
  const entries = useMemo(() => collectViewEntries(root), [root]);
  const [query, setQuery] = useState('');
  const [collapsed, setCollapsed] = useState(() => new Set());
  useEffect(() => { setQuery(''); setCollapsed(new Set()); }, [root?.name]);
  const normalized = query.trim().toLowerCase();
  const matches = normalized ? entries.filter((entry) => `${entry.lineage.join(' / ')} ${entry.view?.name || ''} ${entry.view?.source?.table || ''}`.toLowerCase().includes(normalized)) : [];
  const branchNames = entries.filter((entry) => (entry.view?.relations ?? []).length > 0).map((entry) => entry.view.namespace || entry.view.name);
  return <div className="studio-graph-browser">
    <div className="studio-graph-controls"><InputGroup leftIcon="search" aria-label="Find graph view" placeholder="Find a view, path, or table" value={query} onChange={(event) => setQuery(event.target.value)} rightElement={query ? <Button minimal icon="cross" aria-label="Clear graph search" onClick={() => setQuery('')}/> : undefined}/><Tag minimal>{entries.length} {entries.length === 1 ? 'view' : 'views'}</Tag><ButtonGroup minimal><Button small icon="collapse-all" disabled={branchNames.length === 0} onClick={() => setCollapsed(new Set(branchNames))}>Collapse</Button><Button small icon="expand-all" disabled={collapsed.size === 0} onClick={() => setCollapsed(new Set())}>Expand</Button></ButtonGroup></div>
    {normalized && <div className="studio-graph-matches" aria-live="polite"><span>{matches.length} {matches.length === 1 ? 'match' : 'matches'}</span>{matches.slice(0, 8).map((entry) => <button type="button" key={entry.view.name} onClick={() => onOpen({type:'view',name:entry.view.namespace||entry.view.name,label:entry.view.name,view:entry.view})}>{entry.lineage.join(' / ')}</button>)}{matches.length > 8 && <span>+{matches.length - 8} more</span>}</div>}
    <div className="studio-component-graph"><ViewBlock view={root} root selected={selected} canEdit={canEdit} canRun={canRun} query={normalized} collapsed={collapsed} onCollapse={(name) => setCollapsed((current) => { const next = new Set(current); if (next.has(name)) next.delete(name); else next.add(name); return next; })} onSelect={onSelect} onOpen={onOpen} onOpenSQL={onOpenSQL} onAdd={onAdd} onEdit={onEdit} onTest={onTest} onRemove={onRemove}/></div>
  </div>;
}
function ViewBlock({view,root=false,relationKind='',selected,canEdit,canRun,query='',collapsed,onCollapse,onSelect,onOpen,onOpenSQL,onAdd,onEdit,onTest,onRemove}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const name=view.namespace||view.name;
  const node={type:'view',name,label:view.name,root,relationKind,view};
  const relations=view.relations??[];
  const isCollapsed=collapsed?.has(name) && !query;
  const isMatch=query && `${displayViewName(view)} ${view.name || ''} ${view?.source?.table || ''}`.toLowerCase().includes(query);
  return <div className="studio-view-branch">
    <div className={`studio-view-block ${selected?.type==='view'&&selected.name===name?'selected':''} ${isMatch?'matched':''}`}>
      <button type="button" aria-pressed={selected?.type==='view'&&selected.name===name} className="studio-view-select" onClick={()=>onOpen(node)}>
        <span className="studio-view-kind-icon" aria-hidden="true">{root ? '◈' : '◇'}</span><strong>{displayViewName(view)}</strong>
      </button>
      <span className="studio-view-node-actions">{relations.length > 0 && <Button minimal small icon={isCollapsed ? 'chevron-right' : 'chevron-down'} aria-label={`${isCollapsed ? 'Expand' : 'Collapse'} ${displayViewName(view)} descendants`} onClick={(event)=>{event.stopPropagation();onCollapse(name)}}/>}<Popover isOpen={menuOpen} onInteraction={setMenuOpen} interactionKind={PopoverInteractionKind.CLICK} placement="right-start" content={<Menu><MenuItem icon="git-branch" text="Add child view" disabled={!canEdit||relationKind==='derived'} onClick={()=>{setMenuOpen(false);onAdd(node)}}/><MenuItem icon="code" text="Open SQL tab" onClick={()=>{setMenuOpen(false);onOpenSQL(node)}}/><MenuItem icon="play" text="Test view" disabled={!canRun} onClick={()=>{setMenuOpen(false);onTest(node)}}/><MenuItem icon="trash" intent="danger" disabled={!canEdit||root} text="Remove view" onClick={()=>{setMenuOpen(false);onRemove(node)}}/></Menu>}><Button minimal small icon="more" aria-label={`Actions for ${displayViewName(view)}`} onClick={(event)=>{event.stopPropagation();setMenuOpen((current)=>!current)}}/></Popover></span>
    </div>
    {relations.length>0&&!isCollapsed&&<div className={`studio-view-children ${relations.length>1?'branched':''}`}>{relations.map((relation)=>{
    const childView=relation.view||{name:relation.name,namespace:relation.name,source:{}};
    const childName=childView.namespace||childView.name;
    const relationKey=`${name}->${childName}`;
    const kind=relation.kind||'subview';
    const relationNode={type:'relation',key:relationKey,relation,relationKind:kind,label:relation.name,parentName:name,child:{type:'view',name:childName,label:childView.name,root:false,relationKind:kind,view:childView}};
    const relationSelected=selected?.type==='relation'&&selected.key===relationKey;
    const relationLabel=`${displayViewName(view)} → ${displayViewName(childView)}`;
    return <div className="studio-relation-branch" key={relation.name}><button type="button" aria-label={`${kind} relation, ${relationLabel}`} aria-pressed={relationSelected} className={`studio-relation-edge ${relationSelected?'selected':''}`} onClick={()=>onSelect(relationNode)}><code>{relationLabel}</code></button><ViewBlock view={childView} relationKind={kind} selected={selected} canEdit={canEdit} canRun={canRun} query={query} collapsed={collapsed} onCollapse={onCollapse} onSelect={onSelect} onOpen={onOpen} onOpenSQL={onOpenSQL} onAdd={onAdd} onEdit={onEdit} onTest={onTest} onRemove={onRemove}/></div>;
    })}</div>}
  </div>;
}
function selectedViewNode(node){return node?.type==='relation'?node.child:node?.type==='view'?node:null;}
function joinText(relation){return (relation?.on??[]).map((item)=>`${item.parentColumn} = ${item.childColumn}`).join(' AND ')||'Join condition';}
function viewSourceSQL(inspection,name){const authored=(inspection?.structure?.views??[]).find((view)=>view?.name===name||view?.name===findView(inspection?.structure?.component?.rootView,name)?.namespace)?.sql;return authored||findView(inspection?.structure?.component?.rootView,name)?.source?.sql||'';}
function findView(view,name){if(!view)return null;if(view.name===name||view.namespace===name)return view;for(const relation of view.relations??[]){const found=findView(relation.view,name);if(found)return found;}return null;}
function findRelationContext(view,name){if(!view)return null;for(const relation of view.relations??[]){const child=relation.view;if(child&&(child.name===name||child.namespace===name))return{parent:view,relation};const nested=findRelationContext(child,name);if(nested)return nested;}return null;}
function displayViewName(view){const raw=view?.name==='reader'?(view?.source?.table||view?.namespace||view?.name):(view?.name||view?.namespace||'View');return String(raw).replace(/^ci_/i,'').split(/[_\s-]+/).filter(Boolean).map((part)=>part.charAt(0).toUpperCase()+part.slice(1).toLowerCase()).join(' ') || 'View';}
function displayNodeName(name,root){return displayViewName(findView(root,name));}
function viewLineage(root,targetName,path=[]){if(!root)return[];const next=[...path,root];if(root.name===targetName||root.namespace===targetName)return next;for(const relation of root.relations??[]){const found=viewLineage(relation.view,targetName,next);if(found.length)return found;}return[];}
function collectViewEntries(root,path=[]){if(!root)return[];const next=[...path,root];return [{view:root,lineage:next.map(displayViewName)},...(root.relations??[]).flatMap((relation)=>collectViewEntries(relation.view,next))];}
function viewResourcePath(view,root){return `${viewLineage(root,view?.name).map((item)=>displayViewName(item).toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_|_$/g,'')).join('/')}.sql`;}
function columnIdentity(column){return String(column?.source||column?.name||'');}
function mergeViewColumns(compiled=[], discovered=[]){const result=new Map();for(const column of discovered??[])result.set(columnIdentity(column).toLowerCase(),column);for(const column of compiled??[]){const key=columnIdentity(column).toLowerCase();result.set(key,{...(result.get(key)||{}),...column});}return [...result.values()];}
function formatFunction(functionOccurrence){return `${functionOccurrence.name}(${(functionOccurrence.args ?? []).join(', ')})`;}
function tableReference(value){const parts=String(value||'').split('.').map((item)=>item.trim()).filter(Boolean);return parts.length>1?{schema:parts.slice(0,-1).join('.'),table:parts.at(-1)}:{table:parts[0]||''};}
function normalizedIdentity(value){return String(value||'').trim().toLowerCase().replace(/[^a-z0-9]+/g,'');}

function collectViews(structure) {
  if (Array.isArray(structure?.views) && structure.views.length > 0) {
    return structure.views.map((view) => view.name).filter(Boolean);
  }
  const result = [];
  const visit = (view) => {
    if (!view?.name || result.includes(view.name)) return;
    result.push(view.name);
    for (const relation of view.relations ?? []) visit(relation.view);
  };
  visit(structure?.component?.rootView);
  return result;
}

function ExecutionEvidence({ evidence }) {
  if (!evidence) return null;
  return <div className="studio-execution-evidence" aria-label="Execution evidence"><span><strong>Version</strong> v{evidence.versionNo || '—'} · rev {evidence.sourceRevision || '—'}</span><span><strong>Connector</strong> {evidence.connector || '—'}</span><span><strong>Rows</strong> {evidence.returnedRows ?? 0} / {evidence.limit ?? '—'}</span><span><strong>Encoded</strong> {formatBytes(evidence.encodedBytes)}</span><Tag minimal intent={evidence.truncated ? 'warning' : 'success'}>{evidence.truncated ? 'Truncated to budget' : 'Complete within budget'}</Tag></div>;
}

function RelationEvidence({ result }) {
  const metrics = [
    ['Parent rows', result?.parentRows ?? 0],
    ['Matched', result?.matchedParents ?? 0],
    ['Unmatched', result?.unmatchedParents ?? 0],
    ['Children attached', result?.attachedChildren ?? 0],
  ];
  return <div className="studio-relation-test-result">
    <div className="studio-relation-metrics">{metrics.map(([label,value])=><div key={label}><strong>{value}</strong><span>{label}</span></div>)}</div>
    <div className="studio-relation-test-meta"><Tag minimal intent="success">Executed</Tag><span>{formatDuration(result?.duration)}</span><span>v{result?.evidence?.versionNo ?? '—'} · rev {result?.evidence?.sourceRevision ?? '—'}</span></div>
  </div>;
}

function formatBytes(value) {
  const bytes=Number(value)||0;
  if(bytes<1024)return`${bytes} B`;
  if(bytes<1024*1024)return`${(bytes/1024).toFixed(1)} KB`;
  return`${(bytes/(1024*1024)).toFixed(1)} MB`;
}

function formatDuration(value) {
  const nanoseconds = Number(value);
  if (!Number.isFinite(nanoseconds)) return 'completed';
  if (nanoseconds < 1e6) return `${Math.round(nanoseconds / 1e3)}µs`;
  return `${(nanoseconds / 1e6).toFixed(1)}ms`;
}
function inputDeclarationCount(structure){return (structure?.declarations??[]).filter((item)=>item?.parameter&&!['output','component'].includes(String(item.parameter.source?.kind||'').toLowerCase())).length;}
function predicateOptionCount(structure){return (structure?.declarations??[]).reduce((count,item)=>count+(item?.predicates?.length??0),0);}

import React, { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { ForgeThemeBoundary, ForgeThemeProvider } from 'forge/theme';
import { Alert, Button, ButtonGroup, Callout, Card, HTMLSelect, HTMLTable, InputGroup, Navbar, Spinner, Tag, Tree } from '@blueprintjs/core';
import { ConnectorDialog } from './ConnectorDialog.jsx';
import { NamespaceDialog } from './NamespaceDialog.jsx';
import { OverviewWorkspace } from './OverviewWorkspace.jsx';
import { ReportDialog } from './ReportDialog.jsx';
import { RuntimeWorkspace } from './RuntimeWorkspace.jsx';
import { SecurityWorkspace } from './SecurityWorkspace.jsx';
import { overviewFromSettled } from './overviewModel.js';

const ReaderBuilder = lazy(() => import('./ReaderBuilder.jsx').then((module) => ({ default: module.ReaderBuilder })));
const SchemaBrowser = lazy(() => import('./SchemaBrowser.jsx').then((module) => ({ default: module.SchemaBrowser })));

const reportColumns = [
  { label: 'Title', key: 'title' },
  { label: 'Namespace', key: 'namespace', hideNarrow: true },
  { label: 'Connector', key: 'defaultConnectorName', hideNarrow: true },
  { label: 'Status', key: 'status' },
  { label: 'Updated', key: 'updatedAt', hideNarrow: true },
];

const connectorColumns = [
  { label: 'Name', key: 'name' },
  { label: 'Driver', key: 'driver', hideNarrow: true },
  { label: 'Status', key: 'status' },
  { label: 'Owner', key: 'ownerId', hideNarrow: true },
];

const namespaceColumns = [
  { label: 'Title', key: 'title' },
  { label: 'Namespace', key: 'name' },
  { label: 'Status', key: 'status' },
  { label: 'Updated', key: 'updatedAt', hideNarrow: true },
];

const noExtensions = [];

// StudioApp is intentionally only a Forge composition. Its data boundary is
// StudioAPI, which maps one-for-one to the public Studio SDK operation names.
export function StudioApp({ api, mode, subject, extensions = noExtensions }) {
  const [section, setSection] = useState('overview');
  const [reports, setReports] = useState([]);
  const [connectors, setConnectors] = useState([]);
  const [namespaces, setNamespaces] = useState([]);
  const [runtime, setRuntime] = useState(null);
  const [overview, setOverview] = useState(null);
  const [error, setError] = useState('');
  const [connectorConflict, setConnectorConflict] = useState(false);
  const [namespaceConflict, setNamespaceConflict] = useState(false);
  const [loading, setLoading] = useState(true);
  const [testing, setTesting] = useState('');
  const [navigationOpen, setNavigationOpen] = useState(() => typeof window === 'undefined' || !window.matchMedia?.('(max-width: 720px)').matches);
  const [navigationQuery, setNavigationQuery] = useState('');
  const [connectorDialogOpen, setConnectorDialogOpen] = useState(false);
  const [editingConnector, setEditingConnector] = useState(null);
  const [pendingConnectorDelete, setPendingConnectorDelete] = useState(null);
  const [deletingConnector, setDeletingConnector] = useState(false);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [namespaceDialogOpen, setNamespaceDialogOpen] = useState(false);
  const [editingNamespace, setEditingNamespace] = useState(null);
  const [pendingNamespaceDelete, setPendingNamespaceDelete] = useState(null);
  const [deletingNamespace, setDeletingNamespace] = useState(false);
  const [builderReport, setBuilderReport] = useState(null);
  const [openBuilderResources, setOpenBuilderResources] = useState(false);
  const [builderResourceAction, setBuilderResourceAction] = useState('');
  const [reportNamespace, setReportNamespace] = useState('');
  const [reportStatus, setReportStatus] = useState('');
  const [reportSearch, setReportSearch] = useState('');
  const [reportQuery, setReportQuery] = useState('');
  const [reportOffset, setReportOffset] = useState(0);
  const [reportHasNext, setReportHasNext] = useState(false);
  const reportPageSize = 25;
  const [namespaceSearch, setNamespaceSearch] = useState('');
  const [namespaceQuery, setNamespaceQuery] = useState('');
  const [namespaceStatus, setNamespaceStatus] = useState('');
  const [namespaceOffset, setNamespaceOffset] = useState(0);
  const [namespaceHasNext, setNamespaceHasNext] = useState(false);
  const namespacePageSize = 25;
  const [connectorSearch,setConnectorSearch]=useState('');
  const [connectorQuery,setConnectorQuery]=useState('');
  const [connectorStatus,setConnectorStatus]=useState('');
  const [connectorOffset,setConnectorOffset]=useState(0);
  const [connectorHasNext,setConnectorHasNext]=useState(false);
  const connectorPageSize=25;

  const loadSection = useCallback(() => {
    if (section === 'builder' || section === 'schema' || section === 'security' || extensions.some((extension) => extension.id === section)) { setLoading(false); return () => {}; }
    let cancelled = false;
    setError(''); setConnectorConflict(false); setNamespaceConflict(false);
    setLoading(true);
    const request = section === 'overview'
      ? Promise.allSettled([api.listReports({ limit: 6 }), api.listConnectors({ limit: 100 }), api.listNamespaces({ limit: 500 }), api.getRuntimeStatus()]).then(overviewFromSettled)
      : section === 'reports'
      ? Promise.all([api.listReports({ query: reportQuery, namespace: reportNamespace, status: reportStatus, limit: reportPageSize + 1, offset: reportOffset }), api.listNamespaces({ limit: 500 })])
      : section === 'namespaces' ? api.listNamespaces({query:namespaceQuery,status:namespaceStatus,limit:namespacePageSize+1,offset:namespaceOffset}) : ['runtime','skills'].includes(section) ? api.getRuntimeStatus() : api.listConnectors({query:connectorQuery,status:connectorStatus,limit:connectorPageSize+1,offset:connectorOffset});
    request.then((page) => {
      if (cancelled) return;
      if (section === 'overview') { setOverview(page); return; }
      if (['runtime','skills'].includes(section)) { setRuntime(page); return; }
      const result = section === 'reports' ? page[0] : page;
      const items = result?.items ?? [];
      if (section === 'reports') {
        setReports(items.slice(0, reportPageSize));
        setReportHasNext(items.length > reportPageSize);
        setNamespaces(page[1]?.items ?? []);
      }
      else if (section === 'namespaces') { setNamespaces(items.slice(0,namespacePageSize)); setNamespaceHasNext(items.length>namespacePageSize); }
      else { setConnectors(items.slice(0,connectorPageSize)); setConnectorHasNext(items.length>connectorPageSize); }
    }).catch((cause) => !cancelled && setError(cause.message))
      .finally(() => !cancelled && setLoading(false));
    return () => { cancelled = true; };
  }, [api, section, extensions, reportNamespace, reportOffset, reportQuery, reportStatus, namespaceOffset, namespaceQuery, namespaceStatus, connectorOffset, connectorQuery, connectorStatus]);
  useEffect(loadSection, [loadSection]);

  const items = section === 'reports' ? reports : section === 'namespaces' ? namespaces : connectors;
  const visibleItems = items;
  const reportNamespaces = [...new Set((namespaces.length ? namespaces.map((item) => item.name) : reports.map((item) => item.namespace || 'general')).filter(Boolean))].sort();
  const columns = section === 'reports' ? reportColumns : section === 'namespaces' ? namespaceColumns : connectorColumns;
  const testConnector = async (name) => {
    setTesting(name); setError('');
    try {
      await api.testConnector(name);
      await loadSection();
    } catch (cause) { setError(cause.message); }
    finally { setTesting(''); }
  };
  const changeConnectorStatus = async (connector) => {
    setTesting(connector.name); setError(''); setConnectorConflict(false);
    try {
      if (connector.status === 'active') await api.disableConnector(connector.name, connector.etag);
      else await api.activateConnector(connector.name, connector.etag);
      await loadSection();
    } catch (cause) { setConnectorConflict(cause?.code === 'conflict'); setError(cause.message); }
    finally { setTesting(''); }
  };
  const onConnectorCreated = async () => {
    setEditingConnector(null);
    setSection('connectors');
    await loadSection();
  };
  const deleteConnector = async () => {
    if (!pendingConnectorDelete) return;
    setDeletingConnector(true); setError(''); setConnectorConflict(false);
    try {
      await api.deleteConnector(pendingConnectorDelete.name, pendingConnectorDelete.etag);
      setPendingConnectorDelete(null);
      await loadSection();
    } catch (cause) {
      setPendingConnectorDelete(null);
      const stale = cause?.code === 'conflict' && String(cause.message).includes('etag');
      setConnectorConflict(stale); setError(cause.message);
    } finally { setDeletingConnector(false); }
  };
  const onReportCreated = async () => {
    setSection('reports');
    await loadSection();
  };
  const onNamespaceSaved = async () => {
    setEditingNamespace(null);
    setSection('namespaces');
    await loadSection();
  };
  const deleteNamespace = async () => {
    if (!pendingNamespaceDelete) return;
    setDeletingNamespace(true); setError('');
    try {
      await api.deleteNamespace(pendingNamespaceDelete.name, pendingNamespaceDelete.etag);
      setPendingNamespaceDelete(null);
      await loadSection();
    } catch (cause) { setPendingNamespaceDelete(null); setNamespaceConflict(cause?.code==='conflict'&&String(cause.message).includes('etag')); setError(cause.message); }
    finally { setDeletingNamespace(false); }
  };
  const openBuilder = (report) => { setBuilderReport(report); setOpenBuilderResources(Boolean(report?.openResources)); setBuilderResourceAction(report?.resourceAction||''); setSection('builder'); };
  const navigate = (target) => {
    if (!target) return;
    setSection(target);
    if (typeof window !== 'undefined' && window.matchMedia?.('(max-width: 720px)').matches) setNavigationOpen(false);
  };
  const extensionNavigation = extensions.map((extension) => ({ id: extension.id, label: extension.label, icon: extension.icon, className: `studio-nav-extension studio-nav-${extension.id}`, section: extension.id, isSelected: section === extension.id }));
  const navigation = [
    { id: 'overview', label: 'Overview', icon: 'home', className: 'studio-nav-overview', section: 'overview', isSelected: section==='overview' },
    { id: 'connectors', label: 'Connectors', icon: 'database', className: 'studio-nav-connectors', section: 'connectors', isSelected: section==='connectors' },
    { id: 'schema', label: 'Schema Browser', icon: 'diagram-tree', className: 'studio-nav-schema', section: 'schema', isSelected: section==='schema' },
    { id: 'namespaces', label: 'Namespaces', icon: 'folder-shared', className: 'studio-nav-namespaces', section: 'namespaces', isSelected: section==='namespaces' },
    { id: 'security', label: 'Security', icon: 'shield', className: 'studio-nav-security', section: 'security', isSelected: section==='security' },
    { id: 'runtime', label: 'Runtime', icon: 'pulse', className: 'studio-nav-runtime', section: 'runtime', isSelected: section==='runtime' },
    { id: 'skills', label: 'Skills & Resources', icon: 'learning', className: 'studio-nav-skills', section: 'skills', isSelected: section==='skills' },
    { id: 'reports', label: 'Components', icon: 'application', className: 'studio-nav-components', section: 'reports', isSelected: section==='reports'||section==='builder', childNodes: [
      { id: 'reports-catalog', label: 'Catalog', icon: 'th', className: 'studio-nav-components-catalog', section: 'reports', isSelected: section==='reports' },
    ] }, ...extensionNavigation,
  ].filter((item) => item.label.toLowerCase().includes(navigationQuery.toLowerCase()) || item.childNodes?.some((child) => child.label.toLowerCase().includes(navigationQuery.toLowerCase())));
  return <ForgeThemeProvider><ForgeThemeBoundary windowKey="datly-studio">
    <div className="studio-app">
    <Navbar className="studio-navbar"><Navbar.Group align="left"><Navbar.Heading>Datly Studio</Navbar.Heading><Navbar.Divider/>
      <Button icon="menu" minimal title="Toggle navigation" onClick={() => setNavigationOpen((open) => !open)} />
    </Navbar.Group><Navbar.Group align="right"><Tag minimal intent={mode === 'development' ? 'warning' : 'success'}>{mode === 'development' ? 'Development user' : subject || 'Authenticated user'}</Tag></Navbar.Group></Navbar>
    <div className="studio-app-body">
    {navigationOpen && <button type="button" className="studio-sidebar-scrim" aria-label="Close navigation" onClick={() => setNavigationOpen(false)}/>}
    {navigationOpen && <aside className="studio-sidebar"><InputGroup leftIcon="search" placeholder="Find Studio resource" value={navigationQuery} onChange={(event) => setNavigationQuery(event.target.value)} />
      <Tree className="studio-tree" contents={navigation} onNodeClick={(node) => navigate(node.section)} />
    </aside>}
    {section === 'builder' ? <Suspense fallback={<WorkspaceLoading label="Loading component workspace"/>}><ReaderBuilder api={api} report={builderReport} openResources={openBuilderResources} resourceAction={builderResourceAction} onReportUpdated={setBuilderReport} onBack={() => setSection('reports')} /></Suspense> : section === 'schema' ? <Suspense fallback={<WorkspaceLoading label="Loading schema browser"/>}><SchemaBrowser api={api} onOpenBuilder={openBuilder} /></Suspense> : section === 'security' ? <SecurityWorkspace api={api}/> : extensions.some((extension) => extension.id === section) ? extensions.find((extension) => extension.id === section).render({ api, mode, subject, navigate, openComponent: openBuilder }) : section === 'overview' ? <OverviewWorkspace data={overview} loading={loading} error={error} onRefresh={loadSection} onNavigate={navigate} onOpenComponent={openBuilder}/> : ['runtime','skills'].includes(section) ? <RuntimeWorkspace api={api} mode={section} status={runtime} loading={loading} error={error} onRefresh={loadSection} onOpenComponent={openBuilder}/> : <main className="studio-workspace">
      <h1 className="studio-page-heading">{section === 'connectors' ? 'Connectors' : section === 'namespaces' ? 'Namespaces' : 'Components'}</h1>
      <p className="studio-page-description">{section === 'connectors' ? 'Validate and manage data sources used by Studio readers.' : section === 'namespaces' ? 'Govern production component groups without changing Datly package identity.' : 'Create and manage read-only dynamic Datly components.'}</p>
      {section === 'connectors'&&<form className="studio-page-actions studio-catalog-actions" onSubmit={(event)=>{event.preventDefault();setConnectorOffset(0);setConnectorQuery(connectorSearch.trim());}}><InputGroup leftIcon="search" aria-label="Search connectors" placeholder="Find connector, driver, or owner" value={connectorSearch} onChange={(event)=>setConnectorSearch(event.target.value)} rightElement={connectorSearch?<Button type="button" minimal icon="cross" aria-label="Clear connector search" onClick={()=>{setConnectorSearch('');setConnectorQuery('');setConnectorOffset(0);}}/>:undefined}/><HTMLSelect aria-label="Filter connectors by status" value={connectorStatus} onChange={(event)=>{setConnectorStatus(event.target.value);setConnectorOffset(0);}}><option value="">All states</option><option value="active">Active</option><option value="draft">Draft</option><option value="disabled">Disabled</option></HTMLSelect><ButtonGroup className="studio-catalog-primary-actions"><Button type="submit" icon="search">Search</Button><Button type="button" intent="primary" icon="add" onClick={()=>{setEditingConnector(null);setConnectorDialogOpen(true);}}>New connector</Button></ButtonGroup></form>}
      {section === 'namespaces' && <form className="studio-page-actions studio-catalog-actions studio-namespace-actions" onSubmit={(event)=>{event.preventDefault();setNamespaceOffset(0);setNamespaceQuery(namespaceSearch.trim());}}><InputGroup leftIcon="search" aria-label="Search namespaces" placeholder="Find namespace, title, or description" value={namespaceSearch} onChange={(event)=>setNamespaceSearch(event.target.value)} rightElement={namespaceSearch?<Button type="button" minimal icon="cross" aria-label="Clear namespace search" onClick={()=>{setNamespaceSearch('');setNamespaceQuery('');setNamespaceOffset(0);}}/>:undefined}/><HTMLSelect aria-label="Filter namespaces by status" value={namespaceStatus} onChange={(event)=>{setNamespaceStatus(event.target.value);setNamespaceOffset(0);}}><option value="">All states</option><option value="active">Active</option><option value="archived">Archived</option></HTMLSelect><ButtonGroup className="studio-catalog-primary-actions"><Button type="submit" icon="search">Search</Button><Button type="button" intent="primary" icon="add" onClick={() => { setEditingNamespace(null); setNamespaceDialogOpen(true); }}>New namespace</Button></ButtonGroup></form>}
      {section === 'reports' && <form className="studio-page-actions studio-catalog-actions" onSubmit={(event) => { event.preventDefault(); setReportOffset(0); setReportQuery(reportSearch.trim()); }}><InputGroup leftIcon="search" aria-label="Search components" placeholder="Find title, slug, or description" value={reportSearch} onChange={(event) => setReportSearch(event.target.value)} rightElement={reportSearch ? <Button type="button" minimal icon="cross" aria-label="Clear component search" onClick={() => { setReportSearch(''); setReportQuery(''); setReportOffset(0); }}/> : undefined}/><HTMLSelect aria-label="Filter components by namespace" value={reportNamespace} onChange={(event) => { setReportNamespace(event.target.value); setReportOffset(0); }}><option value="">All namespaces</option>{reportNamespaces.map((namespace) => <option key={namespace} value={namespace}>{namespace}</option>)}</HTMLSelect><HTMLSelect aria-label="Filter components by status" value={reportStatus} onChange={(event) => { setReportStatus(event.target.value); setReportOffset(0); }}><option value="">All states</option><option value="draft">Draft</option><option value="active">Active</option><option value="archived">Archived</option></HTMLSelect><ButtonGroup className="studio-catalog-primary-actions"><Button type="submit" icon="search">Search</Button><Button type="button" intent="primary" icon="add" onClick={() => setReportDialogOpen(true)}>New component</Button></ButtonGroup></form>}
      {connectorConflict && <Callout intent="warning" title="Connector changed elsewhere" role="alert" style={{ marginBottom: 16 }}>The connector lifecycle state changed after this catalog was loaded. No change was applied. <Button small intent="warning" minimal icon="refresh" onClick={loadSection}>Refresh catalog</Button></Callout>}
      {namespaceConflict&&<Callout intent="warning" title="Namespace changed elsewhere" role="alert" style={{marginBottom:16}}>No namespace change was applied. Refresh the governed catalog before reviewing and retrying.<Button small intent="warning" minimal icon="refresh" onClick={loadSection}>Refresh catalog</Button></Callout>}
      {error && !connectorConflict&&!namespaceConflict && <Callout intent="danger" title="Studio request failed" role="alert" style={{ marginBottom: 16 }}>{error}<Button small intent="danger" minimal onClick={loadSection}>Retry</Button></Callout>}
      <Card className="studio-card" elevation={0}>
      {loading ? <Spinner size={24} /> : <div className="studio-table-wrap"><HTMLTable className="studio-data-table" striped interactive>
        <thead><tr>{columns.map((column) => <th className={column.hideNarrow ? 'studio-hide-narrow' : ''} key={column.key} style={{ padding: 8 }}>{column.label}</th>)}{section === 'connectors' && <><th className="studio-hide-narrow">Connectivity</th><th aria-label="Connector actions" /></>}{section === 'namespaces' && <th aria-label="Namespace actions" />}{section === 'reports' && <th aria-label="Reader actions" />}</tr></thead>
        <tbody>{visibleItems.map((item) => <tr key={item.id ?? item.name}>
          {columns.map((column) => <td className={column.hideNarrow ? 'studio-hide-narrow' : ''} key={column.key} style={{ borderTop: '1px solid #ddd', padding: 8 }}>{column.key === 'status' ? <><Tag intent={statusIntent(item.status)} minimal>{item.status}</Tag>{section === 'connectors' && item.lastTestStatus && <span className="studio-compact-connectivity"><Tag intent={statusIntent(item.lastTestStatus)} minimal>{item.lastTestStatus}</Tag></span>}</> : displayValue(column.key, item[column.key])}</td>)}
          {section === 'connectors' && <><td className="studio-hide-narrow">{item.lastTestStatus ? <Tag intent={statusIntent(item.lastTestStatus)} minimal>{item.lastTestStatus}</Tag> : <span className="studio-muted">Not tested</span>}</td><td><ButtonGroup minimal>
            <Button small icon="refresh" title="Test connector" aria-label="Test connector" loading={testing === item.name} onClick={() => testConnector(item.name)} />
            <Button small icon="edit" title="Edit connector" aria-label={`Edit ${item.name}`} onClick={() => { setEditingConnector(item); setConnectorDialogOpen(true); }} />
            {(item.status === 'active' || item.lastTestStatus === 'passed') && <Button small icon={item.status === 'active' ? 'power' : 'tick'} title={item.status === 'active' ? 'Disable connector' : 'Activate connector'} aria-label={item.status === 'active' ? 'Disable connector' : 'Activate connector'} loading={testing === item.name} onClick={() => changeConnectorStatus(item)} />}
            <Button small icon="trash" intent="danger" disabled={item.status === 'active'} title={item.status === 'active' ? 'Disable connector before deleting' : 'Delete connector'} aria-label={`Delete ${item.name}`} onClick={() => setPendingConnectorDelete(item)} />
          </ButtonGroup></td></>}
          {section === 'namespaces' && <td><ButtonGroup minimal>
            <Button small icon="edit" title="Edit namespace" aria-label={`Edit ${item.name}`} onClick={() => { setEditingNamespace(item); setNamespaceDialogOpen(true); }}/>
            <Button small icon="trash" intent="danger" title="Delete namespace" aria-label={`Delete ${item.name}`} onClick={() => setPendingNamespaceDelete(item)}/>
          </ButtonGroup></td>}
          {section === 'reports' && <td><Button small icon="application" title="Open Reader Builder" aria-label="Open Reader Builder" onClick={() => openBuilder(item)} /></td>}
        </tr>)}</tbody>
      </HTMLTable></div>}{!loading && !error && items.length === 0 && <div className="studio-empty"><h2>{section === 'reports' && (reportQuery || reportNamespace || reportStatus) ? 'No matching components' : section==='namespaces'&&(namespaceQuery||namespaceStatus)?'No matching namespaces':section==='connectors'&&(connectorQuery||connectorStatus)?'No matching connectors':`No ${section === 'reports' ? 'components' : section} yet`}</h2><span>{section === 'connectors' ? connectorQuery||connectorStatus?'Adjust the search or lifecycle filter to broaden the connector catalog.':'Create a connector, test it, then activate it for Studio readers.' : section === 'namespaces' ? namespaceQuery||namespaceStatus?'Adjust the search or lifecycle filter to broaden the governed catalog.':'Create a governed namespace before organizing production components.' : reportQuery || reportNamespace || reportStatus ? 'Adjust the search or filters to broaden the catalog.' : 'Create the versioned home for your first dynamic reader component.'}</span>{section === 'connectors' ? !(connectorQuery||connectorStatus)&&<Button intent="primary" icon="add" onClick={() => setConnectorDialogOpen(true)} style={{ marginTop: 16 }}>New connector</Button> : section === 'namespaces' ? !(namespaceQuery||namespaceStatus)&&<Button intent="primary" icon="add" onClick={() => setNamespaceDialogOpen(true)} style={{ marginTop: 16 }}>New namespace</Button> : !(reportQuery || reportNamespace || reportStatus) && <Button intent="primary" icon="add" onClick={() => setReportDialogOpen(true)} style={{ marginTop: 16 }}>New component</Button>}</div>}</Card>
      {section === 'reports' && !loading && (reportOffset > 0 || reportHasNext) && <div className="studio-catalog-pagination"><span>Showing {reportOffset + 1}–{reportOffset + reports.length}</span><ButtonGroup minimal><Button icon="chevron-left" disabled={reportOffset === 0} onClick={() => setReportOffset(Math.max(0, reportOffset - reportPageSize))}>Previous</Button><Button icon="chevron-right" disabled={!reportHasNext} onClick={() => setReportOffset(reportOffset + reportPageSize)}>Next</Button></ButtonGroup></div>}
      {section === 'namespaces'&&!loading&&(namespaceOffset>0||namespaceHasNext)&&<div className="studio-catalog-pagination"><span>Showing {namespaceOffset+1}–{namespaceOffset+namespaces.length}</span><ButtonGroup minimal><Button icon="chevron-left" disabled={namespaceOffset===0} onClick={()=>setNamespaceOffset(Math.max(0,namespaceOffset-namespacePageSize))}>Previous</Button><Button icon="chevron-right" disabled={!namespaceHasNext} onClick={()=>setNamespaceOffset(namespaceOffset+namespacePageSize)}>Next</Button></ButtonGroup></div>}
      {section==='connectors'&&!loading&&(connectorOffset>0||connectorHasNext)&&<div className="studio-catalog-pagination"><span>Showing {connectorOffset+1}–{connectorOffset+connectors.length}</span><ButtonGroup minimal><Button icon="chevron-left" disabled={connectorOffset===0} onClick={()=>setConnectorOffset(Math.max(0,connectorOffset-connectorPageSize))}>Previous</Button><Button icon="chevron-right" disabled={!connectorHasNext} onClick={()=>setConnectorOffset(connectorOffset+connectorPageSize)}>Next</Button></ButtonGroup></div>}
    </main>}</div></div>
    <ConnectorDialog api={api} connector={editingConnector} isOpen={connectorDialogOpen} onClose={() => { setConnectorDialogOpen(false); setEditingConnector(null); }} onCreated={onConnectorCreated} />
    <ReportDialog api={api} isOpen={reportDialogOpen} onClose={() => setReportDialogOpen(false)} onCreated={onReportCreated} />
    <NamespaceDialog api={api} namespace={editingNamespace} isOpen={namespaceDialogOpen} onClose={() => { setNamespaceDialogOpen(false); setEditingNamespace(null); }} onSaved={onNamespaceSaved}/>
    <Alert isOpen={Boolean(pendingConnectorDelete)} intent="danger" icon="trash" confirmButtonText="Delete connector" cancelButtonText="Cancel" loading={deletingConnector} onCancel={() => setPendingConnectorDelete(null)} onConfirm={deleteConnector} canEscapeKeyCancel canOutsideClickCancel>
      <p>Delete <strong>{pendingConnectorDelete?.name}</strong> from the Studio connector catalog?</p>
      <p className="studio-muted">Active connectors must be disabled first. Studio blocks deletion while any report still references this connector.</p>
    </Alert>
    <Alert isOpen={Boolean(pendingNamespaceDelete)} intent="danger" icon="trash" confirmButtonText="Delete namespace" cancelButtonText="Cancel" loading={deletingNamespace} onCancel={() => setPendingNamespaceDelete(null)} onConfirm={deleteNamespace} canEscapeKeyCancel canOutsideClickCancel>
      <p>Delete <strong>{pendingNamespaceDelete?.name}</strong>?</p>
      <p className="studio-muted">Studio blocks deletion while any component still belongs to this namespace.</p>
    </Alert>
  </ForgeThemeBoundary></ForgeThemeProvider>;
}

function WorkspaceLoading({ label }) {
  return <main className="studio-workspace"><div className="studio-loading" role="status" aria-label={label}><Spinner size={28}/></div></main>;
}

function statusIntent(status) {
  switch (String(status || '').toLowerCase()) {
    case 'active': case 'passed': return 'success';
    case 'draft': return 'warning';
    case 'failed': return 'danger';
    case 'disabled': return 'none';
    default: return 'none';
  }
}

function displayValue(key, value) {
  if (value == null) return '';
  if (key !== 'updatedAt' && key !== 'createdAt') return value;
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date);
}

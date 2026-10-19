import { ModuleLoadBoundary } from './ModuleLoadBoundary.jsx';
import nativeNavigation from './nativeNavigation.json' with {type:'json'};
import React, { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { ForgeThemeBoundary, ForgeThemeProvider } from 'forge/theme';
import { Button, ButtonGroup, Callout, Card, HTMLSelect, HTMLTable, InputGroup, Navbar, Spinner, Tag, Tree } from '@blueprintjs/core';
import { ManagementWindow } from './ManagementWindow.jsx';
import { OverviewWorkspace } from './OverviewWorkspace.jsx';
import { ReportDialog } from './ReportDialog.jsx';
import { SkillsWorkspace } from './SkillsWorkspace.jsx';
import { RuntimeWorkspace } from './RuntimeWorkspace.jsx';
import { ImportComponentButton, DownloadComponentButton } from './ComponentTransfer.jsx';
import { SecurityCenter as SecurityWorkspace } from './SecurityCenter.jsx';
import { overviewFromSettled } from './overviewModel.js';
import { managementSection } from './managementNavigation.js';
import { useNamespaceWorkspace } from './useNamespaceWorkspace.js';

const ReaderBuilder = lazy(() => import('./ReaderBuilder.jsx').then((module) => ({ default: module.ReaderBuilder })));
const SchemaBrowser = lazy(() => import('./SchemaBrowser.jsx').then((module) => ({ default: module.SchemaBrowser })));

const reportColumns = [
  { label: 'Title', key: 'title' },
  { label: 'Namespace', key: 'namespace', hideNarrow: true },
  { label: 'Connector', key: 'defaultConnectorName', hideNarrow: true },
  { label: 'Status', key: 'status' },
  { label: 'Updated', key: 'updatedAt', hideNarrow: true },
];

const noExtensions = [];

// StudioApp is intentionally only a Forge composition. Its data boundary is
// StudioAPI, which maps one-for-one to the public Studio SDK operation names.
export function StudioApp({ api, mode, subject, brand = 'Data Studio', extensions = noExtensions, renderNamespaceSkills, onSignOut }) {
  const [skillEditing, setSkillEditing] = useState(false);
  const [section, setSection] = useState(() => managementSection(globalThis.location?.search, extensions));
  const [reports, setReports] = useState([]);
  const [namespaces, setNamespaces] = useState([]);
  const [runtime, setRuntime] = useState(null);
  const [overview, setOverview] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [signingOut, setSigningOut] = useState(false);
  const [signOutError, setSignOutError] = useState('');
  const [navigationOpen, setNavigationOpen] = useState(() => typeof window === 'undefined' || !window.matchMedia?.('(max-width: 720px)').matches);
  const [navigationQuery, setNavigationQuery] = useState('');
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
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
  const namespaceWorkspace = useNamespaceWorkspace(api, subject, skillEditing || section === 'builder' || section === 'security' || extensions.some((item) => item.id === section) || section === 'connectors' || section === 'namespaces' || reportDialogOpen);
  const currentNamespaceId = namespaceWorkspace.current?.namespaceId;
  const namespaceChanging = namespaceWorkspace.enabled && !namespaceWorkspace.ready;
  useEffect(() => {
    setReports([]);
    setRuntime(null);
    setOverview(null);
    setReportNamespace('');
    setReportOffset(0);
  }, [currentNamespaceId]);

  const loadSection = useCallback(() => {
    if (namespaceWorkspace.enabled && (!namespaceWorkspace.ready || !currentNamespaceId) && !['namespaces', 'connectors', 'schema'].includes(section)) { setLoading(false); return () => {}; }
    if (section === 'connectors' || section === 'namespaces' || section === 'skills' || section === 'builder' || section === 'schema' || section === 'security' || extensions.some((extension) => extension.id === section)) { setLoading(false); return () => {}; }
    let cancelled = false;
    setError('');
    setLoading(true);
    const request = section === 'overview'
      ? Promise.allSettled([api.listComponents({ limit: 6 }), api.listConnectors({ limit: 100 }), api.listNamespaces({ limit: 500 }), api.getRuntimeStatus()]).then(overviewFromSettled)
      : section === 'components'
      ? Promise.all([api.listComponents({ query: reportQuery, namespace: reportNamespace, status: reportStatus, limit: reportPageSize + 1, offset: reportOffset }), api.listNamespaces({ limit: 500 })])
      : api.getRuntimeStatus();
    request.then((page) => {
      if (cancelled) return;
      if (section === 'overview') { setOverview(page); return; }
      if (section === 'runtime') { setRuntime(page); return; }
      const result = section === 'components' ? page[0] : page;
      const items = result?.items ?? [];
      if (section === 'components') {
        setReports(items.slice(0, reportPageSize));
        setReportHasNext(items.length > reportPageSize);
        setNamespaces(page[1]?.items ?? []);
      }
    }).catch((cause) => !cancelled && setError(cause.message))
      .finally(() => !cancelled && setLoading(false));
    return () => { cancelled = true; };
  }, [api, section, extensions, reportNamespace, reportOffset, reportQuery, reportStatus, currentNamespaceId, namespaceWorkspace.ready, namespaceWorkspace.enabled]);
  useEffect(loadSection, [loadSection]);

  const items = reports;
  const visibleItems = items;
  const reportNamespaces = [...new Set((namespaces.length ? namespaces.map((item) => item.name) : reports.map((item) => item.namespace || 'general')).filter(Boolean))].sort();
  const columns = reportColumns;
  const onReportCreated = (report) => openBuilder(report);
  const onManagementOperation = useCallback((operation) => {
    if (['namespaces.create', 'namespaces.update', 'namespaces.delete'].includes(operation)) return namespaceWorkspace.refresh?.();
  }, [namespaceWorkspace.refresh]);
  const openBuilder = (report) => { setBuilderReport(report); setOpenBuilderResources(Boolean(report?.openResources)); setBuilderResourceAction(report?.resourceAction||''); setSection('builder'); };
  const navigate = (target) => {
    if (!target) return;
    if (skillEditing && target !== section) { setError('Close the skill editor before leaving this workspace.'); return; }
    setSection(target);
    if (typeof window !== 'undefined' && window.matchMedia?.('(max-width: 720px)').matches) setNavigationOpen(false);
  };
  const signOut = async () => {
    if (!onSignOut) return;
    setSigningOut(true);
    setSignOutError('');
    try { await onSignOut(); }
    catch (cause) { setSignOutError(cause?.message || 'Sign out failed. Try again.'); }
    finally { setSigningOut(false); }
  };
  const extensionNavigation = extensions.map((extension) => ({ id: extension.id, label: extension.label, icon: extension.icon, className: `studio-nav-extension studio-nav-${extension.id}`, section: extension.id, isSelected: section === extension.id }));
  const nativeNode = (item) => ({id:item.id,label:item.title,icon:item.icon,className:`studio-nav-${item.id}`,section:item.section||item.id,isSelected:section===(item.section||item.id)||(item.id==='components'&&section==='builder'),...(item.children?{childNodes:item.children.map(nativeNode)}:{})});
  const navigation = [...nativeNavigation.menus.map(nativeNode), ...extensionNavigation].filter((item) => item.label.toLowerCase().includes(navigationQuery.toLowerCase()) || item.childNodes?.some((child) => child.label.toLowerCase().includes(navigationQuery.toLowerCase())));
  const navigationButton = (node) => <Button className="studio-navigation-label" minimal onClick={(event) => { event.stopPropagation(); navigate(node.section); }}>{node.label}</Button>;
  const interactiveNavigation = navigation.map((node) => ({ ...node, label: navigationButton(node), childNodes: node.childNodes?.map((child) => ({ ...child, label: navigationButton(child) })) }));
  return <ForgeThemeProvider><ForgeThemeBoundary windowKey="datly-studio">
    <div className="studio-app">
    <Navbar className="studio-navbar"><Navbar.Group align="left"><Navbar.Heading>{brand}</Navbar.Heading><Navbar.Divider/>
      <Button icon="menu" minimal title="Toggle navigation" onClick={() => setNavigationOpen((open) => !open)} />
      {namespaceWorkspace.enabled && <HTMLSelect className="studio-namespace-selector" aria-label="Current namespace" value={currentNamespaceId || ''} disabled={!namespaceWorkspace.ready || skillEditing} onChange={(event) => { namespaceWorkspace.select(event.target.value); setReportNamespace(''); setReportOffset(0); }}><option value="" disabled>{namespaceWorkspace.ready ? 'Choose namespace' : 'Loading namespaces…'}</option>{namespaceWorkspace.items.map((item) => <option key={item.namespaceId} value={item.namespaceId}>{namespaceWorkspace.items.filter((row) => (row.title || row.name) === (item.title || item.name)).length > 1 ? `${item.title || item.name} · ${item.name} · ${item.ownerId}` : item.title || item.name}</option>)}</HTMLSelect>}
    </Navbar.Group><Navbar.Group align="right"><Tag className="studio-identity-tag" minimal intent={mode === 'development' ? 'warning' : 'success'}>{mode === 'development' ? 'Development user' : subject || 'Authenticated user'}</Tag>{onSignOut && <Button className="studio-signout" title="Sign out" aria-label="Sign out" minimal icon="log-out" loading={signingOut} onClick={signOut}>Sign out</Button>}</Navbar.Group></Navbar>
    {signOutError && <Callout className="studio-signout-error" intent="danger" role="alert" title="Could not sign out">{signOutError}<Button small minimal intent="danger" onClick={signOut}>Try again</Button></Callout>}
    {skillEditing && error && <Callout intent="warning" role="alert">{error}</Callout>}
    {namespaceWorkspace.error && <Callout intent="warning" role="alert">{namespaceWorkspace.error}{!namespaceWorkspace.synchronizationError && <Button small minimal icon="refresh" disabled={!namespaceWorkspace.canRetry} onClick={namespaceWorkspace.refresh}>Retry namespaces</Button>}</Callout>}
    <div className="studio-app-body">
    {navigationOpen && <button type="button" className="studio-sidebar-scrim" aria-label="Close navigation" onClick={() => setNavigationOpen(false)}/>}
    {navigationOpen && <aside className="studio-sidebar"><InputGroup leftIcon="search" placeholder="Find Studio resource" value={navigationQuery} onChange={(event) => setNavigationQuery(event.target.value)} />
      <Tree className="studio-tree" contents={interactiveNavigation} onNodeClick={(node) => navigate(node.section)} />
    </aside>}
    {namespaceWorkspace.enabled && !currentNamespaceId && !['namespaces', 'connectors', 'schema'].includes(section) ? <main className="studio-main"><Callout title="Choose a namespace">Select an available namespace above, or create one in Namespaces.</Callout></main> : ['connectors', 'namespaces'].includes(section) ? <main className="studio-workspace"><ManagementWindow key={section} api={api} uri={`window://datastudio/${section}`} instanceKey={`native-datastudio-${section}`} onOperationCompleted={onManagementOperation}/></main> : section === 'builder' ? <ModuleLoadBoundary key="builder" onBack={() => setSection('components')}><Suspense fallback={<WorkspaceLoading label="Loading component workspace"/>}><ReaderBuilder api={api} report={builderReport} openResources={openBuilderResources} resourceAction={builderResourceAction} onReportUpdated={setBuilderReport} onResourcesClosed={()=>{setOpenBuilderResources(false);setBuilderResourceAction('');}} onBack={() => setSection('components')} /></Suspense></ModuleLoadBoundary> : section === 'schema' ? <ModuleLoadBoundary key="schema" onBack={() => setSection('components')}><Suspense fallback={<WorkspaceLoading label="Loading schema browser"/>}><SchemaBrowser api={api} onOpenBuilder={openBuilder} /></Suspense></ModuleLoadBoundary> : section === 'security' ? <SecurityWorkspace api={api}/> : extensions.some((extension) => extension.id === section) ? extensions.find((extension) => extension.id === section).render({ api, mode, subject, navigate, openComponent: openBuilder, currentNamespace: namespaceWorkspace.current, namespaceItems: namespaceWorkspace.items }) : section === 'overview' ? <OverviewWorkspace data={overview} loading={loading} error={error} onRefresh={loadSection} onNavigate={navigate} onOpenComponent={openBuilder}/> : section === 'skills' ? <SkillsWorkspace key={currentNamespaceId || 'unscoped'} api={api} namespace={namespaceWorkspace.current} subject={subject} renderNamespaceSkills={renderNamespaceSkills} onDirtyChange={setSkillEditing} onOpenComponent={openBuilder}/> : section === 'runtime' ? <RuntimeWorkspace key={currentNamespaceId || "unscoped"} api={api} mode={section} status={runtime} loading={loading} error={error} onRefresh={loadSection} onOpenComponent={openBuilder}/> : <main className="studio-workspace">
      <div className="studio-page-heading-row"><h1 className="studio-page-heading">Components</h1>{section==='components'&&<ImportComponentButton api={api} onImported={openBuilder}/>}</div>
      <p className="studio-page-description">Create and manage read-only dynamic Datly components.</p>
      {section === 'components' && <form className="studio-page-actions studio-catalog-actions" onSubmit={(event) => { event.preventDefault(); setReportOffset(0); setReportQuery(reportSearch.trim()); }}><InputGroup leftIcon="search" aria-label="Search components" placeholder="Find title, slug, or description" value={reportSearch} onChange={(event) => setReportSearch(event.target.value)} rightElement={reportSearch ? <Button type="button" minimal icon="cross" aria-label="Clear component search" onClick={() => { setReportSearch(''); setReportQuery(''); setReportOffset(0); }}/> : undefined}/>{!namespaceWorkspace.enabled && <HTMLSelect aria-label="Filter components by namespace" value={reportNamespace} onChange={(event) => { setReportNamespace(event.target.value); setReportOffset(0); }}><option value="">All namespaces</option>{reportNamespaces.map((namespace) => <option key={namespace} value={namespace}>{namespace}</option>)}</HTMLSelect>}<HTMLSelect aria-label="Filter components by status" value={reportStatus} onChange={(event) => { setReportStatus(event.target.value); setReportOffset(0); }}><option value="">All states</option><option value="draft">Draft</option><option value="active">Active</option><option value="archived">Archived</option></HTMLSelect><div className="studio-catalog-primary-actions"><Button type="submit" icon="search" title="Search components" aria-label="Search components"/><Button type="button" intent="primary" icon="add" title="New component" aria-label="New component" disabled={namespaceChanging} onClick={() => setReportDialogOpen(true)}>New component</Button></div></form>}
      {error && <Callout intent="danger" title="Studio request failed" role="alert" style={{ marginBottom: 16 }}>{error}<Button small intent="danger" minimal onClick={loadSection}>Retry</Button></Callout>}
      <Card className="studio-card" elevation={0}>
      {loading ? <Spinner size={24} /> : <div className="studio-table-wrap"><HTMLTable className="studio-data-table" striped interactive>
        <thead><tr>{columns.map((column) => <th className={column.hideNarrow ? 'studio-hide-narrow' : ''} key={column.key} style={{ padding: 8 }}>{column.label}</th>)}{section === 'components' && <th aria-label="Reader actions" />}</tr></thead>
        <tbody>{visibleItems.map((item) => <tr key={item.id ?? item.name}>
          {columns.map((column) => <td className={column.hideNarrow ? 'studio-hide-narrow' : ''} key={column.key} style={{ borderTop: '1px solid #ddd', padding: 8 }}>{column.key === 'status' ? <><Tag intent={statusIntent(item.status)} minimal>{item.status}</Tag></> : displayValue(column.key, item[column.key])}</td>)}
          {section === 'components' && <td><ButtonGroup minimal><Button small icon="application" title="Open Reader Builder" aria-label="Open Reader Builder" onClick={() => openBuilder(item)} /><DownloadComponentButton api={api} report={item}/></ButtonGroup></td>}
        </tr>)}</tbody>
      </HTMLTable></div>}{!loading && !error && items.length === 0 && <div className="studio-empty"><h2>{section === 'components' && (reportQuery || reportNamespace || reportStatus) ? 'No matching components' : `No ${section === 'components' ? 'components' : section} yet`}</h2><span>{reportQuery || reportNamespace || reportStatus ? 'Adjust the search or filters to broaden the catalog.' : 'Create the versioned home for your first dynamic reader component.'}</span>{!(reportQuery || reportNamespace || reportStatus) && <Button intent="primary" icon="add" disabled={namespaceChanging} onClick={() => setReportDialogOpen(true)} style={{ marginTop: 16 }}>New component</Button>}</div>}</Card>
      {section === 'components' && !loading && (reportOffset > 0 || reportHasNext) && <div className="studio-catalog-pagination"><span>Showing {reportOffset + 1}–{reportOffset + reports.length}</span><ButtonGroup minimal><Button icon="chevron-left" disabled={reportOffset === 0} onClick={() => setReportOffset(Math.max(0, reportOffset - reportPageSize))}>Previous</Button><Button icon="chevron-right" disabled={!reportHasNext} onClick={() => setReportOffset(reportOffset + reportPageSize)}>Next</Button></ButtonGroup></div>}
    </main>}</div></div>
    <ReportDialog currentNamespace={namespaceWorkspace.current} api={api} isOpen={reportDialogOpen} onClose={() => setReportDialogOpen(false)} onCreated={onReportCreated} />
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

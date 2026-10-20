import nativeNavigation from './nativeNavigation.json' with {type:'json'};
import React, { useCallback, useState } from 'react';
import { ForgeThemeBoundary, ForgeThemeProvider } from 'forge/theme';
import { Button, Callout, HTMLSelect, InputGroup, Navbar, Spinner, Tag, Tree } from '@blueprintjs/core';
import { ManagementWindow } from './ManagementWindow.jsx';
import { managementSection } from './managementNavigation.js';
import { useNamespaceWorkspace } from './useNamespaceWorkspace.js';


const noExtensions = [];

// StudioApp is intentionally only a Forge composition. Its data boundary is
// StudioAPI, which maps one-for-one to the public Studio SDK operation names.
export function StudioApp({ api, mode, subject, brand = 'Data Studio', extensions = noExtensions, renderNamespaceSkills, onSignOut }) {
  const [skillEditing, setSkillEditing] = useState(false);
  const [section, setSection] = useState(() => managementSection(globalThis.location?.search, extensions));
  const [error, setError] = useState('');
  const [signingOut, setSigningOut] = useState(false);
  const [signOutError, setSignOutError] = useState('');
  const [navigationOpen, setNavigationOpen] = useState(() => typeof window === 'undefined' || !window.matchMedia?.('(max-width: 720px)').matches);
  const [navigationQuery, setNavigationQuery] = useState('');
  const [builderReport, setBuilderReport] = useState(null);
  const namespaceWorkspace = useNamespaceWorkspace(api, subject, skillEditing || section === 'builder' || section === 'security' || extensions.some((item) => item.id === section));
  const currentNamespaceId = namespaceWorkspace.current?.namespaceId;
  const onManagementOperation = useCallback((operation) => {
    if (['namespaces.create', 'namespaces.update', 'namespaces.delete'].includes(operation)) return namespaceWorkspace.refresh?.();
  }, [namespaceWorkspace.refresh]);
  const openBuilder = (report) => { setBuilderReport(report); setSection('builder'); };
  const navigate = (target, discardConfirmed = false) => {
    if (!target) return;
    if (skillEditing && target !== section && !discardConfirmed) { setError('Close the editor before leaving this workspace.'); return; }
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
      {namespaceWorkspace.enabled && <HTMLSelect className="studio-namespace-selector" aria-label="Current namespace" value={currentNamespaceId || ''} disabled={!namespaceWorkspace.ready || skillEditing} onChange={(event) => { namespaceWorkspace.select(event.target.value);  }}><option value="" disabled>{namespaceWorkspace.ready ? 'Choose namespace' : 'Loading namespaces…'}</option>{namespaceWorkspace.items.map((item) => <option key={item.namespaceId} value={item.namespaceId}>{namespaceWorkspace.items.filter((row) => (row.title || row.name) === (item.title || item.name)).length > 1 ? `${item.title || item.name} · ${item.name} · ${item.ownerId}` : item.title || item.name}</option>)}</HTMLSelect>}
    </Navbar.Group><Navbar.Group align="right"><Tag className="studio-identity-tag" minimal intent={mode === 'development' ? 'warning' : 'success'}>{mode === 'development' ? 'Development user' : subject || 'Authenticated user'}</Tag>{onSignOut && <Button className="studio-signout" title="Sign out" aria-label="Sign out" minimal icon="log-out" loading={signingOut} onClick={signOut}>Sign out</Button>}</Navbar.Group></Navbar>
    {signOutError && <Callout className="studio-signout-error" intent="danger" role="alert" title="Could not sign out">{signOutError}<Button small minimal intent="danger" onClick={signOut}>Try again</Button></Callout>}
    {skillEditing && error && <Callout intent="warning" role="alert">{error}</Callout>}
    {namespaceWorkspace.error && <Callout intent="warning" role="alert">{namespaceWorkspace.error}{!namespaceWorkspace.synchronizationError && <Button small minimal icon="refresh" disabled={!namespaceWorkspace.canRetry} onClick={namespaceWorkspace.refresh}>Retry namespaces</Button>}</Callout>}
    <div className="studio-app-body">
    {navigationOpen && <button type="button" className="studio-sidebar-scrim" aria-label="Close navigation" onClick={() => setNavigationOpen(false)}/>}
    {navigationOpen && <aside className="studio-sidebar"><InputGroup leftIcon="search" placeholder="Find Studio resource" value={navigationQuery} onChange={(event) => setNavigationQuery(event.target.value)} />
      <Tree className="studio-tree" contents={interactiveNavigation} onNodeClick={(node) => navigate(node.section)} />
    </aside>}
    {namespaceWorkspace.enabled && !currentNamespaceId && !['namespaces', 'connectors', 'schema'].includes(section) ? <main className="studio-main"><Callout title="Choose a namespace">Select an available namespace above, or create one in Namespaces.</Callout></main> : ['connectors', 'namespaces', 'overview', 'runtime', 'components', 'schema', 'skills', 'security', 'builder'].includes(section) ? <div className={section==='builder'?'studio-reader-host':'studio-workspace'} role={section==='builder'?undefined:'main'}><ManagementWindow key={`${section}:${currentNamespaceId||'unscoped'}`} api={api} disabled={namespaceWorkspace.enabled&&!namespaceWorkspace.ready} uri={`window://datastudio/${section==='components'?'component-catalog':section==='builder'?'reader-workbench':section}`} parameters={{currentNamespace:namespaceWorkspace.current,report:builderReport,hasNamespaceSkillsRenderer:Boolean(renderNamespaceSkills)}} renderSlots={renderNamespaceSkills?{'namespace-skills':props=>renderNamespaceSkills({api,namespace:namespaceWorkspace.current,subject,onDirtyChange:props.onDirtyChange})}:undefined} onDirtyChange={setSkillEditing} instanceKey={`native-datastudio-${section}`} onOperationCompleted={onManagementOperation} onNavigate={(target,parameters)=>target==='builder'?openBuilder(parameters.report):navigate(target,parameters?.discardConfirmed===true)}/></div> : extensions.some((extension) => extension.id === section) ? extensions.find((extension) => extension.id === section).render({ api, mode, subject, navigate, openComponent: openBuilder, currentNamespace: namespaceWorkspace.current, namespaceItems: namespaceWorkspace.items }) : null}</div></div>
  </ForgeThemeBoundary></ForgeThemeProvider>;
}

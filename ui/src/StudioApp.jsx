import nativeNavigation from './nativeNavigation.json' with {type:'json'};
import React, { useCallback, useId, useState } from 'react';
import { ForgeThemeBoundary, ForgeThemeProvider } from 'forge/theme';
import { Button, Callout, HTMLSelect, InputGroup, Navbar, Spinner, Tag, Tree } from '@blueprintjs/core';
import { WorkspaceRuntimeSurface, PersistentWindowContent } from 'forge/components';
import { ManagementWindow } from './ManagementWindow.jsx';
import { managementSection } from './managementNavigation.js';
import { useNamespaceWorkspace } from './useNamespaceWorkspace.js';


const noExtensions = [];
function WorkspaceContent({active,render}) {
  const [host,setHost] = useState(null);
  return <><div ref={setHost} className="studio-workspace-content"/><PersistentWindowContent host={active?host:null}>{render()}</PersistentWindowContent></>;
}
const nativeSections = ['audit', 'connectors', 'namespaces', 'overview', 'runtime', 'components', 'schema', 'skills', 'security', 'builder'];

// StudioApp is intentionally only a Forge composition. Its data boundary is
// StudioAPI, which maps one-for-one to the public Studio SDK operation names.
export function StudioApp({ api, mode, subject, brand = 'Data Studio', extensions = noExtensions, renderNamespaceSkills, onSignOut }) {
  const workspaceId = useId();
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
    <WorkspaceRuntimeSurface key={currentNamespaceId||'unscoped'} className="studio-workspace-runtime"
      tabProps={{visibility:'never'}}
      panelProps={{id:`studio-${workspaceId}`,entries:[...nativeSections.map(windowId=>({windowId,label:windowId,kind:'window'})),...extensions.map(extension=>({windowId:extension.id,label:extension.label,kind:'extension'}))],selectedId:section,className:'studio-workspace-panel',
        renderContent:(entry,{active})=><WorkspaceContent active={active} render={()=>{
          const target=entry.windowId;
          if(namespaceWorkspace.enabled&&!currentNamespaceId&&!['namespaces','connectors','schema'].includes(target))return <main className="studio-main"><Callout title="Choose a namespace">Select an available namespace above, or create one in Namespaces.</Callout></main>;
          if(!nativeSections.includes(target))return extensions.find(extension=>extension.id===target)?.render({api,mode,subject,navigate,openComponent:openBuilder,currentNamespace:namespaceWorkspace.current,namespaceItems:namespaceWorkspace.items});
          return <div className={target==='builder'?'studio-reader-host':'studio-management-host'} role={target==='builder'?undefined:'main'}>
            <ManagementWindow key={target==='builder'?`${builderReport?.id||''}:${builderReport?.versionNo||''}`:target} api={api} disabled={namespaceWorkspace.enabled&&!namespaceWorkspace.ready}
              uri={`window://datastudio/${target==='components'?'component-catalog':target==='builder'?'reader-workbench':target}`}
              parameters={{currentNamespace:namespaceWorkspace.current,report:builderReport,hasNamespaceSkillsRenderer:Boolean(renderNamespaceSkills)}}
              renderSlots={renderNamespaceSkills?{'namespace-skills':props=>renderNamespaceSkills({api,namespace:namespaceWorkspace.current,subject,onDirtyChange:props.onDirtyChange})}:undefined}
              onDirtyChange={value=>{if(target===section)setSkillEditing(value);}}
              instanceKey={`native-datastudio-${target}`} onOperationCompleted={onManagementOperation}
              onNavigate={(next,parameters)=>next==='builder'?openBuilder(parameters.report):navigate(next,parameters?.discardConfirmed===true)}/>
          </div>;
        }}/>}}/>
    </div></div>
  </ForgeThemeBoundary></ForgeThemeProvider>;
}

import React,{lazy,Suspense,useMemo,useRef} from 'react';
import {ResourceWindow} from 'forge/resource-window';
import 'forge/packs/blueprint';
import {managementWindowTransport} from './managementWindow.js';
import {downloadFile} from './hostDownload.js';
import {liveACLReviewURL} from './aclReviewURL.js';
const ExplorerWorkbench=lazy(()=>import('./ExplorerWorkbench.jsx'));

// The native host supplies only trusted admission/transport capabilities. The
// loader, contexts, window UI and nested-window renderer are owned by Forge.
export default function ManagementWindowRuntime({api,onOperationCompleted,renderSlots,...props}) {
 const callbacks=useRef({});callbacks.current={onOperationCompleted,renderSlots};
 const hostServices=useMemo(()=>({hostWindow:{
  loadResource:uri=>api.readManagementResource(uri),
  executeResource:(metadata,request)=>managementWindowTransport(api,metadata,(...args)=>callbacks.current.onOperationCompleted?.(...args))(request),
  download:downloadFile,
  reviewURL:resource=>liveACLReviewURL(resource,import.meta.env.BASE_URL),
  renderSlot:(name,slotProps)=>name==='explorer-sql'?<Suspense fallback={<div role="status">Loading SQL workspace…</div>}><ExplorerWorkbench {...slotProps}/></Suspense>:callbacks.current.renderSlots?.[name]?.(slotProps),
 }}),[api]);
 const parameters=props.uri?.endsWith('/schema')?{...props.parameters,hasSQLWorkbench:props.parameters?.hasSQLWorkbench!==false}:props.parameters;
 return <ResourceWindow {...props} parameters={parameters} hostServices={hostServices}/>;
}

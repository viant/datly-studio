import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup } from '@blueprintjs/core';

export function DownloadComponentButton({ api, report, versionNo, disabled = false }) {
  const [busy,setBusy]=useState(false), [error,setError]=useState('');
  const download=async()=>{
    setBusy(true);setError('');
    try {
      let selected=versionNo ?? report.currentDraftVersion;
      if(!selected){const page=await api.listVersions(report.id,{limit:1,orderBy:'version_no DESC'});selected=page.items?.[0]?.versionNo;}
      if(!selected)throw new Error('This component has no version to download.');
      const result=await api.downloadComponent(report.id,selected);
      const bytes=Uint8Array.from(atob(result.archive),character=>character.charCodeAt(0));
      const url=URL.createObjectURL(new Blob([bytes],{type:result.mediaType}));
      const link=document.createElement('a');link.href=url;link.download=result.filename;document.body.appendChild(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);
    }catch(cause){setError(cause.message);}finally{setBusy(false);}
  };
  const label=versionNo?`Download component v${versionNo}`:`Download ${report.title}`;
  return <><Button type="button" minimal icon="download" title={label} aria-label={label} loading={busy} disabled={disabled} onClick={download}/><Dialog isOpen={Boolean(error)} title="Download failed" onClose={()=>setError('')}><DialogBody><Callout intent="danger" role="alert">{error}</Callout></DialogBody><DialogFooter actions={<Button onClick={()=>setError('')}>Close</Button>}/></Dialog></>;
}

export function ImportComponentButton({api,onImported}) {
  const [open,setOpen]=useState(false),[reports,setReports]=useState([]),[target,setTarget]=useState(''),[file,setFile]=useState(null),[entry,setEntry]=useState(''),[busy,setBusy]=useState(false),[error,setError]=useState('');
  useEffect(()=>{if(!open)return;let cancelled=false;setError('');setFile(null);setEntry('');setTarget('');api.listReports({limit:500}).then(page=>{if(!cancelled)setReports(page.items??[]);}).catch(cause=>{if(!cancelled)setError(cause.message);});return()=>{cancelled=true;};},[api,open]);
  const format=file?.name.toLowerCase().endsWith('.zip')?'zip':/\.(tar\.gz|tgz)$/i.test(file?.name??'')?'tar.gz':file?.name.toLowerCase().endsWith('.tar')?'tar':null;
  const submit=async(event)=>{
    event.preventDefault();if(!file||!target)return;setBusy(true);setError('');
    try {
      if(file.size>16*1024*1024)throw new Error('Choose a file smaller than 16 MiB.');
      let result;
      if(format){const bytes=new Uint8Array(await file.arrayBuffer());let binary='';for(let index=0;index<bytes.length;index+=32768)binary+=String.fromCharCode(...bytes.subarray(index,index+32768));result=await api.loadArchive(target,{format,archive:btoa(binary),entryDql:entry.trim()});}
      else {if(!file.name.toLowerCase().endsWith('.dql'))throw new Error('Choose a DQL, ZIP, TAR, or TAR.GZ file.');result=await api.loadDQL(target,{dql:await file.text()});}
      setOpen(false);onImported?.({...reports.find(item=>item.id===target),versionNo:result.version.versionNo,currentDraftVersion:result.version.versionNo});
    }catch(cause){setError(cause.message);}finally{setBusy(false);}
  };
  return <><Button type="button" icon="import" title="Import component" aria-label="Import component" onClick={()=>setOpen(true)}/><Dialog isOpen={open} title="Import component" icon="import" onClose={()=>!busy&&setOpen(false)} canEscapeKeyClose={!busy} canOutsideClickClose={!busy}><form onSubmit={submit}><DialogBody>
    {error&&<Callout intent="danger" role="alert">{error}</Callout>}
    <FormGroup label="Component" labelFor="import-target" helperText="Creates a new draft in the selected component. Create a component first if needed."><HTMLSelect id="import-target" fill value={target} disabled={busy} onChange={event=>setTarget(event.target.value)}><option value="">Select component</option>{reports.map(report=><option key={report.id} value={report.id}>{report.title}</option>)}</HTMLSelect></FormGroup>
    <FormGroup label="DQL or archive" labelFor="import-file"><input id="import-file" type="file" accept=".dql,.zip,.tar,.tar.gz,.tgz" disabled={busy} onChange={event=>{setFile(event.target.files?.[0]??null);setError('');}}/></FormGroup>
    {format&&<FormGroup label="Root DQL file" labelFor="import-entry" helperText="Optional for a single root DQL. For multiple entries, enter the root filename to import; all dependencies are retained."><InputGroup id="import-entry" placeholder="component.dql" value={entry} disabled={busy} onChange={event=>setEntry(event.target.value)}/></FormGroup>}
  </DialogBody><DialogFooter actions={<><Button disabled={busy} onClick={()=>setOpen(false)}>Cancel</Button><Button type="submit" icon="import" intent="primary" title="Import as new draft" aria-label="Import as new draft" disabled={!file||!target} loading={busy}/></>}/></form></Dialog></>;
}

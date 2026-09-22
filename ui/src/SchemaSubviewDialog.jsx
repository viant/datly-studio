import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Spinner } from '@blueprintjs/core';

export function SchemaSubviewDialog({ api, isOpen, connector, table, sql, onClose, onApplied }) {
  const [reports, setReports] = useState([]);
  const [reportId, setReportId] = useState('');
  const [inspection, setInspection] = useState(null);
  const [draft, setDraft] = useState({ name: '', parent: '', on: '' });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const report = useMemo(() => reports.find((item) => item.id === reportId), [reports, reportId]);
  const views = inspection?.structure?.views ?? [];
  useEffect(() => {
    if (!isOpen) return;
    setReports([]); setReportId(''); setInspection(null); setError(''); setSaving(false);
    setDraft({ name: identifier(table?.name), parent: '', on: '' });
    setLoading(true);
    api.listReports({ connectorName: connector, limit: 100 }).then((page) => {
      const items=page?.items??[]; setReports(items); setReportId(items[0]?.id??'');
    }).catch((cause)=>setError(cause.message)).finally(()=>setLoading(false));
  }, [api,isOpen,connector,table?.name]);
  useEffect(() => {
    if (!reportId) return;
    let cancelled=false; setLoading(true); setInspection(null);
    api.listVersions(reportId,{limit:1}).then(async(page)=>{
      const version=page?.items?.[0]; if(!version) throw new Error('Selected reader has no version.');
      const value=await api.inspectVersion(reportId,version.versionNo); if(cancelled)return;
      setInspection(value);
      const parent=value?.structure?.views?.[0]?.name??'';
      const child=suggestViewName(table,value?.structure,parent);
      setDraft((current)=>({...current,name:child,parent,on:suggestRelation(child,parent,table,value?.structure)}));
    }).catch((cause)=>!cancelled&&setError(cause.message)).finally(()=>!cancelled&&setLoading(false));
    return()=>{cancelled=true;};
  },[api,reportId,table?.name]);
  const update=(name)=>(event)=>setDraft((current)=>({...current,[name]:event.target.value}));
  const submit=async(event)=>{
    event.preventDefault();
    if(!inspection?.version||!/^[a-z][a-z0-9_]*$/.test(draft.name)||!draft.parent||!draft.on.trim()||!sql.trim()){setError('Select a component and enter a child identifier, parent, and complete relation keys.');return;}
    setSaving(true);setError('');
    try{
      const result=await api.applyReaderCommand(reportId,inspection.version.versionNo,{expectedSourceRevision:inspection.version.sourceRevision,operation:{type:'addView',view:{name:draft.name,kind:'subview',parent:draft.parent,join:'JOIN',sql:sql.trim(),on:draft.on.trim()}}});
      if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.[0]?.message||'Datly rejected the related view.');
      await onApplied(report);onClose();
    }catch(cause){setError(cause.message);}finally{setSaving(false);}
  };
  return <Dialog className="studio-connector-dialog studio-subview-dialog" isOpen={isOpen} onClose={onClose} title="Add table to component" icon="git-branch" canOutsideClickClose={!saving}>
    <form onSubmit={submit}><DialogBody><p className="studio-dialog-lead">Attach the selected SQL as a compiled Datly subview. Complete key mappings are required.</p>{error&&<Callout intent="danger" role="alert">{error}</Callout>}
      <FormGroup label="Component" labelFor="schema-subview-reader" required>{loading&&reports.length===0?<Spinner size={20}/>:<HTMLSelect id="schema-subview-reader" value={reportId} onChange={(event)=>{setError('');setReportId(event.target.value);}} fill disabled={saving}><option value="">Select component</option>{reports.map((item)=><option key={item.id} value={item.id}>{item.title}</option>)}</HTMLSelect>}</FormGroup>
      <div className="studio-form-grid"><FormGroup label="Child view identifier" labelFor="schema-subview-name" required><InputGroup id="schema-subview-name" value={draft.name} onChange={update('name')} disabled={saving}/></FormGroup><FormGroup label="Parent view" labelFor="schema-subview-parent" required><HTMLSelect id="schema-subview-parent" value={draft.parent} onChange={update('parent')} fill disabled={saving||loading}>{views.map((view)=><option key={view.name} value={view.name}>{view.name}</option>)}</HTMLSelect></FormGroup></div>
      <FormGroup label="Relation keys" labelFor="schema-subview-on" helperText="For example domains.VENDOR_ID=vendor.ID. Verify every composite key part." required><InputGroup id="schema-subview-on" value={draft.on} onChange={update('on')} disabled={saving} /></FormGroup>
    </DialogBody><DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving} disabled={loading||!inspection}>Compile and open</Button></>}/></form>
  </Dialog>;
}

function identifier(value){const result=String(value??'').toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_+|_+$/g,'').replace(/^[^a-z]+/,'');return result||'view';}
function suggestViewName(table,structure,parent){const root=structure?.component?.rootView;const parentTable=identifier(terminalName(findView(root,parent)?.source?.table));let child=identifier(table?.name);if(parentTable&&child.startsWith(`${parentTable}_`))child=child.slice(parentTable.length+1);if(child.startsWith('ci_'))child=child.slice(3);if(!child.endsWith('s'))child=child.endsWith('y')?`${child.slice(0,-1)}ies`:`${child}s`;return child||'items';}
function suggestRelation(child,parent,table,structure){const root=structure?.component?.rootView;const parentTable=terminalName(findView(root,parent)?.source?.table);const reference=(table?.columns??[]).find((column)=>column.referenceTable&&(!parentTable||terminalName(column.referenceTable)===parentTable));return reference?`${child}.${reference.name}=${parent}.${reference.referenceColumn}`:'';}
function findView(view,name){if(!view)return null;if(view.name===name||view.namespace===name)return view;for(const relation of view.relations??[]){const found=findView(relation.view,name);if(found)return found;}return null;}
function terminalName(value){const parts=String(value??'').replaceAll('`','').toLowerCase().split('.');return parts[parts.length-1];}

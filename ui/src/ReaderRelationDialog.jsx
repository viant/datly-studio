import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup } from '@blueprintjs/core';

export function ReaderRelationDialog({ isOpen, node, structure, onClose, onApply }) {
  const views=useMemo(()=>(structure?.views??[]).map((view)=>view.name).filter((name)=>name&&name!==node?.child?.name),[structure,node?.child?.name]);
  const [parent,setParent]=useState('');
  const [on,setOn]=useState('');
  const [saving,setSaving]=useState(false);
  const [error,setError]=useState('');
  useEffect(()=>{if(!isOpen)return;setParent(node?.parentName||views[0]||'');setOn(relationText(node?.relation));setSaving(false);setError('');},[isOpen,node,views.join('|')]);
  const submit=async(event)=>{event.preventDefault();if(!node?.child?.name||!parent||!on.trim()){setError('Select a parent and provide every parent/child equality key.');return;}setSaving(true);setError('');try{await onApply({type:'updateRelation',relation:{name:node.child.name,parent,on:on.trim()}});onClose();}catch(cause){setError(cause.message);}finally{setSaving(false);}};
  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title={`Modify relation · ${node?.child?.label||''}`} icon="diagram-tree" canOutsideClickClose={!saving}><form onSubmit={submit}><DialogBody><p className="studio-dialog-lead">Datly patches only this relation’s ON span, recompiles the graph, and verifies that the child resolves under the selected parent.</p>{error&&<Callout intent="danger" role="alert">{error}</Callout>}<FormGroup label="Parent view" labelFor="relation-parent" required><HTMLSelect id="relation-parent" value={parent} onChange={(event)=>setParent(event.target.value)} fill disabled={saving}>{views.map((name)=><option key={name} value={name}>{name}</option>)}</HTMLSelect></FormGroup><FormGroup label="Relation keys" labelFor="relation-on" helperText="Use all equality parts for composite identity; changing the parent does not alter child SQL." required><InputGroup id="relation-on" value={on} onChange={(event)=>setOn(event.target.value)} disabled={saving}/></FormGroup>{parent!==node?.parentName&&<Callout intent="warning" compact>Re-parenting changes response nesting. Datly will reject cycles, missing parents, and ambiguous key ownership.</Callout>}</DialogBody><DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>Apply relation</Button></>}/></form></Dialog>;
}

function relationText(relation){return (relation?.on??[]).map((item)=>`${item.childNamespace}.${item.childColumn}=${item.parentNamespace}.${item.parentColumn}`).join(' AND ');}

import React, { useState } from 'react';
import { Button, Callout, Card, FormGroup, HTMLSelect, InputGroup } from '@blueprintjs/core';

export function ReaderOutputDialog({ structure, initialName = '', onClose, onApply }) {
  const outputs = (structure?.declarations ?? []).map(({parameter})=>parameter).filter((item)=>item?.source?.kind === 'output');
  const initial = outputs.find((item)=>item.name === initialName);
  const [name,setName] = useState(initial?.name || '');
  const [type,setType] = useState(initial?.typeExpr || outputs.find((item)=>item.source.name === 'view')?.typeExpr || '');
  const [source,setSource] = useState(initial?.source?.name || 'view');
  const sources = [...new Set(['view','body','status','metrics','derived','summary',initial?.source?.name].filter(Boolean))];
  const changeSource = (event) => {
    const next=event.target.value;setSource(next);
    if(!initial) setType(next==='metrics'?'github.com/viant/xdatly/response.Metrics':next==='status'||next==='body'?'string':outputs.find((item)=>item.source.name==='view')?.typeExpr || '');
  };
  const [saving,setSaving] = useState(false);
  const [error,setError] = useState('');
  const [removing,setRemoving] = useState(false);
  const [confirmation,setConfirmation] = useState('');
  const apply = async (operation) => {
    setSaving(true);setError('');
    try { await onApply(operation);onClose(); }
    catch(cause) { setError(cause.message); }
    finally { setSaving(false); }
  };
  const submit = (event) => {
    event.preventDefault();
    if(!/^[A-Za-z][A-Za-z0-9_]*$/.test(name) || !type.trim() || !source.trim()) {setError('Enter an output name, type, and kind source.');return;}
    return apply({type:initial?'updateOutputField':'addField',field:{...(initial?{existingName:initial.name}:{}),name,type:type.trim(),sourceKind:'output',sourceName:source.trim()}});
  };
  return <Card className="studio-card studio-columns-panel studio-inline-contract-form" elevation={0}>
    <form onSubmit={submit}>
      <div className="studio-panel-heading"><div><h2>{initial?'Edit output':'Add output'}</h2><p className="studio-muted">Bind a typed response parameter. Use output/view for the root reader result.</p></div>{initial&&<Button type="button" icon="trash" intent="danger" disabled={saving} onClick={()=>setRemoving(true)}>Remove output</Button>}</div>
      {error&&<Callout intent="danger" role="alert">{error}</Callout>}
      <div className="studio-form-grid studio-binding-grid"><FormGroup label="Output name" labelFor="output-name" required><InputGroup id="output-name" value={name} onChange={(event)=>setName(event.target.value)} disabled={saving} autoFocus/></FormGroup><FormGroup label="Type expression" labelFor="output-type" required><InputGroup id="output-type" value={type} onChange={(event)=>setType(event.target.value)} disabled={saving} placeholder="[]*Record"/></FormGroup></div>
      <div className="studio-form-grid studio-binding-grid"><FormGroup label="Kind" labelFor="output-kind"><HTMLSelect id="output-kind" value="output" fill disabled={saving}><option value="output">output</option></HTMLSelect></FormGroup><FormGroup label="Kind source" labelFor="output-source" helperText="view returns root rows; status and metrics expose execution details. Derived outputs require a query definition." required><HTMLSelect id="output-source" value={source} onChange={changeSource} fill disabled={saving}>{sources.map((item)=><option key={item} value={item}>{item}</option>)}</HTMLSelect></FormGroup></div>
      {removing&&<Callout intent="danger" title={`Remove ${initial.name}?`}><p>Type the output name to remove its declaration. The reader must still compile.</p><InputGroup aria-label="Confirm output name" value={confirmation} onChange={(event)=>setConfirmation(event.target.value)} disabled={saving}/><Button type="button" intent="danger" disabled={saving||confirmation!==initial.name} onClick={()=>apply({type:'removeOutputField',field:{existingName:initial.name}})}>Confirm removal</Button></Callout>}
      <div className="studio-schema-actions"><Button type="submit" intent="primary" loading={saving}>{initial?'Save output':'Add output'}</Button><Button type="button" onClick={onClose} disabled={saving}>Back to outputs</Button></div>
    </form>
  </Card>;
}

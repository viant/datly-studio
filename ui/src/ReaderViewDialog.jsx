import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';

export function ReaderViewDialog({ isOpen, view, sql, onClose, onApply }) {
  const [source,setSource]=useState(''); const [saving,setSaving]=useState(false); const [error,setError]=useState('');
  useEffect(()=>{if(isOpen){setSource(sql||'');setSaving(false);setError('');}},[isOpen,sql]);
  const submit=async(event)=>{event.preventDefault();if(!view||!source.trim()){setError('View SQL is required.');return;}setSaving(true);setError('');try{await onApply({type:'updateView',view:{name:view,sql:source.trim()}});onClose();}catch(cause){setError(cause.message);}finally{setSaving(false);}};
  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title={`Modify view · ${view||''}`} icon="edit" canOutsideClickClose={!saving}><form onSubmit={submit}><DialogBody><p className="studio-dialog-lead">Datly replaces only this view’s source span and recompiles the complete graph.</p>{error&&<Callout intent="danger" role="alert">{error}</Callout>}<div className="studio-code-editor"><Editor ariaLabel={`${view||'View'} SQL source`} value={source} onChange={setSource} language="sql" height="300px" readOnly={saving}/></div></DialogBody><DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>Apply SQL</Button></>}/></form></Dialog>;
}

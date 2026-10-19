import React,{useEffect,useState} from 'react';
import {Button,Callout,FormGroup,HTMLSelect,InputGroup} from '@blueprintjs/core';
import {fetchReaderDataSource} from './readerDataSources.js';

export function ReaderCodecFields({api,draft,onChange,disabled}) {
 const [items,setItems]=useState([]),[loading,setLoading]=useState(false),[error,setError]=useState(''),[attempt,setAttempt]=useState(0);
 useEffect(()=>{if(!api?.fetchMCPDataSource)return;let active=true;setLoading(true);setError('');fetchReaderDataSource(api,'codecs').then(items=>{if(active)setItems(items);}).catch(cause=>{if(active)setError(cause.message);}).finally(()=>{if(active)setLoading(false);});return()=>{active=false;};},[api,attempt]);
 const selected=items.find(item=>item.name===draft.codecName);
 const choose=event=>{const name=event.target.value;const codec=items.find(item=>item.name===name);onChange(current=>({...current,codecName:name,codecArgs:'',...(codec?.inputType&&codec?.outputType?{type:`${codec.inputType},${codec.outputType}`} : {}),...(name==='JwtClaim'?{sourceKind:'header',sourceName:'Authorization',required:true,errorStatusCode:401,errorMessage:'Unauthorized'}:{})}));};
 return <>
 {error&&<Callout intent="warning" title="Codec discovery unavailable">{error}<Button small onClick={()=>setAttempt(value=>value+1)}>Retry codecs</Button></Callout>}
 <div className="studio-form-grid"><FormGroup label="Codec" labelFor="parameter-codec" helperText={selected?.description||'Choose a codec available on this host.'}><HTMLSelect id="parameter-codec" value={draft.codecName} onChange={choose} fill disabled={disabled||loading||Boolean(error)}><option value="">{loading?'Loading codecs…':'No codec'}</option>{draft.codecName&&!selected&&<option value={draft.codecName}>{draft.codecName} (current)</option>}{items.map(item=><option key={item.name} value={item.name}>{item.name}</option>)}</HTMLSelect></FormGroup><FormGroup label="Codec arguments" labelFor="parameter-codec-args" helperText={selected?.acceptsArguments===false?'This codec accepts no arguments.':'Ordered comma-separated values.'}><InputGroup id="parameter-codec-args" value={draft.codecArgs} onChange={event=>onChange(current=>({...current,codecArgs:event.target.value}))} disabled={disabled||!draft.codecName||selected?.acceptsArguments===false}/></FormGroup></div>
 {draft.codecName==='JwtClaim'&&<Callout compact icon="lock" title="Verified JWT">Binds the Authorization header to verified claims using the host’s JWT verifier. Keep this input required to reject missing credentials.</Callout>}
 </>;
}

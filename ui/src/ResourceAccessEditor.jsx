import React, {useMemo, useState, useCallback} from 'react';
import {ResourceAccessEditor as SharedResourceAccessEditor, NamedPolicyManager} from 'forge/resource-policy-editor';

export function ResourceAccessEditor(props) {
 const [generation,setGeneration] = useState(0), [editorState,setEditorState] = useState({dirty:false,saving:false});
 const changed=useCallback(value=>{setEditorState(value);props.onStateChange?.(value);},[props.onStateChange]);
 const api = useMemo(() => {
  if (typeof props.api?.listNamedPolicies !== 'function') return props.api;
  return Object.assign(Object.create(props.api), {getResourceAccessContext: async resource => {
   const [context, namedPolicies] = await Promise.all([props.api.getResourceAccessContext(resource), props.api.listNamedPolicies().catch(() => null)]);
   return {...context, namedPolicies: namedPolicies || [], namedPoliciesUnavailable: namedPolicies === null};
  }});
 }, [props.api,generation]);
 return <>{typeof props.api?.listNamedPolicies === 'function' && <NamedPolicyManager api={props.api} disabled={props.readOnly||editorState.dirty||editorState.saving} onChange={()=>setGeneration(value=>value+1)}/>}<SharedResourceAccessEditor publicActionNames={['discover','describe','execute','retrieve']} relatedActions={['discover','describe','execute']} applyGroupLabel="Apply to exposure and execution" {...props} api={api} onStateChange={changed}/></>;
}

(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const draft=()=>ds('output_draft').getFormData()||{};
 const set=values=>ds('output_draft').setFormData({values:{...draft(),...values}});
 const host=()=>context.services.hostWindow;
 const params=()=>host().parameters;
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(ref,parameters)=>record(await ds(ref).fetchRecords({parameters}));
 let busy=false;
 const run=async action=>{if(busy)return false;busy=true;host().setBusy?.(true);set({saving:true,outcome:''});try{await action();return true;}catch(error){set({outcome:error.message});return false;}finally{busy=false;set({saving:false});host().setBusy?.(false);}};
 const apply=async operation=>{
  if(!draft().canEdit)throw new Error('Create an editable draft before changing this output.');
  const result=await call('output_apply',{reportId:params().reportId,versionNo:params().versionNo,command:{expectedSourceRevision:draft().sourceRevision,operation}});
  if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.map(item=>item.message).join('\n')||'Datly rejected the output change.');
  await host().complete({result,close:true});
 };
 return {
  initialize:async()=>{set({loading:true});try{
   const value=await call('output_inspect',{reportId:params().reportId,versionNo:params().versionNo,discoverColumns:false});
   const p=(value.structure?.declarations||[]).find(item=>item.parameter?.name===params().outputName&&item.parameter.source?.kind==='output')?.parameter;
   if(params().outputName&&!p)throw new Error('The output no longer exists. Close this editor and refresh the component.');
   ds('output_draft').setFormData({values:{loading:false,saving:false,editing:Boolean(p),existingName:p?.name||'',canEdit:value.capabilities?.canEdit!==false&&value.version.state==='draft',sourceRevision:value.version.sourceRevision,name:p?.name||'',type:p?.typeExpr||'string',kind:'output',source:p?.source?.name||'body',confirmation:'',outcome:''}});
  }catch(error){set({loading:false,canEdit:false,outcome:error.message});}},
  close:()=>draft().saving?false:host().close(),
  changeSource:()=>{const source=draft().source;if(source==='metrics')set({type:'github.com/viant/xdatly/response.Metrics'});else if(source==='status')set({type:'string'});},
  save:()=>run(async()=>{const p=draft();if(!/^[A-Za-z][A-Za-z0-9_]*$/.test(p.name||''))throw new Error('Enter an output name.');if(!p.editing&&['derived','summary'].includes(p.source))throw new Error('This source requires an existing query definition.');await apply({type:p.editing?'updateOutputField':'addField',field:{existingName:p.editing?p.existingName:undefined,name:p.name,type:p.type,sourceKind:'output',sourceName:p.source}});}),
  confirmRemove:()=>{set({confirmation:''});context.handlers.window.openDialog({execution:{args:['remove-output']}});},
  remove:()=>run(async()=>{if(draft().confirmation!==draft().existingName)throw new Error('Type the exact output name to confirm removal.');await apply({type:'removeOutputField',field:{existingName:draft().existingName}});})
 };
})()

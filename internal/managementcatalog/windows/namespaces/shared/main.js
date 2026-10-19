(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const draft=()=>ds('namespace_draft');
 const call=(operation,parameters)=>ds('namespace_'+operation).fetchRecords({parameters});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 let busy=false;
 const run=async fn=>{if(busy)return false;busy=true;try{await fn();return true;}catch(error){draft().setFormField({item:{dataField:'outcome'},value:error.message});return false;}finally{busy=false;}};
 const refresh=()=>ds('namespaces').fetchCollection();
 const open=values=>{draft().setFormData({values});context.handlers.window.openDialog({execution:{args:['namespace-editor']}});};
 const edit=current=>open({...current,editing:true,allowedRoles:JSON.stringify(current.allowedRoles||[]),mcpPort:current.mcpPort??0,outcome:''});
 return {
  newNamespace:()=>open({name:'',title:'',description:'',visibility:'private',allowedRoles:'[]',mcpEnabled:false,mcpPort:0,editing:false,status:'active',outcome:''}),
  editNamespace:()=>run(async()=>{const row=ds('namespaces').peekSelection()?.selected;if(!row)throw new Error('Select a namespace.');edit(record(await call('get',{name:row.name})));}),
  saveNamespace:()=>run(async()=>{
   const form=draft().getFormData();
   const input={title:form.title,description:form.description||'',visibility:form.visibility,allowedRoles:JSON.parse(form.allowedRoles||'[]'),mcpEnabled:Boolean(form.mcpEnabled),mcpPort:Number(form.mcpPort)};
   if(form.editing)await call('update',{name:form.name,input:{...input,status:form.status,etag:form.etag}});
   else await call('create',{name:form.name,...input});
   refresh();
  }),
  reloadNamespace:()=>run(async()=>{const form=draft().getFormData();edit(record(await call('get',{name:form.name})));}),
  refreshNamespaces:refresh
 };
})()

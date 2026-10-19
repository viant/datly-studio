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
   if(form.canManage===false)throw new Error('This namespace is read-only.');
   const name=(form.name||'').trim(),title=(form.title||'').trim();
   if((!form.editing&&!/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(name))||!title)throw new Error('Enter a title and a lowercase dot-separated namespace such as inventory or delivery.forecasting.');
   const roles=JSON.parse(form.allowedRoles||'[]');
   if(!Array.isArray(roles)||roles.some(role=>typeof role!=='string'))throw new Error('Viewer roles must be a JSON array of strings.');
   const port=Number(form.mcpPort);
   if(form.mcpEnabled&&(!Number.isInteger(port)||port<0||port>65535))throw new Error('Enter an MCP port from 0 to 65535.');
   const input={title,description:(form.description||'').trim(),visibility:form.visibility,allowedRoles:[...new Set(roles.map(role=>role.trim()).filter(Boolean))],mcpEnabled:Boolean(form.mcpEnabled),mcpPort:Number.isInteger(port)&&port>=0&&port<=65535?port:0};
   if(form.editing)await call('update',{name,input:{...input,status:form.status,etag:form.etag}});
   else await call('create',{name,...input});
   refresh();
  }),
  reloadNamespace:()=>run(async()=>{const form=draft().getFormData();edit(record(await call('get',{name:form.name})));}),
  confirmDeleteNamespace:()=>run(async()=>{
   const row=ds('namespaces').peekSelection()?.selected;if(!row)throw new Error('Select a namespace.');
   if(row.canManage===false)throw new Error('This namespace is read-only.');
   draft().setFormData({values:{...row,outcome:'Deletion is blocked while a component belongs to this namespace.'}});
   context.handlers.window.openDialog({execution:{args:['namespace-delete']}});
  }),
  deleteNamespace:()=>run(async()=>{const form=draft().getFormData();await call('delete',{name:form.name,etag:form.etag});await refresh();}),
  searchNamespaces:()=>run(async()=>{const filter=ds('namespace_filter').getFormData();ds('namespaces').setInputParameters({query:(filter.query||'').trim(),status:filter.status||'',offset:0});await refresh();}),
  refreshNamespaces:refresh
 };
})()

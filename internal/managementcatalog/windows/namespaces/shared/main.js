(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const draft=()=>ds('namespace_draft');
 const call=(operation,parameters)=>ds('namespace_'+operation).fetchRecords({parameters});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 let busy=false;
 const run=async fn=>{if(busy)return false;busy=true;draft().setFormField({item:{dataField:"pending"},value:true});try{await fn();return true;}catch(error){draft().setFormField({item:{dataField:'outcome'},value:error.message});return false;}finally{busy=false;draft().setFormField({item:{dataField:"pending"},value:false});}};
 const refresh=()=>ds('namespaces').fetchCollection();
 const open=values=>{draft().setFormData({values});context.handlers.window.openDialog({execution:{args:['namespace-editor']}});};
 const roleRows=values=>(values||[]).map(role=>({role}));
 const roleValues=(rows,label)=>{
  if(!Array.isArray(rows)||rows.some(row=>typeof row?.role!=='string'||!row.role.trim()))throw new Error(`Enter each ${label} role or remove its empty row.`);
  return [...new Set(rows.map(row=>row.role.trim()))];
 };
 const edit=current=>open({...current,editing:true,viewerRoleRows:roleRows(current.allowedRoles),managerRoleRows:roleRows(current.managerRoles),mcpPort:current.mcpPort??0,outcome:''});
 return {
  initialize:()=>run(async()=>{const permissions=record(await ds("namespace_permissions").fetchRecords({parameters:{limit:1}}));draft().setFormField({item:{dataField:"canCreate"},value:permissions?.canCreate===true});}),
  newNamespace:()=>{if(draft().getFormData()?.canCreate!==true)return false;return open({name:'',title:'',description:'',visibility:'private',viewerRoleRows:[],managerRoleRows:[],mcpEnabled:false,mcpPort:0,editing:false,status:'active',outcome:''});},
  editNamespace:()=>run(async()=>{const row=ds('namespaces').peekSelection()?.selected;if(!row)throw new Error('Select a namespace.');edit(record(await call('get',{name:row.name})));}),
  editNamespacePolicies:()=>run(async()=>{
   const row=ds('namespaces').peekSelection()?.selected;if(!row)throw new Error('Select a namespace.');if(row.canManage===false)throw new Error('This namespace is read-only.');
   const result=record(await call('settings_get',{name:row.name})),security=result.settings?.security,limits=result.settings?.resourceLimits||{};
   draft().setFormData({values:{...row,etag:result.etag,inheritPredicatePackages:security?.predicatePackages==null,predicatePackages:(security?.predicatePackages||[]).join('\n'),requireSecurityPredicate:security?.requireSecurityPredicate===true,maxBytesProcessed:limits.maxBytesProcessed??'',maxReferencedTables:limits.maxReferencedTables??'',maxReturnedRows:limits.maxReturnedRows??'',outcome:''}});
   context.handlers.window.openDialog({execution:{args:['namespace-policies']}});
  }),
  saveNamespacePolicies:()=>run(async()=>{
   const form=draft().getFormData();if(form.canManage===false)throw new Error('This namespace is read-only.');
   const threshold=(value,label)=>{if(value==null||value==='')return undefined;const number=Number(value);if(!Number.isSafeInteger(number)||number<0)throw new Error(label+' must be a nonnegative whole number.');return number;};
   const settings={security:{predicatePackages:form.inheritPredicatePackages?null:[...new Set((form.predicatePackages||'').split(/\s+/).filter(Boolean))],requireSecurityPredicate:Boolean(form.requireSecurityPredicate)},resourceLimits:{maxBytesProcessed:threshold(form.maxBytesProcessed,'Bytes processed'),maxReferencedTables:threshold(form.maxReferencedTables,'Referenced tables'),maxReturnedRows:threshold(form.maxReturnedRows,'Returned rows')}};
   await call('settings_update',{name:form.name,input:{etag:form.etag,settings}});await refresh();
  }),
  saveNamespace:()=>run(async()=>{
   const form=draft().getFormData();
   if(form.canManage===false)throw new Error('This namespace is read-only.');
   const name=(form.name||'').trim(),title=(form.title||'').trim();
   if((!form.editing&&!/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(name))||!title)throw new Error('Enter a title and a lowercase dot-separated namespace such as inventory or delivery.forecasting.');
   const roles=roleValues(form.viewerRoleRows||[],'viewer');
   const managers=roleValues(form.managerRoleRows||[],'manager');
   if(managers.includes('*'))throw new Error('Enter a specific manager role; * is not allowed.');
   const port=Number(form.mcpPort);
   if(form.mcpEnabled&&(!Number.isInteger(port)||port<0||port>65535))throw new Error('Enter an MCP port from 0 to 65535.');
   const input={title,description:(form.description||'').trim(),visibility:form.visibility,allowedRoles:roles,managerRoles:managers,mcpEnabled:Boolean(form.mcpEnabled),mcpPort:Number.isInteger(port)&&port>=0&&port<=65535?port:0};
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
 };
})()

(() => {
 const rules = connectorBehavior();
 const ds = id => context.Context(id).handlers.dataSource;
 const draft = () => ds('connector_draft');
 const call = (operation, parameters) => ds('connector_' + operation).fetchRecords({parameters});
 const selected = () => ds('connectors').peekSelection()?.selected;
 const record = result => {const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 let busy = false;
 const run = async fn => {
  if (busy) return false;
  busy = true;
  try { await fn(); return true; }
  catch (error) { draft().setFormField({item:{dataField:'outcome'},value:error.message}); return false; }
  finally { busy = false; }
 };
 const refresh = () => ds('connectors').fetchCollection();
 const open = values => {
  draft().setFormData({values});
  context.handlers.window.openDialog({execution:{args:['connector-editor']}});
 };
 const reload = async name => record(await call('get',{name}));
 return {
  newConnector: () => open({name:'',driver:'mysql',dsnTemplate:'',secretURL:'',secretKey:'',description:'',options:'{}',editing:false,outcome:''}),
  editConnector: () => run(async () => {
   const row = selected(); if (!row) throw new Error('Select a connector.');
   const current = await reload(row.name);
   open({...current,editing:true,dsnTemplate:'',secretURL:'',secretKey:'',clearSecret:false,options:JSON.stringify(current.options||{},null,2),outcome:'Blank connection fields keep the saved configuration. Use Remove secret to clear its reference.'});
  }),
  applyYAML: () => run(async () => {
   const form=draft().getFormData(), value=utilities.yaml.parse(form.yamlText||'');
   const patch={...form,...rules.yamlPatch(value)};
   draft().setFormData({values:patch});
  }),
  saveConnector: () => run(async () => {
   const form=draft().getFormData();
   const {operation,parameters}=rules.saveRequest(form);
   await call(operation,parameters);
   refresh();
  }),
  testConnector: () => run(async () => {
   const row=selected();if(!row)throw new Error('Select a connector.');
   const result=record(await call('test',{name:row.name}));
   draft().setFormField({item:{dataField:'outcome'},value:result.message||result.status||'Probe completed.'});refresh();
  }),
  activateConnector: () => run(async () => {
   const row=selected();if(!row)throw new Error('Select a connector.');
   const current=await reload(row.name);await call('activate',{name:row.name,etag:current.etag});refresh();
  }),
  reloadConnector: () => run(async () => {const form=draft().getFormData();const current=await reload(form.name);draft().setFormData({values:{...current,editing:true,dsnTemplate:'',secretURL:'',secretKey:'',options:JSON.stringify(current.options||{},null,2),outcome:'Reloaded current revision.'}});}),
  refreshConnectors: refresh
 };
})()

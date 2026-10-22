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
 const showSaved = (current, outcome) => draft().setFormData({values:{...current,editing:true,dsnTemplate:'',secretURL:'',secretKey:'',clearSecret:false,...rules.discoveryDraft(current.options),discoveryConfigured:Boolean(current.options?.discovery),outcome}});
 const probeOutcome = result => [result.status==='passed'?'Connection passed.':`Connection failed${result.errorCode?` [${result.errorCode}]`:''}.`,result.message,result.testedAt?`Tested at ${result.testedAt}`:''].filter(Boolean).join('\n');
 return {
  refreshConnectors:()=>run(refresh),
  connectorStatus: props => props?.row?.status,
  connectorState: props => {const active=props?.row?.status==='active';return {text:active?'On':'Off',intent:active?'success':'danger',minimal:false,title:active?'On — click to disable':'Off — click to activate'};},
  toggleConnector: props => run(async () => {
   const row=props?.row||selected();if(!row)throw new Error('Select a connector.');
   await call(row.status==='active'?'disable':'activate',{name:row.name,etag:row.etag});await refresh();
  }),
  newConnector: () => open({name:'',driver:'mysql',dsnTemplate:'',secretURL:'',secretKey:'',description:'',options:'{}',catalogRows:[],schemaRows:[],editing:false,outcome:''}),
  editConnector: props => run(async () => {
   const row = props?.row||selected(); if (!row) throw new Error('Select a connector.');
   const current = await reload(row.name);
   open({...current,editing:true,dsnTemplate:'',secretURL:'',secretKey:'',clearSecret:false,...rules.discoveryDraft(current.options),discoveryConfigured:Boolean(current.options?.discovery),outcome:'Blank connection fields keep the saved configuration. Use Remove secret to clear its reference.'});
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
  saveAndTestConnector: () => run(async () => {
   const form=draft().getFormData();
   const {operation,parameters}=rules.saveRequest(form);
   await call(operation,parameters);
   showSaved(await reload(form.name),'Configuration saved. Testing connection (30-second timeout)…');
   await refresh();
   const result=record(await call('test',{name:form.name}));
   showSaved(await reload(form.name),probeOutcome(result));
   await refresh();
  }),
  testConnector: props => run(async () => {
   const row=props?.row||selected();if(!row)throw new Error('Select a connector.');
   const result=record(await call('test',{name:row.name}));
   draft().setFormField({item:{dataField:'outcome'},value:probeOutcome(result)});await refresh();
  }),
  reloadConnector: () => run(async () => {const form=draft().getFormData();const current=await reload(form.name);draft().setFormData({values:{...current,editing:true,dsnTemplate:'',secretURL:'',secretKey:'',...rules.discoveryDraft(current.options),discoveryConfigured:Boolean(current.options?.discovery),outcome:'Reloaded current revision.'}});}),
  searchConnectors: () => run(async () => {
   const filter=ds('connector_filter').getFormData();
   ds('connectors').setInputParameters({query:(filter.query||'').trim(),status:filter.status||'',offset:0});
   await refresh();
  }),
  confirmDeleteConnector: props => run(async () => {
   const row=props?.row||selected();if(!row)throw new Error('Select a connector.');
   if(row.status==='active')throw new Error('Disable the connector before deleting it.');
   draft().setFormData({values:{name:row.name,etag:row.etag,outcome:'Deletion is blocked while a component references this connector.'}});
   context.handlers.window.openDialog({execution:{args:['connector-delete']}});
  }),
  deleteConnector: () => run(async () => {
   const form=draft().getFormData();await call('delete',{name:form.name,etag:form.etag});await refresh();
  }),
 };
})()

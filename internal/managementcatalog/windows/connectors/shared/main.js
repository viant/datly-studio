(() => {
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
   if (!value || typeof value!=='object' || Array.isArray(value) || Object.keys(value).some(key=>!['Driver','DSN','Secret'].includes(key))) throw new Error('Connector YAML supports Driver, DSN, and Secret only.');
   const patch={...form};
   if(Object.hasOwn(value,'Driver')) { if(typeof value.Driver!=='string') throw new Error('Driver must be a string.'); patch.driver=value.Driver; }
   if(Object.hasOwn(value,'DSN')) { if(typeof value.DSN!=='string') throw new Error('DSN must be a string.'); patch.dsnTemplate=value.DSN; }
   if(Object.hasOwn(value,'Secret')) {
    if(value.Secret===null) {patch.secretURL='';patch.secretKey='';patch.clearSecret=true;}
    else {
     const secret=value.Secret;
     if(!secret || typeof secret!=='object' || Array.isArray(secret) || Object.keys(secret).some(key=>!['URL','Key'].includes(key)) || typeof secret.URL!=='string' || !secret.URL.trim() || (secret.Key!==undefined && typeof secret.Key!=='string')) throw new Error('Secret requires URL and optional Key strings.');
     patch.secretURL=secret.URL;patch.secretKey=secret.Key||'';patch.clearSecret=false;
    }
   }
   draft().setFormData({values:patch});
  }),
  saveConnector: () => run(async () => {
   const form=draft().getFormData();
   const input={driver:form.driver,description:form.description||'',options:JSON.parse(form.options||'{}')};
   const url=(form.secretURL||'').trim(),key=(form.secretKey||'').trim();
   if(key&&!url&&!form.clearSecret) throw new Error('Enter the secret URL together with its key reference.');
   if(!form.editing || (form.dsnTemplate||'').trim()) input.dsnTemplate=(form.dsnTemplate||'').trim();
   if(!form.editing || url || form.clearSecret) input.secretRef=form.clearSecret?'':url+(key?'|'+key:'');
   if(form.editing) await call('update',{name:form.name,input:{...input,etag:form.etag}});
   else await call('create',{name:form.name,...input});
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

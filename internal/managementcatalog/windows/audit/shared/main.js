(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(op,parameters={})=>record(await ds('audit_'+op).fetchRecords({parameters}));
 let busy=false;
 const run=async fn=>{if(busy)return false;busy=true;set('audit_state',{outcome:''});try{await fn();return true;}catch(error){set('audit_state',{outcome:error.message});return false;}finally{busy=false;}};
 const filters=()=>{const filter={...get('audit_filter')};for(const key of ['from','to']){if(filter[key]){const date=new Date(filter[key]);if(Number.isNaN(date.valueOf()))throw new Error('Choose valid From and To dates.');filter[key]=date.toISOString();}else delete filter[key];}if(filter.from&&filter.to&&filter.from>filter.to)throw new Error('From must be before To.');return filter;};
 const load=async beforeId=>{const filter=filters();const page=await call('list',{filter:{...filter,beforeId,limit:50}});ds('audit_rows').setCollection(page.items||[]);set('audit_state',{nextId:page.nextId||0,outcome:page.dropped||page.failed?`Audit capture warning: ${page.dropped||0} dropped, ${page.failed||0} failed writes since startup.`:''});};
 return {
  initialize:()=>run(async()=>{set('audit_state',{nextId:0,canManage:false});const setting=await call('status');set('audit_state',setting);if(setting.canView)await load(0);else set('audit_state',{outcome:'Reading audit events requires the '+setting.viewerRole+' role.'});}),
  exportCSV:()=>run(async()=>{const file=await call('export',{filter:filters()});context.services.hostWindow.download(file);set('audit_state',{outcome:`Exported ${file.count||0} audit events.`+(file.truncated?' Export limit reached; narrow the filters or continue through the API.':'')});}),
  search:()=>run(()=>load(0)),
  older:()=>run(()=>load(get('audit_state').nextId||0)),
  saveSetting:()=>run(async()=>{const setting=await call('configure',{enabled:Boolean(get('audit_state').enabled)});set('audit_state',{...setting,outcome:'Audit setting saved.'});}),
  inspect:props=>run(async()=>{const row=props.row;if(!row)return;const page=await call('list',{filter:{id:row.id}});const item=page.items?.[0];if(!item)throw new Error('Audit event is no longer available.');set('audit_detail',{...item,capture:item.responseTruncated?'Response truncated or omitted at the capture limit.':'Complete captured response'});context.handlers.window.openDialog({execution:{args:['audit-detail']}});}),
 };
})()

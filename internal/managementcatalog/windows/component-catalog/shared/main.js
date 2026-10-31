(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const list=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?(value.length===1&&Array.isArray(value[0]?.items)?value[0].items:value):value?.items||[];};
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(id,parameters)=>record(await ds(id).fetchRecords({parameters}));
 const popup=id=>context.handlers.window.openDialog({execution:{args:[id]}});
 const params=()=>context.windowState?.parameters||{};
 const run=async(id,action)=>{if(get(id).saving)return false;set(id,{saving:true,outcome:''});try{await action();return true;}catch(error){set(id,{outcome:error.message});return false;}finally{set(id,{saving:false});}};
 const slugify=value=>value.toLowerCase().trim().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'');
 const open=report=>context.services.hostWindow.navigate('builder',{report});
 const refresh=async offset=>{
  const filter=get('catalog_filter');const result=await ds('catalog_list').fetchRecords({parameters:{query:(filter.query||'').trim(),namespace:filter.namespace||'',status:filter.status||'',limit:26,offset}});
  const rows=list(result);ds('catalog_rows').setCollection(rows.slice(0,25));set('catalog_state',{offset,hasNext:rows.length>25,outcome:''});
 };
 return {
  openAudit:props=>run('catalog_state',async()=>{const row=props.row||ds('catalog_rows').peekSelection()?.selected;if(!row)return;const setting=await call('catalog_audit_status',{componentId:row.id});set('catalog_audit',{...setting,title:row.title,outcome:setting.effective?'Auditing is active.':'Auditing is disabled for this component or namespace.'});popup('component-audit');}),
  saveAudit:()=>run('catalog_audit',async()=>{const form=get('catalog_audit');const setting=await call('catalog_audit_configure',{componentId:form.componentId,enabled:Boolean(form.enabled)});set('catalog_audit',{...setting,outcome:setting.effective?'Auditing enabled.':'Auditing disabled for this component or namespace.'});await refresh(get('catalog_state').offset||0);}),
  cancel:()=>{context.services.hostWindow.setDirty?.(false);return true;},
  initialize:()=>run('catalog_state',async()=>{
   set('catalog_state',{namespaceUnlocked:!params().currentNamespace,offset:0,hasNext:false});
   set('catalog_filter',{query:'',status:'',namespace:params().currentNamespace?.name||''});
   const [namespaces]=await Promise.all([ds('catalog_namespace_list').fetchRecords({parameters:{status:'active',limit:500}}),refresh(0)]);
   ds('catalog_namespaces').setCollection([{name:'',title:'All namespaces'},...list(namespaces)]);
  }),
  search:()=>run('catalog_state',()=>refresh(0)),
  previous:()=>run('catalog_state',()=>refresh(Math.max(0,(get('catalog_state').offset||0)-25))),
  next:()=>get('catalog_state').hasNext?run('catalog_state',()=>refresh((get('catalog_state').offset||0)+25)):false,
  newComponent:()=>run('catalog_state',async()=>{
   const [connectors,namespaces]=await Promise.all([ds('catalog_connector_list').fetchRecords({parameters:{status:'active',limit:100}}),ds('catalog_namespace_list').fetchRecords({parameters:{status:'active',limit:100}})]);
   const current=params().currentNamespace;const items=current?[current]:list(namespaces);
   ds('catalog_create_namespaces').setCollection(items);ds('catalog_connectors').setCollection(list(connectors).map(row=>({...row,label:row.name+' · '+row.driver})));
   ds('catalog_draft').setFormData({values:{title:'',slug:'',slugTouched:false,namespace:current?.name||items.find(row=>row.name==='general')?.name||items[0]?.name||'',namespaceLocked:Boolean(current),defaultConnectorName:'',description:'',saving:false,canCreate:items.length>0&&list(connectors).length>0,outcome:!items.length?'Create an active namespace before creating a component.':!list(connectors).length?'Create an active connector before creating a component.':''}});context.services.hostWindow.setDirty?.(true);popup('new-component');
  }),
  changeTitle:props=>{const value=typeof props.event==='string'?props.event:get('catalog_draft').title;set('catalog_draft',{title:value,...(!get('catalog_draft').slugTouched?{slug:slugify(value)}:{})});},
  changeSlug:()=>set('catalog_draft',{slugTouched:true}),
  createComponent:()=>run('catalog_draft',async()=>{
   const form=get('catalog_draft');
   if(!form.title?.trim()||!/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(form.namespace||'')||!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(form.slug||'')||!form.defaultConnectorName)throw new Error('Enter a title, namespace, lowercase dash-separated slug, and active connector.');
   if(params().currentNamespace&&form.namespace!==params().currentNamespace.name)throw new Error('Create the component in the selected namespace.');
   const report=await call('catalog_create',{title:form.title.trim(),namespace:form.namespace,slug:form.slug,defaultConnectorName:form.defaultConnectorName,description:(form.description||'').trim()});open(report);
  }),
  openComponent:props=>{const row=props.row||ds('catalog_rows').peekSelection()?.selected;if(row)open(row);},
  download:props=>run('catalog_state',async()=>{
   const row=props.row||ds('catalog_rows').peekSelection()?.selected;if(!row)return;
   let versionNo=row.currentDraftVersion;if(!versionNo){const versions=await ds('catalog_versions').fetchRecords({parameters:{reportId:row.id,input:{limit:1,orderBy:'version_no DESC'}}});versionNo=list(versions)[0]?.versionNo;}
   if(!versionNo)throw new Error('This component has no version to download.');
   context.services.hostWindow.download(await call('catalog_download',{reportId:row.id,versionNo}));
  }),
  openResourceChecks:props=>run('catalog_state',async()=>{
   const report=props.row||ds('catalog_rows').peekSelection()?.selected;if(!report)throw new Error('Select a component.');
   let versionNo=report.currentDraftVersion;if(!versionNo){const versions=list(await ds('catalog_versions').fetchRecords({parameters:{reportId:report.id,input:{limit:1,orderBy:'version_no DESC'}}}));versionNo=versions[0]?.versionNo;}
   if(!versionNo)throw new Error('Create a draft before configuring query resource checks.');
   const inspection=await call('catalog_resource_inspect',{reportId:report.id,versionNo,discoverColumns:false});const version=inspection.version;
   ds('catalog_resource_check').setFormData({values:{reportId:report.id,title:report.title,versionNo:version.versionNo,sourceRevision:version.sourceRevision,resourceCheckEnabled:version.resourceCheckEnabled===true,canEdit:inspection.capabilities?.canEdit!==false&&version.state==='draft',saving:false,outcome:''}});popup('component-resource-check');
  }),
  saveResourceChecks:()=>run('catalog_resource_check',async()=>{const form=get('catalog_resource_check');if(!form.canEdit)throw new Error('Create an editable draft before changing resource checks.');const result=await call('catalog_resource_apply',{reportId:form.reportId,versionNo:form.versionNo,command:{kind:'set_resource_check',expectedSourceRevision:form.sourceRevision,payload:{resourceCheckEnabled:Boolean(form.resourceCheckEnabled)}}});set('catalog_resource_check',{sourceRevision:result.version.sourceRevision});await refresh(0);}),
  openImport:()=>run('catalog_state',async()=>{
   const targets=list(await ds('catalog_list').fetchRecords({parameters:{limit:500}}));ds('catalog_targets').setCollection(targets);ds('catalog_import').setFormData({values:{reportId:'',file:null,entryDql:'',saving:false,outcome:''}});context.services.hostWindow.setDirty?.(true);popup('catalog-import');
  }),
  importComponent:()=>run('catalog_import',async()=>{
   const form=get('catalog_import'),file=form.file;
   if(!file||!form.reportId)throw new Error('Choose a component and DQL or archive file.');
   if(file.size>16*1024*1024)throw new Error('Choose a file smaller than 16 MiB.');
   const name=(file.name||file.filename||'').toLowerCase();let result;
   const format=name.endsWith('.zip')?'zip':/\.(tar.gz|tgz)$/.test(name)?'tar.gz':name.endsWith('.tar')?'tar':null;
   if(format)result=await call('catalog_load_archive',{reportId:form.reportId,input:{format,archive:file.data,entryDql:(form.entryDql||'').trim()}});
   else{if(!name.endsWith('.dql'))throw new Error('Choose a DQL, ZIP, TAR, or TAR.GZ file.');const dql=new TextDecoder().decode(Uint8Array.from(atob(file.data),character=>character.charCodeAt(0)));result=await call('catalog_load_dql',{reportId:form.reportId,input:{dql}});}
   const report=ds('catalog_targets').getCollection().find(row=>row.id===form.reportId);open({...report,versionNo:result.version.versionNo,currentDraftVersion:result.version.versionNo});
  }),
 };
})()

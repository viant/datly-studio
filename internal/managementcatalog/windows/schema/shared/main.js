(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const state=()=>get('schema_state');
 const sqlKey=form=>JSON.stringify([form.connectorName,form.catalogName||'',form.schemaName||'']);
 const rememberSQL=()=>{const form=get('schema_query_draft');if(form.connectorName)set('schema_state',{sqlDrafts:{...(state().sqlDrafts||{}),[sqlKey(form)]:form.sql||''}});};
 const rows=(id,values)=>ds(id).setCollection((values||[]).map((value,index)=>({id:value.id??value.name??String(index),...value})));
 const record=result=>{if(result&&Object.prototype.hasOwnProperty.call(result,'duration'))return result;const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const list=result=>{const value=record(result);return Array.isArray(value)?value:value?.items||[];};
 const call=async(id,input)=>{
  const request=ds(id).fetchRecords({parameters:input});
  if(!['schema_discover','schema_name_list','schema_table_list','schema_table'].includes(id))return record(await request);
  let timer;try{return record(await Promise.race([request,new Promise((_,reject)=>{timer=setTimeout(()=>reject(new Error('Database discovery timed out. Select a project/schema and retry.')),35000);})]));}finally{clearTimeout(timer);}
 };
 const host=()=>context.services?.hostWindow||{};
 const run=async(id,action)=>{if(get(id).saving)return false;set(id,{saving:true,outcome:''});try{await action();return true;}catch(error){set(id,{outcome:error.message});return false;}finally{set(id,{saving:false});}};
 const open=id=>{host().setDirty?.(true);context.handlers.window.openDialog({execution:{args:[id]}});};
 const selectTab=tabId=>context.handlers.window.selectTab({containerId:'schema-detail-tabs',tabId});
 const identifier=value=>String(value||'').toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_+|_+$/g,'').replace(/^[^a-z]+/,'')||'view';
 const words=value=>String(value||'').trim().replace(/([a-z0-9])([A-Z])/g,'$1 $2').split(/[^A-Za-z0-9]+/).filter(Boolean);
 const titleCase=value=>words(value).map(word=>word.length<=3?word.toUpperCase():word[0].toUpperCase()+word.slice(1).toLowerCase()).join(' ');
 const findView=(view,name)=>{if(!view)return null;if(view.name===name||view.namespace===name)return view;for(const relation of view.relations||[]){const found=findView(relation.view,name);if(found)return found;}return null;};
 const terminal=value=>String(value||'').replaceAll('`','').toLowerCase().split('.').at(-1);
 const connection=()=>ds('schema_connectors').getCollection().find(item=>item.name===state().connectorName)||{};
 const scope=()=>state().catalogName?{catalog:state().catalogName}:{};
 const unique=(values,key)=>[...new Map(values.map(item=>[key(item),item])).values()];
 const catalogChoices=discovered=>{
  const saved=connection().options?.discovery;
  const names=Object.hasOwn(saved||{},'catalogs')?saved.catalogs:(discovered||[]).map(item=>item.name);
  rows('schema_catalogs',[{name:'',label:'Connection default'},...unique(names.filter(Boolean).map(name=>({name,label:name})),item=>item.name)]);
 };
 const schemaChoices=discovered=>{
  const saved=connection().options?.discovery;
  const choices=Object.hasOwn(saved||{},'schemas')?saved.schemas.filter(item=>!item.catalog||item.catalog===state().catalogName):(discovered||[]);
  rows('schema_names',[{id:'default',name:'',label:'Connection default'},...unique(choices.filter(item=>item.name).map(item=>({...item,label:item.name})),item=>item.name)]);
 };
 let discoveryEpoch=0;
 const clearObjects=()=>{rows('schema_tables',[]);rows('schema_columns',[]);set('schema_state',{selected:false,detail:null,sql:'',hasMore:false});};
 const localChoices=()=>{
  catalogChoices([]);const catalogs=ds('schema_catalogs').getCollection();
  if(!catalogs.some(item=>item.name===state().catalogName))set('schema_state',{catalogName:catalogs.find(item=>item.name)?.name||''});
  schemaChoices([]);const schemas=ds('schema_names').getCollection();
  if(!schemas.some(item=>item.name===state().schemaName))set('schema_state',{schemaName:schemas.find(item=>item.name)?.name||''});
 };
 const cancelDiscovery=()=>{++discoveryEpoch;set('schema_state',{metadataLoading:false,loadStatus:'Choose a scope and click Load schemas.',outcome:''});};
 const discovery=async()=>{
  const form={...state()};if(!form.connectorName)return false;
  const epoch=++discoveryEpoch;
  set('schema_state',{metadataLoading:true,loadStatus:'Loading database and schemas…',outcome:''});
  try{
   const result=await call('schema_discover',{name:form.connectorName,input:form.catalogName?{catalog:form.catalogName}:{}});
   if(epoch!==discoveryEpoch)return false;
   catalogChoices(result?.catalogs);
   const catalogs=ds('schema_catalogs').getCollection();
   const catalogName=form.catalogName||(catalogs.find(item=>item.name)?.name||result?.catalog||'');
   set('schema_state',{catalogName,productVersion:result?.version||result?.product||''});
   schemaChoices(result?.schemas);const choices=ds('schema_names').getCollection();
   const schemaName=form.schemaName&&choices.some(item=>item.name===form.schemaName)?form.schemaName:(choices.find(item=>item.name)?.name||'');
   clearObjects();set('schema_state',{schemaName,loadStatus:'Schemas loaded. Choose one and search tables.',outcome:(result?.warnings||[]).join(' ')});return true;
  }catch(error){if(epoch===discoveryEpoch)set('schema_state',{outcome:error.message,loadStatus:'Load failed. You can retry or edit the lookup lists.'});return false;}
  finally{if(epoch===discoveryEpoch)set('schema_state',{metadataLoading:false});}
 };
 const tables=async offset=>{
  const form=state();if(!form.connectorName)return;
  const result=await call('schema_table_list',{name:form.connectorName,input:{...scope(),schema:form.schemaName||'',query:(form.query||'').trim(),limit:201,offset}});
  rows('schema_tables',(result?.items||[]).slice(0,200));set('schema_state',{tableOffset:offset,hasMore:(result?.items||[]).length>200});
 };
 const inspectComponent=async()=>{
  const form=get('schema_child_draft'),reportId=form.reportId,epoch=(state().childLoadEpoch||0)+1;
  set('schema_state',{childLoadEpoch:epoch});
  set('schema_child_draft',{loading:true,canCompile:false,inspection:null,loadEpoch:epoch,outcome:''});
  try{
   if(!reportId)return;
   const versions=await call('schema_version_list',{reportId,input:{limit:1}}),version=versions?.items?.[0];if(!version)throw new Error('Selected reader has no version.');
   const inspection=await call('schema_inspect',{reportId,versionNo:version.versionNo});
   if(get('schema_child_draft').loadEpoch!==epoch||get('schema_child_draft').reportId!==reportId)return;
   const views=inspection.structure?.views||[],parent=views[0]?.name||'';
   const table=state().detail?.table||{},parentTable=terminal(findView(inspection.structure?.component?.rootView,parent)?.source?.table);
   let child=identifier(table.name);const base=identifier(parentTable);if(parentTable&&child.startsWith(base+'_'))child=child.slice(base.length+1);if(child.startsWith('ci_'))child=child.slice(3);if(!child.endsWith('s'))child=child.endsWith('y')?child.slice(0,-1)+'ies':child+'s';
   const reference=(state().detail?.columns||[]).find(column=>column.referenceTable&&(!parentTable||terminal(column.referenceTable)===parentTable));
   rows('schema_views',views.map(view=>({name:view.name})));
   set('schema_child_draft',{inspection,parent,name:child,on:reference?`${child}.${reference.name}=${parent}.${reference.referenceColumn}`:'',canCompile:inspection.capabilities?.canEdit!==false&&(inspection.version.state||'draft')==='draft'});
  }catch(error){if(get('schema_child_draft').loadEpoch===epoch)set('schema_child_draft',{outcome:error.message});}
  finally{if(get('schema_child_draft').loadEpoch===epoch)set('schema_child_draft',{loading:false});}
 };
 return {
  workbenchState:()=>({...state(),driver:connection().driver||''}),
  executeWorkbench:props=>call('schema_test',{name:props.value.connectorName,input:{sql:props.value.sql,limit:props.value.limit||50,offset:props.value.offset||0,timeoutSeconds:props.value.timeoutSeconds}}),
  componentFromWorkbench:props=>{
   const source=props.value;if(!source?.connectorName||!source.sql?.trim())return false;
   const table=source.detail?.table;
   ds('schema_root_draft').setFormData({values:{title:table?titleCase(table.name):'SQL query',slug:table?words(table.name).map(word=>word.toLowerCase()).join('-'):'sql-query',viewName:table?identifier(table.name):'query',source:{...source},saving:false,outcome:''}});open('schema-root');return true;
  },
  subviewFromWorkbench:props=>{set('schema_state',{...props.value,selected:true});return context.lookupHandler?.('management.newChild')?.({context});},
  openSQL:()=>run('schema_state',async()=>{
   if(state().useWorkbench){set('schema_state',{sqlRequest:(state().sqlRequest||0)+1});return;}
   const current=state();if(!current.connectorName)throw new Error('Select a connector first.');
   const source={connectorName:current.connectorName,driver:connection().driver||'',catalogName:current.catalogName||'',schemaName:current.schemaName||''};
   ds('schema_query_draft').setFormData({values:{...source,sql:(current.sqlDrafts||{})[sqlKey(source)]??current.sql??'',extension:'sql',timeoutSeconds:60,contextLabel:[source.connectorName,source.driver,source.catalogName,source.schemaName].filter(Boolean).join(' · '),saving:false,outcome:''}});
   ds('schema_query_result').setFormData({values:{data:null}});open('schema-query');
  }),
  queryChanged:()=>ds('schema_query_result').setFormData({values:{data:null}}),
  runQuery:()=>run('schema_query_draft',async()=>{
   const form=get('schema_query_draft');if(!form.connectorName||!form.sql?.trim())throw new Error('Enter SQL to run.');
   const timeoutSeconds=Number(form.timeoutSeconds);if(!Number.isInteger(timeoutSeconds)||timeoutSeconds<0)throw new Error('Timeout must be a nonnegative whole number of seconds; 0 disables the Studio timeout.');
   ds('schema_query_result').setFormData({values:{data:null}});
   const result=await call('schema_test',{name:form.connectorName,input:{sql:form.sql.trim(),limit:50,timeoutSeconds}});
   ds('schema_query_result').setFormData({values:result});
  }),
  closeQuery:()=>{rememberSQL();host().setDirty?.(false);return true;},
  componentFromSQL:()=>{
   const form=get('schema_query_draft');if(form.saving)return false;if(!form.sql?.trim()){set('schema_query_draft',{outcome:'Enter SQL before creating a component.'});return false;}
   rememberSQL();context.handlers.window.closeDialog({dialogId:'schema-query'});
   ds('schema_root_draft').setFormData({values:{title:'SQL query',slug:'sql-query',viewName:'query',source:{...form},saving:false,outcome:''}});open('schema-root');return true;
  },
  invalidateSQL:()=>ds('schema_test_result').setFormData({values:{data:null}}),
  initialize:()=>run('schema_state',async()=>{const before=state(),result=await call('schema_connector_list',{status:'active',limit:100});const connectors=result?.items||[];rows('schema_connectors',connectors.map(item=>({...item,label:item.name})));set('schema_state',{useWorkbench:context.windowState?.parameters?.hasSQLWorkbench===true,connectorName:connectors.some(item=>item.name===before.connectorName)?before.connectorName:(connectors[0]?.name||''),catalogName:before.catalogName||'',schemaName:before.schemaName||'',query:'',selected:false,sql:'',timeoutSeconds:before.timeoutSeconds??60,extension:'sql',tableOffset:0,hasMore:false});cancelDiscovery();localChoices();clearObjects();}),
  changeConnector:()=>{cancelDiscovery();set('schema_state',{catalogName:'',schemaName:'',productVersion:''});localChoices();clearObjects();},
  changeCatalog:()=>{cancelDiscovery();set('schema_state',{schemaName:''});schemaChoices([]);clearObjects();},
  loadSchemas:discovery,
  cancelDiscovery,
  editLookups:()=>run('schema_state',async()=>{
   const current=await call('schema_connector_get',{name:state().connectorName}),saved=current.options?.discovery||{};
   const catalogs=saved.catalogs||(ds('schema_catalogs').getCollection().map(item=>item.name).filter(Boolean));
   const schemas=saved.schemas||(ds('schema_names').getCollection().filter(item=>item.name).map(item=>({catalog:state().catalogName||'',name:item.name})));
   ds('schema_lookup_draft').setFormData({values:{connectorName:current.name,etag:current.etag,options:current.options||{},catalogRows:catalogs.map(name=>({name})),schemaRows:schemas.map(item=>({...item})),outcome:''}});
   open('schema-lookups');
  }),
  discoverLookups:()=>run('schema_lookup_draft',async()=>{
   const form=get('schema_lookup_draft'),result=await call('schema_discover',{name:form.connectorName,input:scope()});
   set('schema_lookup_draft',{catalogRows:unique([...(form.catalogRows||[]),...(result?.catalogs||[]).map(item=>({name:item.name}))],item=>item.name),schemaRows:unique([...(form.schemaRows||[]),...(result?.schemas||[]).map(item=>({catalog:item.catalog||state().catalogName||'',name:item.name}))],item=>item.catalog+'\n'+item.name),outcome:(result?.warnings||[]).join(' ')});
  }),
  saveLookups:()=>run('schema_lookup_draft',async()=>{
   const form=get('schema_lookup_draft'),catalogs=form.catalogRows||[],schemas=form.schemaRows||[];
   if(catalogs.some(item=>!item.name?.trim())||schemas.some(item=>!item.name?.trim()))throw new Error('Enter a name for each lookup entry or remove its empty row.');
   const options={...form.options,discovery:{catalogs:[...new Set(catalogs.map(item=>item.name.trim()))],schemas:unique(schemas.map(item=>({catalog:(item.catalog||'').trim(),name:item.name.trim()})),item=>item.catalog+'\n'+item.name)}};
   const updated=await call('schema_connector_update',{name:form.connectorName,input:{options,etag:form.etag}});
   rows('schema_connectors',ds('schema_connectors').getCollection().map(item=>item.name===form.connectorName?{...item,...updated,options}:item));
   catalogChoices([]);const choices=ds('schema_catalogs').getCollection();set('schema_state',{catalogName:choices.some(item=>item.name===state().catalogName)?state().catalogName:(choices.find(item=>item.name)?.name||''),schemaName:''});cancelDiscovery();localChoices();clearObjects();
  }),
  changeSchema:()=>{cancelDiscovery();clearObjects();},
  searchTables:()=>run('schema_state',()=>tables(0)),
  previousTables:()=>run('schema_state',()=>tables(Math.max(0,(state().tableOffset||0)-200))),
  nextTables:()=>state().hasMore?run('schema_state',()=>tables((state().tableOffset||0)+200)):false,
  inspectTable:props=>run('schema_state',async()=>{const table=props.row||ds('schema_tables').peekSelection()?.selected;if(!table)return;const detail=await call('schema_table',{name:state().connectorName,input:{...scope(),schema:table.schema||state().schemaName||'',table:table.name}});set('schema_state',{tableRequest:(state().tableRequest||0)+1,detail,tableName:detail.table.name,selected:true,sql:'SELECT *\nFROM '+(connection().driver==='bigquery'?'`'+[detail.table.catalog||state().catalogName,detail.table.schema,detail.table.name].filter(Boolean).join('.')+'`':[detail.table.schema,detail.table.name].filter(Boolean).join('.')),extension:'sql'});rows('schema_columns',(detail.columns||[]).map(column=>({...column,constraints:[column.primaryKey?'PK':'',column.unique?'unique':'',column.nullable?'nullable':'required'].filter(Boolean).join(' · '),reference:column.referenceTable?column.referenceTable+'.'+column.referenceColumn:''})));ds('schema_test_result').setFormData({values:{data:null}});selectTab('schema-sql');}),
  closeTable:()=>{set('schema_state',{selected:false,detail:null,sql:''});ds('schema_test_result').setFormData({values:{data:null}});},
  testSQL:()=>run('schema_state',async()=>{if(!state().sql?.trim())throw new Error('Enter SQL to test.');const result=await call('schema_test',{name:state().connectorName,input:{sql:state().sql.trim(),limit:50,timeoutSeconds:Number(state().timeoutSeconds??60)}});ds('schema_test_result').setFormData({values:result});selectTab('schema-result');}),
  newRoot:()=>{const table=state().detail?.table;if(!table)return false;ds('schema_root_draft').setFormData({values:{title:titleCase(table.name),slug:words(table.name).map(word=>word.toLowerCase()).join('-'),viewName:identifier(table.name),source:{...state()},saving:false,outcome:''}});open('schema-root');},
  createRoot:()=>run('schema_root_draft',async()=>{
   const form=get('schema_root_draft'),source=form.source||state();if(!form.title?.trim()||!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(form.slug)||!/^[a-z][a-z0-9_]*$/.test(form.viewName)||!source.sql?.trim())throw new Error('Enter a title, lowercase slug, view identifier, and source SQL.');
   let report=form.createdReport,version=form.createdVersion;
   try{if(!report){report=await call('schema_component_create',{title:form.title.trim(),slug:form.slug,description:source.detail?.table?'Reader rooted at '+[source.detail.table.schema,source.detail.table.name].filter(Boolean).join('.')+'.':'Reader created from SQL.',defaultConnectorName:source.connectorName});set('schema_root_draft',{createdReport:report});}
    if(!version){version=await call('schema_version_create',{reportId:report.id,input:{authoringMode:'dql',notes:source.detail?.table?'Root view created from '+source.detail.table.name+'.':'Root view created from SQL.'}});set('schema_root_draft',{createdVersion:version});}
    const result=await call('schema_apply',{reportId:report.id,versionNo:version.versionNo,command:{expectedSourceRevision:version.sourceRevision,operation:{type:'createReader',reader:{route:'/v1/studio/readers/'+form.slug,name:form.viewName,sql:source.sql.trim()}}}});
    if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.[0]?.message||'Datly rejected the initial reader graph.');host().navigate('builder',{report:{...report,versionNo:version.versionNo}});
   }catch(error){throw new Error(report?'Reader '+report.title+' was created, but its initial graph failed: '+error.message:error.message);}
  }),
  newChild:()=>run('schema_state',async()=>{const result=await call('schema_component_list',{limit:100});const components=result?.items||[];rows('schema_components',components);ds('schema_child_draft').setFormData({values:{reportId:components[0]?.id||'',name:identifier(state().detail?.table?.name),parent:'',on:'',relationship:'rows',loading:false,saving:false,canCompile:false,outcome:''}});open('schema-child');inspectComponent();}),
  loadComponent:inspectComponent,
  addChild:()=>run('schema_child_draft',async()=>{
   const form=get('schema_child_draft'),source=state(),inspection=form.inspection;
   if(!form.canCompile||!inspection?.version||!/^[a-z][a-z0-9_]*$/.test(form.name)||!form.parent||!form.on?.trim()||!source.sql?.trim())throw new Error('Select a component and enter a child identifier, parent, and complete relation keys.');
   const report=ds('schema_components').getCollection().find(item=>item.id===form.reportId),lookup=form.relationship==='lookup';
   const addView={type:'addView',view:{name:form.name,kind:'subview',parent:form.parent,join:lookup?'LEFT JOIN':'JOIN',sql:source.sql.trim(),on:lookup?form.on.trim()+' AND 1=1':form.on.trim()}};
   const operation=source.connectorName!==report?.defaultConnectorName?{type:'batch',operations:[addView,{type:'addFunction',function:{name:'use_connector',args:[form.name,JSON.stringify(source.connectorName)]}}]}:addView;
   const result=await call('schema_apply',{reportId:form.reportId,versionNo:inspection.version.versionNo,command:{expectedSourceRevision:inspection.version.sourceRevision,operation}});
   if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.[0]?.message||'Datly rejected the related view.');host().navigate('builder',{report:{...report,versionNo:inspection.version.versionNo}});
  }),
  closeDialog:()=>{const epoch=(state().childLoadEpoch||0)+1;set('schema_state',{childLoadEpoch:epoch});set('schema_child_draft',{loadEpoch:epoch});host().setDirty?.(false);return true;},
 };
})()

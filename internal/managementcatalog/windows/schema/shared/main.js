(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const state=()=>get('schema_state');
 const rows=(id,values)=>ds(id).setCollection((values||[]).map((value,index)=>({id:value.id??value.name??String(index),...value})));
 const record=result=>{if(result&&Object.prototype.hasOwnProperty.call(result,'duration'))return result;const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const list=result=>{const value=record(result);return Array.isArray(value)?value:value?.items||[];};
 const call=async(id,input)=>record(await ds(id).fetchRecords({parameters:input}));
 const host=()=>context.services?.hostWindow||{};
 const run=async(id,action)=>{if(get(id).saving)return false;set(id,{saving:true,outcome:''});try{await action();return true;}catch(error){set(id,{outcome:error.message});return false;}finally{set(id,{saving:false});}};
 const open=id=>{host().setDirty?.(true);context.handlers.window.openDialog({execution:{args:[id]}});};
 const selectTab=tabId=>context.handlers.window.selectTab({containerId:'schema-detail-tabs',tabId});
 const identifier=value=>String(value||'').toLowerCase().replace(/[^a-z0-9]+/g,'_').replace(/^_+|_+$/g,'').replace(/^[^a-z]+/,'')||'view';
 const words=value=>String(value||'').trim().replace(/([a-z0-9])([A-Z])/g,'$1 $2').split(/[^A-Za-z0-9]+/).filter(Boolean);
 const titleCase=value=>words(value).map(word=>word.length<=3?word.toUpperCase():word[0].toUpperCase()+word.slice(1).toLowerCase()).join(' ');
 const findView=(view,name)=>{if(!view)return null;if(view.name===name||view.namespace===name)return view;for(const relation of view.relations||[]){const found=findView(relation.view,name);if(found)return found;}return null;};
 const terminal=value=>String(value||'').replaceAll('`','').toLowerCase().split('.').at(-1);
 const tables=async offset=>{
  const form=state();if(!form.connectorName)return;
  const result=await call('schema_table_list',{name:form.connectorName,input:{schema:form.schemaName||'',query:(form.query||'').trim(),limit:201,offset}});
  rows('schema_tables',(result?.items||[]).slice(0,200));set('schema_state',{tableOffset:offset,hasMore:(result?.items||[]).length>200});
 };
 const schemas=async()=>{
  const form=state();if(!form.connectorName)return;
  const result=await call('schema_name_list',{name:form.connectorName,input:{}});
  rows('schema_names',[{id:'default',name:'',label:'Default schema'},...(result?.items||[]).filter(item=>item.name).map(item=>({...item,label:item.name}))]);
  set('schema_state',{schemaName:'',selected:false,sql:''});rows('schema_columns',[]);await tables(0);
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
  invalidateSQL:()=>ds('schema_test_result').setFormData({values:{data:null}}),
  initialize:()=>run('schema_state',async()=>{const result=await call('schema_connector_list',{status:'active',limit:100});const connectors=result?.items||[];rows('schema_connectors',connectors.map(item=>({...item,label:item.name})));set('schema_state',{connectorName:connectors[0]?.name||'',schemaName:'',query:'',selected:false,sql:'',extension:'sql',tableOffset:0,hasMore:false});await schemas();}),
  changeConnector:()=>run('schema_state',schemas),
  changeSchema:()=>run('schema_state',()=>tables(0)),
  searchTables:()=>run('schema_state',()=>tables(0)),
  previousTables:()=>run('schema_state',()=>tables(Math.max(0,(state().tableOffset||0)-200))),
  nextTables:()=>state().hasMore?run('schema_state',()=>tables((state().tableOffset||0)+200)):false,
  inspectTable:props=>run('schema_state',async()=>{const table=props.row||ds('schema_tables').peekSelection()?.selected;if(!table)return;const detail=await call('schema_table',{name:state().connectorName,input:{schema:table.schema||state().schemaName||'',table:table.name}});set('schema_state',{detail,tableName:detail.table.name,selected:true,sql:'SELECT *\nFROM '+[detail.table.schema,detail.table.name].filter(Boolean).join('.'),extension:'sql'});rows('schema_columns',(detail.columns||[]).map(column=>({...column,constraints:[column.primaryKey?'PK':'',column.unique?'unique':'',column.nullable?'nullable':'required'].filter(Boolean).join(' · '),reference:column.referenceTable?column.referenceTable+'.'+column.referenceColumn:''})));ds('schema_test_result').setFormData({values:{data:null}});selectTab('schema-sql');}),
  closeTable:()=>{set('schema_state',{selected:false,detail:null,sql:''});ds('schema_test_result').setFormData({values:{data:null}});},
  testSQL:()=>run('schema_state',async()=>{if(!state().sql?.trim())throw new Error('Enter SQL to test.');const result=await call('schema_test',{name:state().connectorName,input:{sql:state().sql.trim(),limit:50}});ds('schema_test_result').setFormData({values:result});selectTab('schema-result');}),
  newRoot:()=>{const table=state().detail?.table;if(!table)return false;ds('schema_root_draft').setFormData({values:{title:titleCase(table.name),slug:words(table.name).map(word=>word.toLowerCase()).join('-'),viewName:identifier(table.name),saving:false,outcome:''}});open('schema-root');},
  createRoot:()=>run('schema_root_draft',async()=>{
   const form=get('schema_root_draft'),source=state();if(!form.title?.trim()||!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(form.slug)||!/^[a-z][a-z0-9_]*$/.test(form.viewName)||!source.sql?.trim())throw new Error('Enter a title, lowercase slug, view identifier, and source SQL.');
   let report=form.createdReport,version=form.createdVersion;
   try{if(!report){report=await call('schema_component_create',{title:form.title.trim(),slug:form.slug,description:'Reader rooted at '+[source.detail.table.schema,source.detail.table.name].filter(Boolean).join('.')+'.',defaultConnectorName:source.connectorName});set('schema_root_draft',{createdReport:report});}
    if(!version){version=await call('schema_version_create',{reportId:report.id,input:{authoringMode:'dql',notes:'Root view created from '+source.detail.table.name+'.'}});set('schema_root_draft',{createdVersion:version});}
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

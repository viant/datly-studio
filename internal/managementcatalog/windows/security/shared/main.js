(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const rows=(id,values)=>ds(id).setCollection((values||[]).map((value,index)=>({id:value.id??value.name??String(index),...value})));
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(id,input={})=>record(await ds(id).fetchRecords({parameters:input}));
 const host=()=>context.services.hostWindow,state=()=>get('security_state');
 const open=id=>context.handlers.window.openDialog({execution:{args:[id]}});
 const run=async(action,ref='security_state')=>{if(get(ref).saving)return false;set(ref,{saving:true,outcome:''});try{await action();return true;}catch(error){set(ref,{outcome:error.message});if(ref!=='security_state')set('security_state',{outcome:error.message});return false;}finally{set(ref,{saving:false});}};
 const resourceLabel=row=>({...row,tenantVersion:(row.resource?.tenant||'')+' · v'+row.resource?.version,permissions:(row.public?'Public':'Protected')+(row.inherited?' · inherited':'')+' · '+(row.explicitAcl?'Explicit ACL':row.public?'No explicit ACL':'Component authorization'),hasVersions:(row.versions||[]).length>1});
 const pageResources=async offset=>{const filter=get('security_filter'),result=await call('security_access',{kind:filter.kind||'',access:filter.access||'',query:(filter.query||'').trim(),limit:10,offset});rows('security_resources',(result.items||[]).map(resourceLabel));set('security_state',{offset,hasMore:Boolean(result.hasMore)});};
 const select=item=>{
  if(!item?.resource)return false;
  ds('security_selection').setFormData({values:{...item,resource:item.resource,displayName:item.name,actionsByKind:state().actionsByKind,choices:state().choices,reviewURL:item.explicitAcl===false?'':host().reviewURL?.(item.resource)||'',inheritedMessage:item.inherited&&item.explicitAcl!==false?'Skill '+item.name+' inherits permissions from '+item.ownerName+'. Changes below apply to the owning component.':''}});
  rows('security_info',[{label:item.name+' · '+(item.public?'Public':'Protected'),detail:item.public?'No explicit ACL is defined for this resource.':'This resource uses component authorization without a standalone policy at this version.'}]);
  set('security_state',{hasSelection:true,policyDirty:false,policySaving:false,pendingSelection:null});host().setDirty?.(false);return true;
 };
 const choose=item=>{if(state().policySaving)return false;if(state().policyDirty){set('security_state',{pendingSelection:item||null,pendingClose:!item});open('security-discard');return false;}if(item)return select(item);set('security_state',{hasSelection:false});host().setDirty?.(false);return true;};
 const selected=(props,id)=>props?.row||ds(id).peekSelection()?.selected;
 const pagePredicates=async offset=>{const filter=get('security_predicate_filter'),result=await call('security_predicate_list',{query:(filter.query||'').trim(),status:filter.status||'',limit:26,offset}),items=result.items||[];rows('security_predicates',items.slice(0,25).map(item=>({...item,goType:item.typeName+' · '+item.packagePath,linkedLabel:item.linked?'Linked':'Unavailable',sqlMetadata:item.alias||item.columns?.length?(item.alias||'—')+'.{'+(item.columns||[]).join(', ')+'}':'Not declared'})));set('security_state',{predicateOffset:offset,predicateHasMore:items.length>25,predicatesLoaded:true});};
 const types=async()=>{const result=await call('security_predicate_types'),items=(result.items||[]).map(item=>({...item,id:item.packagePath+'#'+item.typeName,label:item.packagePath+' · '+item.typeName}));rows('security_types',items);filterTypes();};
 const filterTypes=()=>{const query=(get('security_type_filter').query||'').toLowerCase();rows('security_filtered_types',ds('security_types').getCollection().filter(item=>[item.packagePath,item.typeName,item.alias,item.description].join(' ').toLowerCase().includes(query)));};
 const predicateDraft=item=>{ds('security_predicate_draft').setFormData({values:{...item,editing:Boolean(item?.name),status:item?.status||'active',handler:item?.packagePath?item.packagePath+'#'+item.typeName:'',columns:(item?.columns||[]).join(', '),saving:false,outcome:''}});host().setDirty?.(true);open('security-predicate-editor');};
 return {
  initialize:()=>run(async()=>{const parameters=host().parameters||{},permissionsOnly=parameters.permissionsOnly===true,kinds=parameters.kinds||['component','skill'];rows('security_kinds',kinds.map(id=>({id,label:id==='component'?'Components':id==='skill'?'Skills':id})));rows('security_kind_filters',[{id:'',label:'All types'},...ds('security_kinds').getCollection()]);set('security_state',{permissionsOnly,heading:permissionsOnly?'Permissions':'Security',description:permissionsOnly?'Manage resource permissions.':'Manage resource permissions and reusable authorization predicates.',actionsByKind:parameters.actionsByKind,choices:parameters.choices,offset:0,hasMore:false,predicateOffset:0,predicateHasMore:false,hasSelection:false,policyDirty:false,policySaving:false,predicatesLoaded:false});ds('security_filter').setFormData({values:{query:'',kind:'',access:''}});ds('security_predicate_filter').setFormData({values:{query:'',status:''}});ds('security_type_filter').setFormData({values:{query:''}});await pageResources(0);}),
  searchResources:()=>run(()=>pageResources(0)),
  previousResources:()=>run(()=>pageResources(Math.max(0,(state().offset||0)-10))),
  nextResources:()=>state().hasMore?run(()=>pageResources((state().offset||0)+10)):false,
  chooseResource:props=>choose(selected(props,'security_resources')),
  versions:props=>{const item=selected(props,'security_resources');if(!item)return;set('security_state',{versionedItem:item});rows('security_versions',(item.versions||[]).map(entry=>({...entry,id:JSON.stringify(entry.resource),version:entry.resource.version,tenant:entry.resource.tenant,published:entry.published?'Published':''})));open('security-versions');},
  chooseVersion:props=>{const version=selected(props,'security_versions');if(!version)return;const accepted=choose({...state().versionedItem,...version,resource:version.resource});if(accepted)context.handlers.window.closeDialog({dialogId:'security-versions'});},
  policyStateChanged:props=>{const value=props.value||{};set('security_state',{policyDirty:Boolean(value.dirty),policySaving:Boolean(value.saving)});host().setDirty?.(Boolean(value.dirty||value.saving));},
  closePermissions:()=>choose(null),
  discardPermissions:()=>{if(state().policySaving)return false;const next=state().pendingSelection;if(state().pendingClose){set('security_state',{hasSelection:false,policyDirty:false});host().setDirty?.(false);}else select(next);context.handlers.window.closeDialog({dialogId:'security-versions'});context.handlers.window.closeDialog({dialogId:'security-identity'});return true;},
  createPolicy:()=>set('security_selection',{explicitAcl:true}),
  openIdentity:()=>{ds('security_identity').setFormData({values:{kind:ds('security_kinds').getCollection()[0]?.id||'',id:'',tenant:'',version:'1',outcome:''}});open('security-identity');},
  loadIdentity:()=>{const form=get('security_identity'),resource={kind:form.kind,id:form.id?.trim(),tenant:form.tenant?.trim(),version:form.version?.trim()};if(!resource.id||!resource.tenant||!resource.version){set('security_identity',{outcome:'Enter resource ID, tenant, and version.'});return false;}return choose({name:resource.id,id:resource.id,kind:resource.kind,resource,explicitAcl:true});},
  tabChanged:props=>props.tabId==='security-predicates-tab'&&!state().predicatesLoaded?run(async()=>{const results=await Promise.allSettled([pagePredicates(0),types()]);const errors=results.filter(result=>result.status==='rejected').map(result=>result.reason.message);if(errors.length)set('security_state',{outcome:errors.join('\n')});}):false,
  searchPredicates:()=>run(()=>pagePredicates(0)),
  previousPredicates:()=>run(()=>pagePredicates(Math.max(0,(state().predicateOffset||0)-25))),
  nextPredicates:()=>state().predicateHasMore?run(()=>pagePredicates((state().predicateOffset||0)+25)):false,
  newPredicate:()=>run(async()=>{await types();predicateDraft({});}),
  editPredicate:props=>run(async()=>{const row=selected(props,'security_predicates');if(!row)return;const value=await call('security_predicate_get',{name:row.name});await types();predicateDraft(value);}),
  closePredicate:()=>{host().setDirty?.(false);return true;},
  savePredicate:()=>run(async()=>{
   const form=get('security_predicate_draft'),[packagePath,typeName]=String(form.handler||'').split('#');if(!form.name?.trim()||!form.title?.trim()||!packagePath||!typeName)throw new Error('Enter a canonical name, display name, and linked handler.');
   const input={title:form.title.trim(),description:(form.description||'').trim(),packagePath,typeName,alias:(form.alias||'').trim(),columns:String(form.columns||'').split(',').map(value=>value.trim()).filter(Boolean)};
   if(form.editing)await call('security_predicate_update',{name:form.name,input:{...input,status:form.status||'active',etag:form.etag}});else await call('security_predicate_create',{...input,name:form.name.trim()});
   host().setDirty?.(false);await pagePredicates(state().predicateOffset||0);
  },'security_predicate_draft'),
  confirmDelete:props=>{const row=selected(props,'security_predicates');if(!row)return;ds('security_confirm').setFormData({values:{...row,saving:false,outcome:''}});open('security-predicate-delete');},
  deletePredicate:()=>run(async()=>{const form=get('security_confirm');await call('security_predicate_delete',{name:form.name,etag:form.etag});await pagePredicates(0);},'security_confirm'),
  discoverTypes:()=>run(async()=>{await types();open('security-discovery');}),
  refreshTypes:()=>run(types),
  filterTypes,
 };
})()

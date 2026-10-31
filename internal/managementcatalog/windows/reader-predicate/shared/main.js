(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const draft=()=>ds('predicate_editor_draft').getFormData()||{};
 const set=values=>ds('predicate_editor_draft').setFormData({values:{...draft(),...values}});
 const host=()=>context.services?.hostWindow||{},params=()=>(context.windowState?.parameters||{});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(ref,parameters)=>record(await ds(ref).fetchRecords({parameters}));
 const rows=(ref,items)=>ds(ref).setCollection((items||[]).map((item,index)=>({id:item.id??item.name??String(index),...item})));
 let inspection,views=[],busy=false;
 const run=async action=>{if(busy)return false;busy=true;host().setBusy?.(true);set({saving:true,outcome:''});try{await action();return true;}catch(error){set({outcome:error.message});return false;}finally{busy=false;set({saving:false});host().setBusy?.(false);}};
 const single=name=>['expr','literal_in','multi_in','multi_not_in','when_present','when_not_present'].includes(name);
 const handlerDefaults=predicate=>{
  const values=predicate?.name==='handler'?[...(predicate.args||[]).slice(1),...(predicate.handlerArgs||[])]:[];
  return {predicateHandlerArgs:values.length?JSON.stringify(values):'',predicateHadHandlerArgs:values.length>0};
 };
 const handlerArguments=form=>{
  const text=String(form.predicateHandlerArgs||'').trim();
  if(!text)return form.predicateHadHandlerArgs?[]:undefined;
  let values;try{values=JSON.parse(text);}catch{throw new Error('Enter handler arguments as a JSON array.');}
  if(!Array.isArray(values)||values.some(value=>value!==null&&typeof value!=='string'&&typeof value!=='boolean'&&!Number.isSafeInteger(value)))throw new Error('Handler arguments support only strings, booleans, integers, and null.');
  return values;
 };
 const dimensions=()=>rows('predicate_dimensions',(views.find(view=>view.name===draft().predicateView)?.node.columns||[]).map(column=>({name:column.source||column.name,label:column.source&&column.source!==column.name?`${column.name} (${column.source})`:column.name})));
 const apply=async operation=>{if(!draft().canEdit||draft().locked)throw new Error('This predicate is read-only. Required authorization uses the privileged workflow.');const result=await call('predicate_apply',{reportId:params().reportId,versionNo:params().versionNo,command:{expectedSourceRevision:draft().sourceRevision,operation}});if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.map(item=>item.message).join('\n')||'Datly rejected this predicate change.');await host().complete({result,close:true});};
 return {
  initialize:async()=>{set({loading:true});try{
   inspection=await call('predicate_inspect',{reportId:params().reportId,versionNo:params().versionNo,discoverColumns:true});views=[];
   const visit=node=>{if(!node)return;const name=node.namespace||node.name;views.push({name,label:node.name&&node.name!==name?`${node.name} (${name})`:name,node});for(const relation of node.relations||[])visit(relation.view);};visit(inspection.structure?.component?.rootView);
   const inputs=(inspection.structure?.declarations||[]).filter(item=>item.parameter?.source?.kind!=='output');rows('predicate_inputs',inputs.map(item=>({name:item.parameter.name})));
   rows('predicate_views',views);rows('predicate_names',(inspection.structure?.availablePredicates||[]).map(name=>({name,label:name==='handler'?'Custom Go predicate':name})));
   let selected;
   for(const input of inputs)for(const item of input.predicates||[])if(`${input.parameter.name}:${item.ordinal}`===params().predicateID)selected={field:input.parameter.name,occurrence:item.ordinal,predicate:item.predicate};
   if(params().predicateID&&!selected)throw new Error('The predicate no longer exists. Close this editor and refresh.');
   const p=selected?.predicate,expansions=p?(inspection.structure.predicateExpansions||[]).filter(item=>item.group===p.group):[];
   ds('predicate_editor_draft').setFormData({values:{loading:false,saving:false,canEdit:inspection.capabilities?.canEdit!==false&&inspection.version.state==='draft',sourceRevision:inspection.version.sourceRevision,editing:Boolean(selected),securityName:'',locked:p?.group===99,field:selected?.field||params().field||inputs[0]?.parameter.name||'',occurrence:selected?.occurrence||0,predicateName:p?.name||'equal',predicateView:expansions[0]?.view||views[0]?.name||'',expansionViews:expansions.map(item=>item.view),predicateArgs:(p?.args||[]).join(', '),predicateAlias:p?.args?.[0]||'',predicateDimension:p?.args?.[1]||'',predicateHandler:p?.name==='handler'?p.args?.[0]||'':'',...handlerDefaults(p),predicateGroup:p?.group??1,predicateOperator:expansions[0]?.operator||'AND',predicateAbsent:p?.applyWhenAbsent===true,outcome:p?.group===99?'Required authorization predicates are read-only in ordinary filter authoring.':''}});dimensions();
   try{const security=await call('predicate_security_list',{status:'active',limit:200});rows('predicate_security',[{id:'none',name:'',label:'No authorization template'},...(security.items||[]).filter(item=>item.linked&&item.status==='active').map(item=>({...item,label:item.title+' — '+item.packagePath+'.'+item.typeName}))]);}catch(error){set({outcome:'Security predicate lookup unavailable: '+error.message});}
   try{const types=await call('predicate_types',{namespaceId:params().namespaceId});const handlers=(types.items||[]).map(item=>({name:`${item.packagePath}.${item.typeName}`,label:`${item.typeName} — ${item.packagePath}`}));if(draft().predicateHandler&&!handlers.some(item=>item.name===draft().predicateHandler))handlers.push({name:draft().predicateHandler,label:draft().predicateHandler+' (current)'});rows('predicate_handlers',handlers);}catch(error){set({outcome:'Custom predicate discovery unavailable: '+error.message});}
  }catch(error){set({loading:false,canEdit:false,outcome:error.message});}},
  close:()=>draft().saving?false:host().close(),changePredicate:()=>true,
  changeView:()=>{dimensions();set({predicateArgs:'',predicateDimension:''});},
  changeDimension:()=>{const p=draft(),qualified=/^([A-Za-z_][\w]*)\.([A-Za-z_][\w]*)$/.exec(p.predicateDimension||'');set({predicateAlias:qualified?.[1]||p.predicateAlias,predicateArgs:qualified?`${qualified[1]}, ${qualified[2]}`:`${p.predicateAlias||''}, ${p.predicateDimension||''}`});},
  save:()=>run(async()=>{const p=draft();if(p.securityName){if(!p.field||!p.predicateView)throw new Error('Choose an input and target view.');if(p.editing)throw new Error('Authorization templates are assigned as new required predicates.');const security=ds('predicate_security').getCollection().find(item=>item.name===p.securityName);if(!security?.linked||security.status!=='active')throw new Error('Choose an active linked Security predicate.');await apply({type:'addFieldPredicate',predicate:{field:p.field,occurrence:0,view:p.predicateView,name:'handler',args:[security.packagePath+'.'+security.typeName],group:99,groupOperator:'AND',applyWhenAbsent:true}});return;}const name=p.predicateName,args=name==='handler'?[p.predicateHandler]:single(name)?[p.predicateArgs]:String(p.predicateArgs||'').split(/\r?\n|,/).map(value=>value.trim()).filter(Boolean),group=Number(p.predicateGroup);if(!p.field||!p.predicateView||!args.length||args.some(value=>!value?.trim()))throw new Error('Choose an input, target view, and arguments.');if(!Number.isInteger(group)||group<0||group===99)throw new Error('Choose a non-negative filter group; group 99 is reserved for authorization.');const handlerArgs=name==='handler'?handlerArguments(p):undefined;await apply({type:p.editing?'updateFieldPredicate':'addFieldPredicate',predicate:{field:p.field,occurrence:p.occurrence,view:p.predicateView,expansionViews:p.editing?p.expansionViews:undefined,name,args,...(handlerArgs!==undefined?{handlerArgs}:{}),group,groupOperator:p.predicateOperator,applyWhenAbsent:p.predicateAbsent===true}});}),
  confirmRemove:()=>context.handlers.window.openDialog({execution:{args:['remove-predicate']}}),
  remove:()=>run(()=>apply({type:'removeFieldPredicate',predicate:{field:draft().field,occurrence:draft().occurrence}})),
 };
})()

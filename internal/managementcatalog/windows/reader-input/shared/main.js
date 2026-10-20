(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const draft=()=>get('input_draft');
 const host=()=>context.services.hostWindow;
 const params=()=>host().parameters;
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(ref,input={})=>record(await ds(ref).fetchRecords({parameters:input}));
 const rows=(ref,items)=>ds(ref).setCollection((items||[]).map((item,index)=>({id:item.id??item.name??String(index),...item})));
 let inspection=null,busy=false;
 const run=async(action,ref='input_draft')=>{
  if(busy)return false;
  busy=true;host().setBusy?.(true);set('input_draft',{saving:true});set(ref,{outcome:''});
  try{await action();return true;}
  catch(error){set(ref,{outcome:error.message});return false;}
  finally{busy=false;set('input_draft',{saving:false});host().setBusy?.(false);}
 };
 const views=()=>{const result=[];const visit=node=>{if(!node)return;const name=node.namespace||node.name;if(!result.some(item=>item.name===name))result.push({name,label:node.name&&node.name!==name?`${node.name} (${name})`:name,node});for(const relation of node.relations||[])visit(relation.view);};visit(inspection?.structure?.component?.rootView);return result;};
 const dimensions=view=>rows('input_dimensions',(views().find(item=>item.name===view)?.node.columns||[]).map(column=>({name:column.source||column.name,label:column.source&&column.source!==column.name?`${column.name} (${column.source})`:column.name})));
 const single=name=>['expr','literal_in','multi_in','multi_not_in','when_present','when_not_present','handler'].includes(name);
 const attached=()=>{
  const declaration=(inspection?.structure?.declarations||[]).find(item=>item.parameter?.name===draft().existingName);
  rows('input_attached',(declaration?.predicates||[]).map(item=>({id:`${draft().existingName}:${item.ordinal}`,name:item.predicate.name,args:(item.predicate.args||[]).join(', '),group:item.predicate.group,occurrence:item.ordinal,predicate:item.predicate,views:(inspection.structure.predicateExpansions||[]).filter(exp=>exp.group===item.predicate.group).map(exp=>exp.view)})));
 };
 const accept=value=>{
  inspection=value;
  set('input_draft',{sourceRevision:value.version.sourceRevision,canEdit:value.capabilities?.canEdit!==false&&value.version.state==='draft'});
  attached();
 };
 const apply=async operation=>{
  if(!draft().canEdit)throw new Error('This version is read-only. Create an editable draft first.');
  const result=await call('input_apply',{reportId:params().reportId,versionNo:params().versionNo,command:{expectedSourceRevision:draft().sourceRevision,operation}});
  if(!result?.applied)throw new Error(result?.inspection?.diagnostics?.map(item=>item.message).join('\n')||'Datly rejected the parameter change.');
  accept(result.inspection);
  return result;
 };
 const predicate=(form,field,occurrence=0)=>{
  const name=form.predicateName;
  const args=name==='handler'?[form.predicateHandler]:single(name)?[form.predicateArgs]:String(form.predicateArgs||'').split(/\r?\n|,/).map(value=>value.trim()).filter(Boolean);
  if(!field||!name||!form.predicateView||!args.length||args.some(value=>!value?.trim()))throw new Error('Choose a predicate, target view, and arguments.');
  const group=Number(form.predicateGroup);
  if(!Number.isInteger(group)||group<0||group===99)throw new Error('Choose a non-negative filter group. Required authorization uses the privileged workflow.');
  return {field,occurrence,name,args,view:form.predicateView,expansionViews:form.expansionViews,group,groupOperator:form.predicateOperator||'AND',applyWhenAbsent:form.predicateAbsent===true};
 };
 const predicateDefaults=()=>({predicateName:'',predicateView:views()[0]?.name||'',predicateAlias:'',predicateDimension:'',predicateArgs:'',predicateHandler:'',predicateGroup:1,predicateOperator:'AND',predicateAbsent:false});
 const initialize=async()=>{
  set('input_draft',{loading:true,outcome:''});
  try{
   const value=await call('input_inspect',{reportId:params().reportId,versionNo:params().versionNo,discoverColumns:true});
   inspection=value;
   const parameter=(value.structure?.declarations||[]).find(item=>item.parameter?.name===params().inputName)?.parameter;
   if(params().inputName&&!parameter)throw new Error('The input no longer exists. Close this editor and refresh the component.');
   const mode=parameter?.source?.kind==='const'||params().mode==='constant';
   const kinds=['query','path','header','cookie','form','body','http_request','const','param','component','view','data_view','input','caller_output','generator','cache','http_client','mcp_client','async'];
   for(const declaration of value.structure?.declarations||[]){const name=declaration.parameter?.source?.kind;if(name&&name!=='output'&&!kinds.includes(name))kinds.push(name);}
   rows('input_kinds',kinds.map(name=>({name,label:name})));
   rows('input_views',[{id:'none',name:'',label:'No query selector'},...views()]);
   rows('input_predicates',[{id:'none',name:'',label:'No predicate'},...(value.structure?.availablePredicates||[]).map(name=>({name,label:name==='handler'?'Custom Go predicate':name}))]);
   ds('input_draft').setFormData({values:{loading:false,saving:false,editing:Boolean(parameter),existingName:parameter?.name||'',canEdit:value.capabilities?.canEdit!==false&&value.version.state==='draft',sourceRevision:value.version.sourceRevision,name:parameter?.name||'',type:parameter?.outputTypeExpr&&parameter.outputTypeExpr!==parameter.typeExpr?`${parameter.typeExpr},${parameter.outputTypeExpr}`:parameter?.typeExpr||'string',kind:parameter?.source?.kind||(mode?'const':'query'),source:parameter?.source?.name||'',required:parameter?.required===true,querySelector:parameter?.querySelector?.view||parameter?.querySelector||'',codec:parameter?.codec?.body||'',codecArgs:(parameter?.codec?.args||[]).join(', '),codecAllowsArgs:parameter?.codec?.body!=='JwtClaim',errorCode:parameter?.errorStatusCode||'',errorMessage:parameter?.errorMessage||'',uri:parameter?.activation?.uri||parameter?.resourceRef||'',emitOutput:parameter?.emitOutput===true,value:parameter?.value??'',description:parameter?.description||'',example:parameter?.example||'',outcome:'',...predicateDefaults()}});
   dimensions(views()[0]?.name);attached();
   const results=await Promise.allSettled([call('input_codec_types'),call('input_handler_types')]);
   const warnings=[];
   if(results[0].status==='fulfilled')rows('input_codecs',[{id:'none',name:'',label:'No codec'},...(results[0].value.items||[]).map(item=>({...item,label:item.name}))]);else warnings.push('Codec discovery: '+results[0].reason.message);
   if(results[1].status==='fulfilled')rows('input_handlers',(results[1].value.items||[]).map(item=>({name:`${item.packagePath}.${item.typeName}`,label:`${item.typeName} — ${item.packagePath}`})));else warnings.push('Predicate discovery: '+results[1].reason.message);
   if(warnings.length)set('input_draft',{outcome:warnings.join('\n')});
  }catch(error){set('input_draft',{loading:false,canEdit:false,outcome:error.message});}
 };
 const popup=id=>context.handlers.window.openDialog({execution:{args:[id]}});
 const activePredicateContext=props=>props?.context?.identity?.dataSourceRef==='predicate_draft'?'predicate_draft':'input_draft';
 return {
  initialize,
  close:()=>draft().saving||draft().loading?false:host().close(),
  confirmReload:()=>popup('reload-input'),
  changeName:()=>{if(draft().kind==='const'&&!draft().source)set('input_draft',{source:draft().name});},
  changeKind:()=>{if(draft().kind==='const'&&!draft().source)set('input_draft',{source:draft().name});},
  changeCodec:()=>{const p=draft();if(p.codec==='JwtClaim')set('input_draft',{type:'string,*github.com/viant/scy/auth/jwt.Claims',kind:'header',source:'Authorization',required:true,errorCode:401,errorMessage:'Unauthorized',codecArgs:'',codecAllowsArgs:false});else set('input_draft',{codecAllowsArgs:true});},
  changePredicate:()=>true,
  changeView:props=>{const ref=activePredicateContext(props);dimensions(get(ref).predicateView);set(ref,{predicateDimension:'',predicateArgs:''});},
  changeDimension:props=>{const ref=activePredicateContext(props),p=get(ref);const qualified=/^([A-Za-z_][\w]*)\.([A-Za-z_][\w]*)$/.exec(p.predicateDimension||'');set(ref,{predicateAlias:qualified?.[1]||p.predicateAlias,predicateArgs:qualified?`${qualified[1]}, ${qualified[2]}`:`${p.predicateAlias||''}, ${p.predicateDimension||''}`});},
  save:()=>run(async()=>{
   const p=draft();
   if(!/^[A-Za-z][A-Za-z0-9_]*$/.test(p.name||''))throw new Error('Enter an UpperCamel input name.');
   if(!p.source?.trim()&&!['body','http_request','input'].includes(p.kind))throw new Error('Enter a kind source.');
   const errorStatusCode=p.errorCode===''?0:Number(p.errorCode);
   if(errorStatusCode!==0&&(!Number.isInteger(errorStatusCode)||errorStatusCode<400||errorStatusCode>599))throw new Error('Error code must be 400–599 or empty.');
   const field={name:p.name,type:p.type,sourceKind:p.kind,sourceName:p.source.trim(),required:p.kind==='const'?undefined:p.required===true,codec:p.codec?{name:p.codec,args:String(p.codecArgs||'').split(',').map(value=>value.trim()).filter(Boolean)}:null,updateCodec:true,errorStatusCode,errorMessage:p.errorMessage||'',uri:p.uri?.trim()||null,updateUri:true,emitOutput:p.emitOutput===true,description:p.description?.trim()||null,updateDescription:true,example:p.example===''?null:p.example,updateExample:true};
   if(p.editing){field.existingName=p.existingName;field.updateQuerySelector=p.kind==='const'?'':p.querySelector||'';field.updateValue=true;field.value=p.kind==='const'&&p.value!==''?p.value:null;}else{field.querySelector=p.querySelector||'';if(p.kind==='const'&&p.value!=='')field.value=p.value;}
   let operation={type:p.editing?'updateField':'addField',field};
   if(!p.editing&&p.predicateName)operation={type:'batch',operations:[operation,{type:'addFieldPredicate',predicate:predicate(p,p.name)}]};
   const result=await apply(operation);await host().complete({result,close:true});
  }),
  confirmRemove:()=>{set('input_draft',{confirmName:''});popup('remove-input');},
  remove:()=>run(async()=>{if(draft().confirmName!==draft().existingName)throw new Error('Type the exact input name to confirm removal.');const result=await apply({type:'removeField',field:{existingName:draft().existingName}});await host().complete({result,close:true});}),
  newPredicate:()=>{ds('predicate_draft').setFormData({values:{...predicateDefaults(),predicateName:'equal',outcome:'',editing:false,locked:false}});dimensions(views()[0]?.name);popup('predicate');},
  editPredicate:()=>{const row=ds('input_attached').peekSelection()?.selected;if(!row)return;const p=row.predicate;ds('predicate_draft').setFormData({values:{editing:true,occurrence:row.occurrence,locked:p.group===99,predicateName:p.name,predicateView:row.views?.[0]||views()[0]?.name,predicateArgs:(p.args||[]).join(', '),predicateAlias:p.args?.[0]||'',predicateDimension:p.args?.[1]||'',predicateHandler:p.name==='handler'?p.args?.[0]||'':'',predicateGroup:p.group,predicateOperator:'AND',predicateAbsent:p.applyWhenAbsent===true,expansionViews:row.views,outcome:''}});dimensions(get('predicate_draft').predicateView);popup('predicate');},
  savePredicate:()=>run(async()=>{const p=get('predicate_draft');if(p.locked)throw new Error('Required authorization predicates cannot be changed here.');const result=await apply({type:p.editing?'updateFieldPredicate':'addFieldPredicate',predicate:predicate(p,draft().existingName,p.occurrence||0)});await host().complete({result,close:false});},'predicate_draft'),
  confirmPredicateRemove:()=>{const row=ds('input_attached').peekSelection()?.selected;if(!row)return;if(row.group===99){set('input_draft',{outcome:'Required authorization predicates cannot be removed here.'});return;}popup('remove-predicate');},
  removePredicate:()=>run(async()=>{const row=ds('input_attached').peekSelection()?.selected;if(!row)throw new Error('Select a predicate.');if(row.group===99)throw new Error('Required authorization predicates cannot be removed here.');const result=await apply({type:'removeFieldPredicate',predicate:{field:draft().existingName,occurrence:row.occurrence}});await host().complete({result,close:false});context.handlers.window.closeDialog({dialogId:'predicate'});})
 };
})()

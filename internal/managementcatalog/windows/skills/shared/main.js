(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const get=id=>ds(id).getFormData()||{};
 const set=(id,values)=>ds(id).setFormData({values:{...get(id),...values}});
 const rows=(id,values)=>ds(id).setCollection((values||[]).map((value,index)=>({id:value.id??value.skillId??String(index),...value})));
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const call=async(id,input={})=>record(await ds(id).fetchRecords({parameters:input}));
 const host=()=>context.services?.hostWindow||{},params=()=>(context.windowState?.parameters||{});
 const state=()=>get('skill_state'),draft=()=>get('skill_draft');
 const run=async(action,ref='skill_state')=>{if(state().saving)return false;set('skill_state',{saving:true,outcome:'',notice:''});set(ref,{saving:true});try{await action();return true;}catch(error){set('skill_state',{outcome:error.code==='conflict'?'Skill changed elsewhere: '+error.message:error.message});if(error.code==='conflict')set('skill_draft',{conflict:true});return false;}finally{set('skill_state',{saving:false});set(ref,{saving:false});}};
 const markDirty=()=>{set('skill_draft',{dirty:true,hasContent:Boolean(draft().content?.trim())});host().setDirty?.(true);};
 const refreshAfter=async message=>{set('skill_state',{notice:message,outcome:''});try{await page(state().offset||0);}catch(error){set('skill_state',{outcome:'Catalog refresh failed: '+error.message});}};
 const fileRows=()=>rows('skill_files',(draft().files||[]).map(file=>({...file,format:file.isBinary?'Binary (preserved)':'Text'})));
 const copyBundle=bundle=>({title:bundle?.title||'',content:bundle?.content||'',files:(bundle?.files||[]).map((file,index)=>({...file,id:index+1})),nextFileId:(bundle?.files||[]).length+1});
 const openEditor=(skill,bundle,isNew=false)=>{
  ds('skill_draft').setFormData({values:{...copyBundle(bundle),skill,isNew,editorTitle:isNew?'New namespace skill':'Edit '+skill.name,revisionLabel:isNew?'Independent namespace draft':'Editing revision '+skill.latestRevision+' · publication changes only when you publish',dirty:isNew,hasContent:Boolean(bundle?.content?.trim()),conflict:false,saving:false,extension:'md'}});
  rows('skill_history',[]);ds('skill_revision').setFormData({values:{content:'',extension:'md'}});fileRows();set('skill_state',{editing:true});host().setDirty?.(true);
 };
 const page=async offset=>{
  if(!state().editable||state().useHost)return;
  const result=await call('skill_list',{query:(get('skill_filter').query||'').trim(),limit:21,offset});const list=result.items||[];
  rows('skill_rows',list.slice(0,20).map(skill=>({...skill,displayTitle:skill.title||skill.name,publishedRevision:skill.publishedRevision||0,publishable:skill.latestRevision!==(skill.publishedRevision||0),publication:skill.publishedRevision?'Published revision '+skill.publishedRevision:'Draft only'})));set('skill_state',{offset,hasNext:list.length>20});
 };
 const showHistory=async skill=>{const result=await call('skill_revisions',{skillId:skill.skillId,limit:100});rows('skill_history',result.items||[]);};
 const selected=(props,id)=>props?.row||ds(id).peekSelection()?.selected;
 const requireEditable=()=>{if(!state().editable)throw new Error('Namespace skills are managed by the namespace owner and namespace administrators.');};
 const cleanFiles=()=>draft().files.map(({id,format,...file})=>file);
 const openComponent=(component,versionNo,create=false,resourceSkillId='')=>host().navigate('builder',{report:{...component,versionNo,resourceSkillId,openResources:true,resourceAction:create?'newSkill':''}});
 const filterComponents=()=>{const query=(get('skill_component_filter').query||'').toLowerCase();rows('skill_component_rows',(state().componentSkillRows||[]).filter(row=>(row.skillRoot+' '+row.componentTitle).toLowerCase().includes(query)));};
 const loadComponents=()=>run(async()=>{
  const components=[],seen=new Set(),skills=[],failures=[];
  for(let offset=0;;offset+=100){const result=await call('skill_component_list',{limit:100,offset}),items=result.items||[];if(items.length&&items.every(item=>seen.has(item.id)))throw new Error('Component catalog did not advance. Refresh and retry.');for(const item of items){seen.add(item.id);components.push(item);}if(items.length<100)break;}
  rows('skill_components',components);
  for(const component of components){try{let versionNo=component.currentDraftVersion;if(!versionNo){const versions=await call('skill_version_list',{reportId:component.id,input:{limit:1}});versionNo=versions.items?.[0]?.versionNo;}if(!versionNo)continue;const snapshot=await call('skill_resources',{reportId:component.id,versionNo});for(const skill of snapshot.skills||[])skills.push({...skill,id:component.id+':'+versionNo+':'+skill.skillId,component,componentTitle:component.title,versionNo});}catch(error){if(error.name==='AbortError')throw error;failures.push(component.title+': '+error.message);}}
  set('skill_state',{componentSkillRows:skills,componentsLoaded:true,outcome:failures.length?'Some component skills could not be read: '+failures.join('\n'):''});filterComponents();
 });
 return {
  initialize:()=>run(async()=>{const namespace=params().currentNamespace||{};set('skill_state',{editable:namespace.canManage!==false,mcpEnabled:Boolean(namespace.mcpEnabled),useHost:Boolean(params().hasNamespaceSkillsRenderer),editing:false,offset:0,hasNext:false,componentsLoaded:false,componentSkillRows:[]});ds('skill_filter').setFormData({values:{query:''}});ds('skill_component_filter').setFormData({values:{query:'',componentId:''}});await page(0);}),
  ownershipChanged:props=>{if(props.tabId==='component-owned'&&!state().componentsLoaded)return loadComponents();},
  externalDirty:props=>{const editing=Boolean(props.value||props.args?.[1]?.busy);set('skill_state',{editing});host().setDirty?.(editing);},
  searchNamespace:()=>run(()=>page(0)),
  previousNamespace:()=>run(()=>page(Math.max(0,(state().offset||0)-20))),
  nextNamespace:()=>state().hasNext?run(()=>page((state().offset||0)+20)):false,
  newNamespace:()=>{requireEditable();openEditor({}, {title:'',content:'---\nname: namespace-guide\ndescription: Describe when to use this namespace skill.\n---\n\nWrite the instructions here.\n',files:[]},true);},
  editNamespace:props=>run(async()=>{requireEditable();const skill=selected(props,'skill_rows');if(!skill)return;const result=await call('skill_get',{skillId:skill.skillId,revision:0});openEditor(result.skill,result.revision.bundle);}),
  markDirty,
  saveNamespace:()=>run(async()=>{
   requireEditable();const form=draft();if(form.conflict)throw new Error('Reload the saved revision before saving again.');if(!form.content?.trim())throw new Error('Enter SKILL.md content.');
   const bundle={title:form.title||'',content:form.content,files:cleanFiles()};
   const result=await call(form.isNew?'skill_create':'skill_update',form.isNew?{bundle}:{skillId:form.skill.skillId,etag:form.skill.etag,bundle});
   set('skill_state',{editing:false});host().setDirty?.(false);await refreshAfter('Saved '+result.skill.name+', revision '+result.skill.latestRevision+'.');
  },'skill_draft'),
  reloadNamespace:()=>run(async()=>{const skill=draft().skill;if(!skill?.skillId)return;const result=await call('skill_get',{skillId:skill.skillId,revision:0});openEditor(result.skill,result.revision.bundle);}),
  discardNamespace:()=>{if(state().saving)return false;set('skill_state',{editing:false,outcome:''});host().setDirty?.(false);return true;},
  newFile:()=>{if(state().saving)return false;ds('skill_file_draft').setFormData({values:{id:draft().nextFileId||1,path:'',content:'',isBinary:false,outcome:''}});context.handlers.window.openDialog({execution:{args:['skill-file']}});},
  editFile:props=>{if(state().saving)return false;const file=selected(props,'skill_files');if(!file)return;ds('skill_file_draft').setFormData({values:{...file,outcome:''}});context.handlers.window.openDialog({execution:{args:['skill-file']}});},
  saveFile:()=>{if(state().saving)return false;const file=get('skill_file_draft');if(file.isBinary)return false;if(!file.path?.trim()){set('skill_file_draft',{outcome:'Enter a relative supporting-file path.'});return false;}const previous=(draft().files||[]).find(item=>item.id===file.id);const value={...previous,id:file.id,path:file.path.trim(),content:file.content||''};const files=previous?draft().files.map(item=>item.id===file.id?value:item):[...(draft().files||[]),value];set('skill_draft',{files,nextFileId:Math.max(draft().nextFileId||1,file.id+1)});fileRows();markDirty();return true;},
  removeFile:props=>{if(state().saving)return false;const file=selected(props,'skill_files');if(!file)return false;set('skill_draft',{files:(draft().files||[]).filter(item=>item.id!==file.id)});fileRows();markDirty();},
  showHistory:props=>run(async()=>{const skill=selected(props,'skill_rows');if(!skill)return;const result=await call('skill_get',{skillId:skill.skillId,revision:0});openEditor(result.skill,result.revision.bundle);await showHistory(skill);context.handlers.window.selectTab({containerId:'skill-editor-tabs',tabId:'namespace-skill-history'});}),
  editorTabChanged:props=>props.tabId==='namespace-skill-history'&&!draft().isNew?run(()=>showHistory(draft().skill)):false,
  viewRevision:props=>{const revision=selected(props,'skill_history');if(revision)ds('skill_revision').setFormData({values:{content:revision.bundle?.content||'',extension:'md'}});},
  publishNamespace:props=>run(async()=>{requireEditable();if(!state().mcpEnabled)throw new Error('Enable MCP in namespace settings before publishing this skill.');const skill=selected(props,'skill_rows');if(!skill)return;await call('skill_publish',{skillId:skill.skillId,etag:skill.etag,revision:skill.latestRevision});await refreshAfter('Published '+skill.name+', revision '+skill.latestRevision+'.');}),
  unpublishNamespace:props=>run(async()=>{requireEditable();const skill=selected(props,'skill_rows');if(!skill?.publishedRevision)throw new Error('This skill has no published revision.');await call('skill_unpublish',{skillId:skill.skillId,etag:skill.etag});await refreshAfter('Unpublished '+skill.name+'.');}),
  confirmArchive:props=>{requireEditable();const skill=selected(props,'skill_rows');if(!skill||skill.publishedRevision){set('skill_state',{outcome:'Unpublish the skill before archiving it.'});return false;}ds('skill_confirm').setFormData({values:skill});context.handlers.window.openDialog({execution:{args:['skill-archive']}});},
  archiveNamespace:()=>run(async()=>{requireEditable();const skill=get('skill_confirm');await call('skill_delete',{skillId:skill.skillId,etag:skill.etag});set('skill_state',{offset:0});await refreshAfter('Archived '+skill.name+'.');}),
  loadComponents,filterComponents,
  newComponentSkill:()=>{const component=ds('skill_components').getCollection().find(item=>item.id===get('skill_component_filter').componentId);if(component)openComponent(component,component.currentDraftVersion,true);else set('skill_state',{outcome:'Choose a component to create a skill.'});},
  editComponentSkill:props=>{const skill=selected(props,'skill_component_rows');if(skill)openComponent(skill.component,skill.versionNo,false,skill.skillId);},
 };
})()

(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const draft=()=>ds('component_draft');
 const form=()=>draft().getFormData();
 const set=patch=>draft().setFormData({values:{...form(),...patch}});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 let busy=false,epoch=0,request=new AbortController();
 const namespace=()=>draft().getWindowFormData().namespaceId;
 const call=async(method,parameters={})=>{
  const namespaceId=namespace();if(!namespaceId)throw new Error('Choose a namespace.');
  return record(await ds('component_'+method.replaceAll('.','_')).fetchRecords({parameters:{namespaceId,...parameters},signal:request.signal}));
 };
 const run=async fn=>{
  if(busy)return false;busy=true;const started=epoch;
  try{await fn();return started===epoch;}catch(error){if(started===epoch&&!request.signal.aborted)set({outcome:error.message});return false;}finally{if(started===epoch)busy=false;}
 };
 const refresh=()=>{const namespaceId=namespace();if(!namespaceId)throw new Error('Choose a namespace.');ds('components').setInputParameters({namespaceId});ds('components').fetchCollection();};
 const chosen=()=>{const row=ds('components').peekSelection()?.selected;if(!row)throw new Error('Select a component.');return row;};
 const identity=()=>{const current=form();if(!current.reportId||!current.versionNo)throw new Error('Open a reader version.');return {reportId:current.reportId,versionNo:Number(current.versionNo)};};
 const showResources=snapshot=>{
  ds('component_files').setCollection((snapshot.files||[]).map(file=>({...file,treePath:file.namespace+'/'+file.resourcePath})));
  const prior=form().selectedFile;
  const selectedFile=prior?(snapshot.files||[]).find(file=>file.resourceId===prior.resourceId || file.namespace===prior.namespace&&file.resourcePath===prior.resourcePath)||prior:null;
  set({files:snapshot.files||[],folders:snapshot.folders||[],...(snapshot.version?{sourceRevision:snapshot.version.sourceRevision}:{}),selectedFile});
 };
 const load=async(reportId,versionNo)=>{
  const started=epoch;
  const inspection=await call('versions.inspect',{reportId,versionNo:Number(versionNo),discoverColumns:false});
  if(started!==epoch)return;
  const version=inspection.version;
  set({selectedFile:null,reportId,versionNo:version.versionNo,sourceRevision:version.sourceRevision,state:version.state,source:inspection.dql||version.authoredDql||'',extension:'sql',location:'Reader DQL',inspection,diagnostics:JSON.stringify(inspection.diagnostics||[],null,2),previewJSON:'',outcome:'Loaded source revision '+version.sourceRevision});
  const snapshot=await call('resources.get',{reportId,versionNo:Number(versionNo)});
  if(started===epoch)showResources(snapshot);
 };
 const metadata=async editing=>{
  if(!namespace())throw new Error('Choose a namespace.');
  const current=editing?await call('components.get',{id:chosen().id}):{slug:'',title:'',description:'',defaultConnectorName:''};
  set({metadata:current,editingMetadata:editing,slug:current.slug,title:current.title,description:current.description||'',defaultConnectorName:current.defaultConnectorName,outcome:''});
  context.handlers.window.openDialog({execution:{args:['component-metadata']}});
 };
 return {
  idle:()=>true,
  selectNamespace:()=>{
   request.abort();request=new AbortController();epoch++;busy=false;
   draft().setFormData({values:{outcome:'',versionNo:0,source:'',extension:'sql',previewInput:'{}',limit:50,reason:''}});
   ds('components').setCollection([]);ds('component_files').setCollection([]);
   if(namespace())refresh();
  },
  refreshComponents:()=>run(async()=>refresh()),
  newComponent:()=>run(()=>metadata(false)),
  editComponent:()=>run(()=>metadata(true)),
  saveComponent:()=>run(async()=>{
   const current=form(),input={slug:current.slug,title:current.title,description:current.description||'',defaultConnectorName:current.defaultConnectorName};
   const saved=current.editingMetadata?await call('components.update',{id:current.metadata.id,input:{...input,etag:current.metadata.etag}}):await call('components.create',input);
   set({reportId:saved.id,metadata:saved,outcome:'Component saved.'});refresh();
  }),
  openReader:()=>run(async()=>{
   const selected=await call('components.get',{id:chosen().id});
   let versionNo=selected.currentDraftVersion;
   if(!versionNo){const page=await call('versions.list',{reportId:selected.id,input:{limit:1,offset:0}});versionNo=page.items?.[0]?.versionNo;}
   set({reportId:selected.id,metadata:selected,previewInput:'{}',limit:50,reason:''});
   if(!versionNo){set({versionNo:0,source:'',extension:'sql',outcome:'Create a draft to begin editing.'});return;}
   await load(selected.id,versionNo);
  }),
  loadVersion:()=>run(async()=>{const current=form();await load(current.reportId,Number(current.versionNo));}),
  createDraft:()=>run(async()=>{const current=form();if(!current.reportId)throw new Error('Select or create a component.');const version=await call('versions.create',{reportId:current.reportId,input:{authoringMode:'dql',authoredDql:current.source||'',notes:current.reason||''}});await load(current.reportId,version.versionNo);refresh();}),
  cloneDraft:()=>run(async()=>{const current=form(),id=identity();const version=await call('versions.clone',{...id,expectedSourceRevision:current.sourceRevision});await load(id.reportId,version.versionNo);refresh();}),
  saveDQL:()=>run(async()=>{const current=form(),id=identity();if(current.selectedFile)throw new Error('Load the reader DQL before saving its source. Use Save selected file for a retained file.');await call('versions.apply',{...id,command:{kind:'set_dql',expectedSourceRevision:current.sourceRevision,payload:{authoredDql:current.source}}});await load(id.reportId,id.versionNo);}),
  openFile:props=>run(async()=>{
   const file=props?.item||props?.nodeData||ds('component_files').peekSelection()?.selected;
   if(!file?.resourceId)throw new Error('Select a source file.');if(file.isBinary)throw new Error('Binary source is retained but cannot be edited as text.');
   const extension=file.resourcePath.split('.').pop().toLowerCase();
   set({selectedFile:file,source:file.content||'',location:file.namespace+'/'+file.resourcePath,extension:extension==='dql'?'sql':extension,outcome:''});
  }),
  newFile:()=>run(async()=>{identity();set({selectedFile:{namespace:'',resourcePath:'',mediaType:'text/plain'},fileNamespace:'',filePath:'',source:'',location:'New source file',extension:'sql'});context.handlers.window.openDialog({execution:{args:['component-file']}});}),
  saveFileIdentity:()=>run(async()=>{const current=form();set({selectedFile:{...current.selectedFile,namespace:current.fileNamespace,resourcePath:current.filePath},location:current.fileNamespace+'/'+current.filePath});}),
  saveFile:()=>run(async()=>{const current=form(),id=identity();if(!current.selectedFile)throw new Error('Open a source file.');const file=current.selectedFile;const snapshot=await call('resources.upsert_file',{...id,resourceId:file.resourceId,namespace:file.namespace,resourcePath:file.resourcePath,mediaType:file.mediaType,content:current.source,expectedSourceRevision:current.sourceRevision});showResources(snapshot);set({outcome:'Source file saved. Reload or validate to inspect the updated reader.'});}),
  validateReader:()=>run(async()=>{const current=form();const result=await call('versions.validate',{...identity(),expectedSourceRevision:current.sourceRevision});set({diagnostics:JSON.stringify(result.diagnostics||[],null,2),...(result.version?{sourceRevision:result.version.sourceRevision}:{}),outcome:result.valid?'Validation passed.':'Validation failed. Review diagnostics.'});}),
  previewReader:()=>run(async()=>{const current=form();const result=await call('preview.execute',{...identity(),input:{input:JSON.parse(current.previewInput||'{}'),limit:Number(current.limit||50),cube:Boolean(current.cube)}});set({previewJSON:JSON.stringify(result.data??result,null,2),diagnostics:JSON.stringify(result.diagnostics||[],null,2),outcome:'Preview completed.'});}),
  publishReader:()=>run(async()=>{const current=form();const result=await call('publications.publish',{...identity(),input:{expectedSourceRevision:current.sourceRevision,reason:current.reason||''}});set({publication:result,outcome:'Publication '+result.status});refresh();}),
  unpublishReader:()=>run(async()=>{const current=form(),id=identity();const publication=await call('publications.get',{reportId:id.reportId});const result=await call('publications.unpublish',{reportId:id.reportId,input:{expectedActiveGeneration:publication.activeGeneration,reason:current.reason||''}});set({publication:result,outcome:'Publication '+result.status});refresh();}),
  rollbackReader:()=>run(async()=>{const current=form();const result=await call('publications.rollback',{...identity(),input:{expectedSourceRevision:current.sourceRevision,reason:current.reason||''}});set({publication:result,outcome:'Rollback '+result.status});refresh();}),
  applyStructureCommand:()=>run(async()=>{const current=form(),id=identity();const operation=JSON.parse(current.structureCommand||'{}');const result=await call('versions.builder',{...id,command:{expectedSourceRevision:current.sourceRevision,operation}});if(!result.applied)throw new Error(result.inspection?.diagnostics?.[0]?.message||'Structural command was rejected.');await load(id.reportId,id.versionNo);})
 };
})()

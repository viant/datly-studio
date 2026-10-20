(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const set=(id,rows)=>ds(id).setCollection(rows);
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 const list=(result,key)=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value:value?.[key]||[];};
 const state=values=>ds('runtime_state').setFormData({values:{...ds('runtime_state').getFormData(),...values}});
 const summary=schema=>schema?.properties?Object.keys(schema.properties).length+' fields':schema?'defined':'not declared';
 let readers=[],tools=[],resources=[],busy=false;
 const inventory=key=>ds('runtime_inventory').getFormData()?.[key]||[];
 const filter=(id,rows,text)=>{const query=(ds(id).getFormData()?.query||'').trim().toLowerCase();return query?rows.filter(row=>text(row).toLowerCase().includes(query)):rows;};
 const changed=(id,props)=>{if(typeof props?.event==='string')ds(id).setFormField({item:{dataField:'query'},value:props.event});};
 const searchComponents=props=>{changed('runtime_component_filter',props);return set('runtime_components',filter('runtime_component_filter',inventory('readers'),row=>[row.title,row.namespace,row.connectorName,row.runtimeRevision].join(' ')));};
 const searchTools=props=>{changed('runtime_tool_filter',props);return set('runtime_tools',filter('runtime_tool_filter',inventory('tools'),row=>[row.name,row.description].join(' ')));};
 const searchResources=props=>{changed('runtime_resource_filter',props);return set('runtime_resources',filter('runtime_resource_filter',inventory('resources'),row=>[row.uriPrefix,row.namespace,row.rootPath,row.componentTitle].join(' ')));};
 const openComponent=props=>{const row=props?.row||props?.item||ds('runtime_components').peekSelection()?.selected;if(row)context.services.hostWindow.navigate('builder',{report:{id:row.reportId,title:row.title,namespace:row.namespace,ownerPackage:row.ownerPackage,defaultConnectorName:row.connectorName,versionNo:row.versionNo}});};
 const schemaFields=schema=>Object.entries(schema?.properties||{}).map(([name,property])=>({id:name,name,type:Array.isArray(property.type)?property.type.join(' | '):property.type||'object',required:(schema.required||[]).includes(name)?'required':'optional',description:property.description||''}));
 return {
  refreshRuntime:async()=>{
   if(busy)return false;busy=true;state({outcome:'',diagnosticCount:0});
   try{
    const results=await Promise.allSettled([ds('runtime_status').fetchRecords(),ds('runtime_live_tools').fetchRecords()]);
    const errors=[];
    if(results[0].status==='fulfilled'){
     const status=record(results[0].value)||{};
     readers=(status.readers||[]).map(row=>({...row,id:row.reportId,version:'v'+row.versionNo,exposed:row.mcpExposures?.length?row.mcpExposures.length+' exposed':'None'}));
     resources=readers.flatMap(reader=>(reader.mcpResources||[]).map(resource=>({...resource,id:reader.reportId+':'+resource.uriPrefix,componentTitle:reader.title})));
     set('runtime_summary',[{id:'current',label:status.activeGeneration?'Generation '+status.activeGeneration:'No active generation',detail:[(status.status||'unknown')+' record',(status.reportCount||0)+' deployed components',status.activatedAt?'Activated '+status.activatedAt:'Not activated',status.host?.status==='ready'?'Live runtime ready · revision '+status.host.revision:status.host?.status==='unavailable'?'Live runtime check failed':'Live runtime not checked'].join(' · '),status:status.host?.status==='ready'?'Ready':'Review'}]);
     state({diagnosticCount:status.diagnostics?.length||0});
     set('runtime_diagnostics',(status.diagnostics||[]).map((row,index)=>({...row,id:index,code:row.code||'Runtime diagnostic'})));
    }else{errors.push('Runtime status unavailable: '+results[0].reason.message);readers=[];resources=[];set('runtime_summary',[]);set('runtime_diagnostics',[]);}
    if(results[1].status==='fulfilled')tools=list(results[1].value,'tools').map(row=>({...row,id:row.name,inputSummary:summary(row.inputSchema),outputSummary:summary(row.outputSchema)}));
    else{tools=[];errors.push('Datly MCP catalog unavailable: '+results[1].reason.message);}
    ds('runtime_inventory').setFormData({values:{readers,tools,resources}});
    searchComponents();searchTools();searchResources();state({outcome:errors.join('\n')});
   }finally{busy=false;}
  },
  searchComponents,searchTools,searchResources,openComponent,
  inspectTool:props=>{
   const tool=props?.row||ds('runtime_tools').peekSelection()?.selected;if(!tool)return false;
   ds('runtime_tool').setFormData({values:{name:tool.name,inputJSON:JSON.stringify(tool.inputSchema||{},null,2),outputJSON:JSON.stringify(tool.outputSchema||{},null,2),extension:'json'}});
   set('runtime_input_fields',schemaFields(tool.inputSchema));set('runtime_output_fields',schemaFields(tool.outputSchema));
   context.handlers.window.openDialog({execution:{args:['runtime-contract']}});
  },
 };
})()

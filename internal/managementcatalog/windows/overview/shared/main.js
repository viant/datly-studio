(() => {
 const ds=id=>context.Context(id).handlers.dataSource;
 const selected=id=>ds(id).peekSelection()?.selected;
 const navigate=(section,parameters)=>context.services.hostWindow.navigate(section,parameters);
 const outcome=message=>ds('overview_state').setFormData({values:{outcome:message}});
 const record=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value[0]:value;};
 let busy=false;
 return {
  refresh:async()=>{
   if(busy)return false;busy=true;outcome('');
   try{
    const results=await Promise.allSettled([
     ds('overview_reports').fetchRecords({parameters:{limit:6,offset:0}}),
     ds('overview_connectors').fetchRecords({parameters:{limit:100,offset:0}}),
     ds('overview_namespaces').fetchRecords({parameters:{limit:500,offset:0}}),
     ds('overview_runtime').fetchRecords(),
    ]);
    const errors=results.map(result=>result.status==='rejected'?result.reason.message:'');
    const list=result=>{const value=result?.rows??result?.data??result;return Array.isArray(value)?value:value?.items||[];};
    const reports=errors[0]?[]:list(results[0].value),connectors=errors[1]?[]:list(results[1].value),namespaces=errors[2]?[]:list(results[2].value),runtime=results[3].status==='fulfilled'?record(results[3].value):null;
    ds('overview_reports').setCollection(reports);
    const active=connectors.filter(item=>item.status==='active'),tested=active.filter(item=>item.lastTestStatus==='passed');
    ds('overview_checks').setCollection([
     {id:'connectors',label:'Connectors',status:!errors[1]&&active.length>0&&tested.length===active.length?'Ready':'Review',value:errors[1]||`${tested.length}/${active.length} active connectors tested`,section:'connectors'},
     {id:'namespaces',label:'Namespaces',status:!errors[2]&&namespaces.length>0?'Ready':'Review',value:errors[2]||`${namespaces.length} governed`,section:'namespaces'},
     {id:'runtime',label:'Live runtime',status:!errors[3]&&runtime?.host?.status==='ready'?'Ready':'Review',value:errors[3]||(runtime?.host?.status==='ready'?`Ready · revision ${runtime.host.revision}`:runtime?.host?.status==='unavailable'?'Check failed':'Not checked'),section:'runtime'},
    ]);
    const issues=[];
    if(!runtime||runtime.host?.status!=='ready')issues.push({id:'runtime',title:'Live runtime could not be verified',detail:'The live HTTP and MCP host did not pass a runtime check.',section:'runtime'});
    if(errors[1])issues.push({id:'connector-catalog',title:'Connector readiness is unavailable',detail:errors[1],section:'connectors'});
    for(const connector of connectors)if(connector.status==='active'&&connector.lastTestStatus!=='passed')issues.push({id:'connector:'+connector.name,title:connector.name+' needs a connection test',detail:connector.lastTestErrorCode||'No successful probe is recorded.',section:'connectors'});
    const drafts=reports.filter(item=>item.status==='draft');if(drafts.length)issues.push({id:'drafts',title:`${drafts.length} draft ${drafts.length===1?'component':'components'} need review`,detail:drafts.slice(0,3).map(item=>item.title).join(', '),section:'components'});
    ds('overview_issues').setCollection(issues);outcome(errors.filter(Boolean).join('\n'));
   }catch(error){outcome(error.message);return false;}finally{busy=false;}
  },
  catalog:()=>navigate('components'),
  openComponent:props=>{const report=props?.item||selected('overview_reports');if(report)navigate('builder',{report});},
  openCheck:props=>{const row=props?.item||selected('overview_checks');if(row)navigate(row.section);},
  openIssue:props=>{const row=props?.item||selected('overview_issues');if(row)navigate(row.section);},
 };
})()

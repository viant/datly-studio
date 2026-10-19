import {useEffect,useState} from 'react';
import {fetchReaderDataSource} from './readerDataSources.js';
export function useReaderLookups(api,report,version,structure,enabled) {
 const [state,setState]=useState({}),[attempt,setAttempt]=useState(0);
 const key=`${report?.id}:${version?.versionNo}:${version?.sourceRevision}`;
 const supported=typeof api?.fetchMCPDataSource==='function'&&Boolean(version?.versionNo);
 useEffect(()=>{
  if(!enabled||!supported)return;
  let cancelled=false;setState({key,loading:true});
  fetchReaderDataSource(api,'definition',{reportId:report.id,versionNo:version.versionNo,discoverColumns:true}).then((result)=>{
   if(cancelled)return;
   if(result?.version?.versionNo!==version.versionNo||result?.version?.sourceRevision!==version.sourceRevision)throw new Error('The component changed while loading view and dimension lookups. Refresh the component.');
   setState({key,structure:result.structure,loading:false});
  }).catch((cause)=>{if(!cancelled)setState({key,error:cause.message,loading:false});});
  return()=>{cancelled=true;};
 },[api,key,enabled,supported,attempt]);
 const current=state.key===key?state:{};
 return {structure:current.structure||structure,loading:enabled&&supported&&(current.loading??!current.structure)&&!current.error,error:current.error||'',retry:()=>setAttempt((value)=>value+1)};
}

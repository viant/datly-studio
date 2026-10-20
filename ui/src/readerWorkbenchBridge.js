import {managementWindowTransport} from './managementWindow.js';
import {fetchReaderDataSource} from './readerDataSources.js';

// Compatibility for React embedders. Native routing renders the metadata
// workbench; this adapter contains no renderer or SDK transport implementation.
export function readerWorkbenchAPI(api) {
 const adapted=Object.create(api);
 adapted.resourceNamespace='datastudio';
 if(typeof api.fetchMCPDataSource==='function')adapted.readLookup=(name,input)=>fetchReaderDataSource(api,name,input);
 adapted.windowServices={hostWindow:{loadResource:uri=>api.readManagementResource(uri),executeResource:(metadata,request)=>managementWindowTransport(api,metadata)(request)}};
 return adapted;
}

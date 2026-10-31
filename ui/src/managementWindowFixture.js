import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {parse} from 'yaml';
export function managementWindowFixture(screenName) {
 const base=resolve(process.cwd(),`../internal/managementcatalog/windows/${screenName}/shared`);
 const {window:definition,dataSources}=parse(readFileSync(resolve(base,'main.yaml'),'utf8'));
 definition.namespace='datastudio';definition.actionAliases=['management'];definition.actions={code:readFileSync(resolve(base,'main.js'),'utf8')};
 if(screenName==='connectors') {const behavior=readFileSync(resolve(base,'behavior.js'),'utf8');definition.actions.code=`(() => {${behavior.replace('export default ', '')};return ${definition.actions.code};})()`;}
 const prefix=screenName==='connectors'?'connector':'namespace';
 if(screenName==='components') {for(const operation of ['components.get','components.create','components.update','versions.list','versions.create','versions.clone','versions.inspect','versions.apply','versions.validate','versions.builder','resources.get','resources.upsert_file','preview.execute','publications.get','publications.publish','publications.unpublish','publications.rollback'])dataSources['component_'+operation.replaceAll('.','_')]={id:'component_'+operation.replaceAll('.','_'),backend:{kind:'mcp_tool',method:'studio.sdk.'+operation},cardinality:'object',cache:{enabled:false}};}
 else if(!['audit','schema','runtime','reader-input','reader-output','reader-predicate','overview','component-catalog','skills','security','access-review','reader-workbench','reader-composition'].includes(screenName)) for(const operation of ['get','create','update',...(screenName==='connectors'?['test','activate','disable','delete']:[ 'delete','settings_get','settings_update'])])dataSources[prefix+'_'+operation]={id:prefix+'_'+operation,backend:{kind:'mcp_tool',method:`studio.sdk.${screenName}.${operation}`},cache:{enabled:false},cardinality:'object'};
 definition.dataSource=dataSources;return definition;
}
export function managementResourceFixture(name) {
 const definition=managementWindowFixture(name), dataSources=definition.dataSource;
 return {schemaVersion:2,format:'window.bundle',targets:[{variant:'default'}],variants:{default:{window:{...definition,dataSource:Object.fromEntries(Object.entries(dataSources).map(([id,value])=>[id,{cardinality:value.cardinality}]))},dataSources}}};
}

// Local visual fixture for the actual namespace window; no server data is changed.
import React from 'react';
import {createRoot} from 'react-dom/client';
import {parse} from 'yaml';
import '@blueprintjs/core/lib/css/blueprint.css';
import '../src/studio.css';
import ManagementWindowRuntime from '../src/ManagementWindowRuntime.jsx';
import source from '../../internal/managementcatalog/windows/namespaces/shared/main.yaml?raw';
import code from '../../internal/managementcatalog/windows/namespaces/shared/main.js?raw';

const {window:definition,dataSources}=parse(source);
definition.namespace='datastudio';
definition.actionAliases=['management'];
definition.actions={code};
for(const operation of ['get','create','update','delete'])dataSources['namespace_'+operation]={id:'namespace_'+operation,backend:{kind:'mcp_tool',method:`studio.sdk.namespaces.${operation}`},cache:{enabled:false},cardinality:'object'};
const resource={schemaVersion:3,targets:[{variant:'default'}],variants:{default:{window:{...definition,dataSource:Object.fromEntries(Object.entries(dataSources).map(([id,value])=>[id,{cardinality:value.cardinality}]))},dataSources}}};
let saved={name:'forecasting',title:'Forecasting',description:'Planning workspace',ownerId:'fixture-owner',status:'active',canManage:true,visibility:'private',allowedRoles:['FORECAST_READER'],managerRoles:['FORECAST_MANAGER'],mcpEnabled:false,mcpPort:0,etag:1};
const api={
 readManagementResource:async()=>resource,
 fetchMCPDataSource:async(source,input)=>{
  switch(source.backend.method){
   case 'studio.sdk.namespaces.list':return {items:[saved]};
   case 'studio.sdk.namespaces.get':return saved;
   case 'studio.sdk.namespaces.update':saved={...saved,...input.input,etag:saved.etag+1};return saved;
   default:throw new Error('This visual fixture only edits its in-memory namespace.');
  }
 },
};
createRoot(document.getElementById('root')).render(<ManagementWindowRuntime api={api} uri="window://datastudio/namespaces" instanceKey="namespace-settings-preview"/>);

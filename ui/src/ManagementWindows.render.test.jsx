import React from 'react';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {parse} from 'yaml';
import WindowContent from '../node_modules/forge/src/components/WindowContent.jsx';
import {SettingProvider} from '../node_modules/forge/src/core/context/Setting.jsx';
import '../node_modules/forge/src/packs/blueprint/index.jsx';
function source(screenName) {
 const base=resolve(process.cwd(),`../internal/managementcatalog/windows/${screenName}/shared`);
 const {window:definition,dataSources}=parse(readFileSync(resolve(base,'main.yaml'),'utf8'));
 definition.namespace='datastudio';definition.actionAliases=['management'];definition.actions={code:readFileSync(resolve(base,'main.js'),'utf8')};
 const prefix=screenName==='connectors'?'connector':'namespace';
 for(const operation of ['get','create','update',...(screenName==='connectors'?['test','activate']:[])])dataSources[prefix+'_'+operation]={id:prefix+'_'+operation,backend:{kind:'mcp_tool',method:`studio.sdk.${screenName}.${operation}`},cache:{enabled:false},cardinality:'object'};
 definition.dataSource=dataSources;return definition;
}
for(const [screenName,openButton,nameLabel,titleLabel] of [['connectors','New connector','Name','Description'],['namespaces','New namespace','Namespace','Display title']]) {
 test(`${screenName} window opens an editable Forge dialog and invokes the SDK create operation`,async()=>{
  const metadata=source(screenName),calls=[];
  const transport=vi.fn(async request=>{calls.push(request);return {data:request.dataSourceRef===screenName?[]:[{name:'example',etag:1}]};});
  render(<SettingProvider endpoints={{}} connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport}}><WindowContent window={{windowId:`management-${screenName}`,windowKey:screenName}} selectedMetadata={metadata}/></SettingProvider>);
  const user=userEvent.setup();await user.click(await screen.findByRole('button',{name:openButton,exact:true}));
  const name=await screen.findByLabelText(nameLabel,{exact:true});await user.type(name,'example');
  await user.type(screen.getByLabelText(titleLabel,{exact:true}),'Example');
  if(screenName==='connectors') {
   await user.click(screen.getByLabelText('YAML configuration (Driver / DSN / Secret.URL / Secret.Key)',{exact:true}));await user.paste('Driver: mysql\nDSN: "${Username}:${Password}@tcp(127.0.0.1:3306)/example"\nSecret:\n  URL: file:///tmp/mysql.enc\n  Key: blowfish://default');
   await user.click(screen.getByRole('button',{name:'Apply YAML',exact:true}));
  } else await user.click(screen.getByLabelText('Expose MCP',{exact:true}));
  await user.click(screen.getByRole('button',{name:'Save',exact:true}));
  const prefix=screenName==='connectors'?'connector':'namespace';await waitFor(()=>expect(calls.some(request=>request.dataSourceRef===prefix+'_create')).toBe(true));
  const input=calls.find(request=>request.dataSourceRef===prefix+'_create').options.inputParameters;
  expect(input.name).toBe('example');
  if(screenName==='connectors') { expect(input.dsnTemplate).toBe('${Username}:${Password}@tcp(127.0.0.1:3306)/example');expect(input.secretRef).toBe('file:///tmp/mysql.enc|blowfish://default'); }
  else {expect(input.mcpEnabled).toBe(true);expect(input.mcpPort).toBe(0);}
 });
}
test('connector window edits the saved revision, preserves blank secrets, and invokes probe and activation',async()=>{
 const metadata=source('connectors'),calls=[];
 const saved={name:'warehouse',driver:'mysql',description:'Existing',status:'draft',etag:7,secretRefDisplay:'file:///tmp/mysql.enc|blowfish://default',secretConfigured:true,options:{pool:2}};
 const transport=vi.fn(async request=>{calls.push(request);if(request.dataSourceRef==='connectors')return {data:[saved]};if(request.dataSourceRef==='connector_get')return {data:[saved]};return {data:[{...saved,status:'passed'}]};});
 render(<SettingProvider endpoints={{}} connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport}}><WindowContent window={{windowId:'management-connector-edit',windowKey:'connectors'}} selectedMetadata={metadata}/></SettingProvider>);
 const user=userEvent.setup();const row=await screen.findByText('warehouse');if(!row.closest('tr').classList.contains('selected-row'))await user.click(row);
 await user.click(screen.getByRole('button',{name:'Edit',exact:true}));
 const description=await screen.findByLabelText('Description',{exact:true});await user.clear(description);await user.type(description,'Updated');
 expect(screen.getByLabelText('Saved secret reference',{exact:true}).value).toBe(saved.secretRefDisplay);
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_update')).toBe(true));
 const input=calls.find(request=>request.dataSourceRef==='connector_update').options.inputParameters;
 expect(input).toEqual({name:'warehouse',input:{driver:'mysql',description:'Updated',options:{pool:2},etag:7}});
 await user.click(screen.getByRole('button',{name:'Test',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_test')).toBe(true));
 await user.click(screen.getByRole('button',{name:'Activate',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_activate')).toBe(true));
 expect(calls.find(request=>request.dataSourceRef==='connector_activate').options.inputParameters).toEqual({name:'warehouse',etag:7});
});

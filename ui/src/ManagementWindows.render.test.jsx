import React,{act} from 'react';
import {EditorView} from '@codemirror/view';
import {managementWindowFixture as source} from './managementWindowFixture.js';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor,within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from '../node_modules/forge/src/components/WindowContent.jsx';
import {SettingProvider} from '../node_modules/forge/src/core/context/Setting.jsx';
import '../node_modules/forge/src/packs/blueprint/index.jsx';

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
 expect(screen.getByLabelText('Secret reference',{exact:true}).value).toBe(saved.secretRefDisplay);
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_update')).toBe(true));
 const input=calls.find(request=>request.dataSourceRef==='connector_update').options.inputParameters;
 expect(input).toEqual({name:'warehouse',input:{driver:'mysql',description:'Updated',options:{pool:2},etag:7}});
 await user.click(screen.getByRole('button',{name:'Test',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_test')).toBe(true));
 await user.click(screen.getByRole('button',{name:'Activate or disable',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_activate')).toBe(true));
 expect(calls.find(request=>request.dataSourceRef==='connector_activate').options.inputParameters).toEqual({name:'warehouse',etag:7});
});

test('component window edits actual CodeMirror source with CAS then validates, previews and publishes',async()=>{
 Range.prototype.getClientRects ||= function(){return [];};Range.prototype.getBoundingClientRect ||= function(){return new DOMRect();};
 const metadata=source('components'),calls=[],namespaceId='a'.repeat(64);
 const component={id:'reader',title:'Example reader',slug:'reader',defaultConnectorName:'main',currentDraftVersion:2};
 let dql='SELECT 1',revision=7,fileSource='SELECT 5';
 const file=()=>({reportId:'reader',versionNo:2,resourceId:'source-file',namespace:'reader_sql',resourcePath:'sql/main.sql',content:fileSource,mediaType:'text/plain'});
 const version=()=>({versionNo:2,sourceRevision:revision,state:'draft',authoredDql:dql});
 const transport=vi.fn(async request=>{
  calls.push(request);const id=request.dataSourceRef,args=request.options.inputParameters;
  if(id==='component_namespaces')return {data:[{namespaceId,name:'inventory',title:'Inventory'}]};
  if(id==='component_connectors')return {data:[{name:'main',status:'active'}]};
  if(id==='components')return {data:[component]};
  if(id==='component_components_get')return {data:[component]};
  if(id==='component_versions_inspect')return {data:[{version:version(),dql,diagnostics:[]}]};
  if(id==='component_resources_get')return {data:[{files:[file()],version:version()}]};
  if(id==='component_resources_upsert_file'){expect(args.expectedSourceRevision).toBe(revision);expect(args.resourceId).toBe('source-file');fileSource=args.content;revision++;return {data:[{files:[file()],version:version()}]};}
  if(id==='component_versions_apply') {expect(args.namespaceId).toBe(namespaceId);expect(args.command.expectedSourceRevision).toBe(revision);dql=args.command.payload.authoredDql;revision++;return {data:[{version:version()}]};}
  if(id==='component_versions_validate')return {data:[{valid:true,version:version(),diagnostics:[]}]};
  if(id==='component_preview_execute')return {data:[{data:[{value:2}],diagnostics:[]}]};
  if(id==='component_publications_publish')return {data:[{status:'active',activeGeneration:4}]};
  if(id==='component_draft'||id==='component_files')return {data:[]};
  throw new Error('Unexpected '+id);
 });
 render(<SettingProvider endpoints={{}} connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport}}><WindowContent window={{windowId:'management-component-editor',windowKey:'components'}} selectedMetadata={metadata}/></SettingProvider>);
 const user=userEvent.setup();await user.click(await screen.findByRole('button',{name:'Namespace',exact:true}));await user.click(await screen.findByText('Inventory'));
 const row=await screen.findByText('Example reader');if(!row.closest('tr').classList.contains('selected-row'))await user.click(row);
 await user.click(screen.getByRole('button',{name:'Open reader',exact:true}));
 await waitFor(()=>expect(document.querySelector('.cm-content').textContent).toBe('SELECT 1'));
 const editor=EditorView.findFromDOM(document.querySelector('.cm-editor'));
 await act(async()=>editor.dispatch({changes:{from:0,to:editor.state.doc.length,insert:'SELECT 2'}}));
 await user.click(screen.getByRole('button',{name:'Save reader DQL',exact:true}));await waitFor(()=>expect(dql).toBe('SELECT 2'));
 await user.click(screen.getByRole('button',{name:'Validate',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='component_versions_validate')).toBe(true));
 await user.click(screen.getByRole('button',{name:'Run preview',exact:true}));await waitFor(()=>expect(screen.getByLabelText('Preview result',{exact:true}).value).toContain('value'));
 await user.click(screen.getByRole('button',{name:'Publish version',exact:true}));await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='component_publications_publish')).toBe(true));
 expect(calls.find(request=>request.dataSourceRef==='component_publications_publish').options.inputParameters.input.expectedSourceRevision).toBe(8);
 await user.click(await screen.findByText('reader_sql'));
 await user.click(await screen.findByText('sql',{exact:true}));
 await user.click(await screen.findByText('sql/main.sql',{exact:true}));
 await waitFor(()=>expect(editor.state.doc.toString()).toBe('SELECT 5'));
 await act(async()=>editor.dispatch({changes:{from:0,to:editor.state.doc.length,insert:'SELECT 6'}}));
 await user.click(screen.getByRole('button',{name:'Save selected file',exact:true}));
 await waitFor(()=>expect(fileSource).toBe('SELECT 6'));

});

test('native management host loads and executes the canonical connector window',async()=>{
 const {ManagementWindow}=await import('./ManagementWindow.jsx');
 const definition=source('connectors');
 const dataSources=definition.dataSource;
 const canonical={...definition,dataSource:Object.fromEntries(Object.entries(dataSources).map(([id,value])=>[id,{cardinality:value.cardinality}]))};
 const resource={schemaVersion:3,targets:[{target:{platform:'web',formFactor:'desktop'},variant:'same'}],variants:{same:{window:canonical,dataSources}}};
 const api={readManagementResource:vi.fn().mockResolvedValue(resource),invoke:vi.fn(async(operation)=>operation==='connectors.list'?{items:[]}:{name:'shared'})};
 render(<ManagementWindow api={api} uri="window://datastudio/connectors" instanceKey="native-shared-connector"/>);
 const user=userEvent.setup();
 await user.click(await screen.findByRole('button',{name:'New connector',exact:true}));
 await user.type(await screen.findByLabelText('Name',{exact:true}),'shared');
 await user.click(screen.getByRole('button',{name:'Save',exact:true}));
 await waitFor(()=>expect(api.invoke).toHaveBeenCalledWith('connectors.create',expect.objectContaining({name:'shared',driver:'mysql'})));
 expect(api.readManagementResource).toHaveBeenCalledOnce();
 expect(api.readManagementResource).toHaveBeenCalledWith('window://datastudio/connectors');
});

test('canonical connector window searches on the server and confirms revision-bound deletion',async()=>{
 const {ManagementWindow}=await import('./ManagementWindow.jsx');
 const definition=source('connectors'),dataSources=definition.dataSource;
 const resource={schemaVersion:3,targets:[{variant:'default'}],variants:{default:{window:{...definition,dataSource:Object.fromEntries(Object.entries(dataSources).map(([id,value])=>[id,{cardinality:value.cardinality}]))},dataSources}}};
 const saved={name:'warehouse',driver:'mysql',status:'disabled',etag:9};
 const calls=[];
 const api={readManagementResource:async()=>resource,invoke:async(operation,input)=>{calls.push([operation,input]);return operation==='connectors.list'?{items:[saved]}:saved;}};
 render(<ManagementWindow api={api} uri="window://datastudio/connectors" instanceKey="canonical-search-delete"/>);
 const user=userEvent.setup();
 const row=await screen.findByText('warehouse');
 await user.type(screen.getByLabelText('Find connector, driver, or owner',{exact:true}),'warehouse');
 await user.click(screen.getByRole('button',{name:'Search',exact:true}));
 await waitFor(()=>expect(calls.some(([operation,input])=>operation==='connectors.list'&&input.query==='warehouse')).toBe(true));
 const refreshed=screen.getByText('warehouse');if(!refreshed.closest('tr').classList.contains('selected-row'))await user.click(refreshed);
 await user.click(screen.getByRole('button',{name:'Delete',exact:true}));
 expect(calls.some(([operation])=>operation==='connectors.delete')).toBe(false);
 await user.click(await screen.findByRole('button',{name:'Delete connector',exact:true}));
 await waitFor(()=>expect(calls).toContainEqual(['connectors.delete',{name:'warehouse',etag:9}]));
});

test('catalog row actions address the clicked connector without prior selection',async()=>{
 const metadata=source('connectors'),calls=[];
 const rows=[{name:'first',driver:'mysql',status:'active',etag:3,lastTestStatus:'passed'},{name:'second-long-connector-name',driver:'sqlite',status:'draft',etag:8,lastTestStatus:'failed'}];
 const transport=vi.fn(async request=>{calls.push(request);if(request.dataSourceRef==='connectors')return {data:rows};if(request.dataSourceRef==='connector_get')return {data:[rows.find(row=>row.name===request.options.inputParameters.name)]};return {data:[{}]};});
 render(<SettingProvider endpoints={{}} connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport}}><WindowContent window={{windowId:'catalog-direct-row-action',windowKey:'connectors'}} selectedMetadata={metadata}/></SettingProvider>);
 const user=userEvent.setup();const row=(await screen.findByText('second-long-connector-name')).closest('tr');
 expect(row.closest('.basic-table-wrapper').classList.contains('is-catalog-presentation')).toBe(true);
 expect(document.querySelector('textarea')).toBeNull();
 await user.click(within(row).getByRole('button',{name:'Activate or disable',exact:true}));
 await waitFor(()=>expect(calls.some(request=>request.dataSourceRef==='connector_activate')).toBe(true));
 expect(calls.find(request=>request.dataSourceRef==='connector_activate').options.inputParameters).toEqual({name:'second-long-connector-name',etag:8});
 await user.click(within(row).getByRole('button',{name:'Edit',exact:true}));
 expect((await screen.findByLabelText('Name',{exact:true})).value).toBe('second-long-connector-name');
});

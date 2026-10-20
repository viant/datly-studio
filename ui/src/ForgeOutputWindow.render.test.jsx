import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/settings';
import 'forge/packs/blueprint';
import {managementWindowFixture} from './managementWindowFixture.js';
let serial=0;
function mount(outputName=''){
 const inspection={version:{versionNo:1,sourceRevision:7,state:'draft'},capabilities:{canEdit:true},structure:{declarations:outputName?[{parameter:{name:outputName,typeExpr:'string',source:{kind:'output',name:'body'}}}]:[]}};
 const calls=[],complete=vi.fn();
 const transport=vi.fn(async request=>{calls.push(request);if(request.dataSourceRef==='output_inspect')return {data:[inspection]};if(request.dataSourceRef==='output_apply')return {data:[{applied:true,inspection:{...inspection,version:{...inspection.version,sourceRevision:8}}}]};return {data:[]};});
 render(<SettingProvider connectorConfig={{window:{service:{uri:'/window'}}}} services={{dataConnectorTransport:transport,hostWindow:{parameters:{reportId:'reader',versionNo:1,outputName},complete,close:vi.fn()}}}><WindowContent window={{windowId:`forge-output-${++serial}`,windowKey:'reader-output'}} selectedMetadata={managementWindowFixture('reader-output')}/></SettingProvider>);
 return {calls,complete};
}
test('Forge output metadata creates metrics with the fully qualified Datly type',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount();
 await user.type(await screen.findByLabelText('Output name',{exact:true}),'Metrics');
 await user.click(screen.getByLabelText('Kind source',{exact:true}));await user.click(await screen.findByRole('menuitem',{name:'metrics',exact:true}));
 expect(screen.getByLabelText('Type expression',{exact:true}).value).toBe('github.com/viant/xdatly/response.Metrics');
 await user.click(screen.getByRole('button',{name:'Save output',exact:true}));await waitFor(()=>expect(complete).toHaveBeenCalled());
 expect(calls.find(item=>item.dataSourceRef==='output_apply').options.inputParameters.command).toEqual({expectedSourceRevision:7,operation:{type:'addField',field:{name:'Metrics',type:'github.com/viant/xdatly/response.Metrics',sourceKind:'output',sourceName:'metrics'}}});
});
test('Forge output removal uses the exact authored name and current revision',async()=>{
 const user=userEvent.setup(),{calls,complete}=mount('Note');await screen.findByDisplayValue('Note');
 await user.click(screen.getByRole('button',{name:'Remove output',exact:true}));await user.type(await screen.findByLabelText('Confirm output name',{exact:true}),'Note');await user.click(screen.getByRole('button',{name:'Confirm removal',exact:true}));
 await waitFor(()=>expect(complete).toHaveBeenCalled());expect(calls.find(item=>item.dataSourceRef==='output_apply').options.inputParameters.command).toEqual({expectedSourceRevision:7,operation:{type:'removeOutputField',field:{existingName:'Note'}}});
});

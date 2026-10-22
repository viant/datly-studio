import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {ResourceAccessEditor} from './ResourceAccessEditor.jsx';

const resource={kind:'window',id:'window://example/one',version:'1',tenant:'tenant'};
const ref={kind:'policy',id:'stable-viant',version:'logical',tenant:'tenant'};
const local={mode:'protected',rule:{kind:'role',value:'EDITOR'},requiredScopes:['read'],entityType:'account'};
test('assigns existing named policy by stable reference and preserves local restrictions through CAS review',async()=>{
 const user=userEvent.setup();const named={resource:ref,revision:3,displayName:'Viant users',policies:{'*':{mode:'protected',rule:{kind:'group',value:'Viant internal'}}}};
 const api={listNamedPolicies:vi.fn(async()=>[named]),getResourceAccess:vi.fn(async()=>({resource,revision:8,policies:{open:local}})),getResourceAccessContext:vi.fn(async()=>({canManage:true,choices:{role:[{id:'EDITOR',label:'Editor'}],entityTypes:['account']}})),replaceResourceAccess:vi.fn(async document=>({...document,revision:9})),namedPolicyImpact:vi.fn(async()=>[])};
 render(<ResourceAccessEditor api={api} resource={resource} actions={['open']}/>);
 await screen.findByText('Policy revision 8');await user.selectOptions(screen.getByLabelText('Access mode'),'named');await user.selectOptions(screen.getByLabelText('Named policy'),JSON.stringify(ref));
 await user.click(screen.getByRole('button',{name:'Review changes'}));expect(api.replaceResourceAccess).not.toHaveBeenCalled();expect(screen.getByText(/Named policy stable-viant/)).toBeTruthy();
 await user.click(screen.getByRole('button',{name:'Save permissions'}));await screen.findByText('Permissions saved.');
 expect(api.replaceResourceAccess.mock.calls[0][0]).toEqual({resource,revision:8,policies:{open:{...local,mode:'named',named:{resource:ref,revision:3}}}});
});
test('creates a reusable membership policy once and refreshes window assignment choices',async()=>{
 const user=userEvent.setup();const policies=[];const api={listNamedPolicies:vi.fn(async()=>[...policies]),createNamedPolicy:vi.fn(async(id,definition)=>{const result={resource:{...ref,id},revision:1,...definition};policies.push(result);return result;}),getResourceAccess:vi.fn(async()=>({resource,revision:1,policies:{open:local}})),getResourceAccessContext:vi.fn(async()=>({canManage:true,choices:{}})),namedPolicyImpact:vi.fn(async()=>[])};
 render(<ResourceAccessEditor api={api} resource={resource} actions={['open']}/>);
 await waitFor(()=>expect(screen.getByRole('button',{name:'Create named policy'}).disabled).toBe(false));await user.click(screen.getByRole('button',{name:'Create named policy'}));await user.type(screen.getByLabelText('Policy name'),'Viant users');await user.type(screen.getByLabelText('Verified membership group'),'Viant internal');await user.click(screen.getByRole('button',{name:'Review named policy'}));expect(api.createNamedPolicy).not.toHaveBeenCalled();await user.click(screen.getByRole('button',{name:'Save named policy'}));await waitFor(()=>expect(api.createNamedPolicy).toHaveBeenCalledTimes(1));
 expect(api.createNamedPolicy.mock.calls[0][1]).toEqual({displayName:'Viant users',policies:{'*':{mode:'protected',rule:{kind:'group',value:'Viant internal'}}}});
 await waitFor(()=>expect(screen.getByLabelText('Access mode').querySelector('option[value="named"]')).toBeTruthy());
});

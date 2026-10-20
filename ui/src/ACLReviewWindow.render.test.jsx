import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ACLReviewWindow, ACLReviewFrame, LiveACLReview } from './ACLReviewWindow.jsx';
import { createReviewFixture } from './aclReviewFixture.js';
import { liveACLReviewResource } from './aclReviewURL.js';
import {managementResourceFixture} from './managementWindowFixture.js';
import './ManagementWindowRuntime.jsx';
import { StudioAPI } from './studioApi.js';

test('switches viewport and scenario without contacting a live service', async () => {
  const user = userEvent.setup();
  render(<ACLReviewWindow/>);
  await user.selectOptions(screen.getByLabelText('Viewport'), 'phone');
  expect(screen.getByTitle('ACL permissions preview').style.width).toBe('390px');
  await user.selectOptions(screen.getByLabelText('Resource'), 'skill');
  await user.selectOptions(screen.getByLabelText('Scenario'), 'conflict');
  expect(screen.getByTitle('ACL permissions preview').getAttribute('src')).toContain('scenario=conflict&kind=skill');
  await user.click(screen.getByRole('button', { name: 'Reset preview' }));
  expect(screen.getByTitle('ACL permissions preview').getAttribute('src')).toContain('iteration=1');
});

test('read-only scenario uses the actual editor and disables policy changes', async () => {
  render(<ACLReviewFrame scenario="readonly" kind="skill"/>);
  await screen.findByText('You have read-only access.');
  expect(screen.queryByRole('button', { name: 'Review changes' })).toBeNull();
  expect(screen.getByLabelText('Access mode').disabled).toBe(true);
});

test('component review exposes execute without a report preview action', async () => {
  const user = userEvent.setup();
  render(<ACLReviewWindow/>);
  expect(screen.queryByRole('option', { name: 'Report' })).toBeNull();
  await user.selectOptions(screen.getByLabelText('Resource'), 'component');
  expect(screen.getByTitle('ACL permissions preview').getAttribute('src')).toContain('kind=component');
  const review = render(<ACLReviewFrame scenario="editable" kind="component"/>);
  await screen.findByText('Policy revision 7');
  expect(screen.getByRole('button', { name: /execute Protected/ })).toBeTruthy();
  expect(screen.queryByRole('button', { name: /preview Protected/ })).toBeNull();
  await user.selectOptions(screen.getByLabelText('Permission action'), 'execute');
  expect(screen.getByLabelText('Permission action').value).toBe('execute');
  expect(screen.getByLabelText('Required entity scope').value).toBe('project');
  review.unmount();
});

test('synthetic save and conflict never issue HTTP requests', async () => {
  const fetcher = vi.spyOn(globalThis, 'fetch');
  try {
    const fixture = createReviewFixture();
    const initial = await fixture.api.getResourceAccess();
    const saved = await fixture.api.replaceResourceAccess(initial);
    expect(saved.revision).toBe(initial.revision + 1);
    await expect(createReviewFixture('conflict').api.replaceResourceAccess(initial)).rejects.toMatchObject({ code: 'conflict' });
    expect(fetcher).not.toHaveBeenCalled();
  } finally { fetcher.mockRestore(); }
});

test('live review loads the same metadata resource and uses read-only policy MCP operations',async()=>{
 const resource={kind:'skill',id:'deploy',tenant:'one',version:'2'},sent=[];
 const fetcher=vi.fn(async(value,init)=>{
  const request=new Request(value,init),body=await request.json();sent.push({request,body});
  let result;
  if(body.method==='resources/read')result={contents:[{uri:'window://datastudio/access-review',mimeType:'application/json',text:JSON.stringify(managementResourceFixture('access-review'))}]};
  else if(body.params.name==='authz.sdk.policies.get')result={structuredContent:{document:{resource,revision:4,policies:{retrieve:{mode:'public'}}}}};
  else if(body.params.name==='authz.sdk.policies.context')result={structuredContent:{context:{canManage:true,choices:{}}}};
  else throw new Error('Unexpected operation '+body.params?.name);
  return new Response(JSON.stringify({jsonrpc:'2.0',id:1,result}),{status:200,headers:{'Content-Type':'application/json'}});
 });
 render(<LiveACLReview api={new StudioAPI({mode:'authenticated',apiBaseURL:'https://studio.example.com'},{fetcher})} resource={resource}/>);
 await screen.findByText('Policy revision 4');expect(screen.getByLabelText('Access mode').disabled).toBe(true);expect(screen.getByLabelText('Permission action').value).toBe('retrieve');expect(screen.queryByRole('button',{name:'Review changes'})).toBeNull();
 expect(screen.getByText('Live · read-only')).toBeTruthy();
 expect(sent.map(({body})=>body.method==='resources/read'?body.method:body.params.name)).toEqual(['resources/read','authz.sdk.policies.get','authz.sdk.policies.context']);
 for(const {request,body} of sent){expect(request.url).toBe('https://studio.example.com/v1/studio/sdk-mcp/mcp');expect(request.credentials).toBe('include');expect(request.headers.get('Authorization')).toBeNull();if(body.method==='tools/call')expect(body.params.arguments).toEqual({resource});}
});

test('live review URL requires a declared resource contract and exact identity', () => {
	expect(liveACLReviewResource(new URLSearchParams('mode=live&kind=component&id=ops&tenant=one&version=3'))).toEqual({ kind: 'component', id: 'ops', tenant: 'one', version: '3' });
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=report&id=ops&tenant=one&version=3'))).toThrow(/valid resource type/);
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=other&id=ops&tenant=one&version=3'))).toThrow(/valid resource type/);
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=component&id=&tenant=one&version=3'))).toThrow(/valid resource type/);
});

import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ACLReviewWindow, ACLReviewFrame, LiveACLReview } from './ACLReviewWindow.jsx';
import { createReviewFixture } from './aclReviewFixture.js';
import { liveACLReviewResource } from './aclReviewURL.js';
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

test('live review uses current SDK policy read-only without fixture or writes', async () => {
	const resource = { kind: 'skill', id: 'deploy', tenant: 'one', version: '2' };
	const sent = [];
	const fetcher = vi.fn(async (request) => {
		sent.push({ url: request.url, credentials: request.credentials, authorization: request.headers.get('Authorization'), body: await request.clone().text() });
		return new Response(JSON.stringify(request.url.endsWith('policies.get')
			? { document: { resource, revision: 4, policies: { retrieve: { mode: 'public' } } } }
			: { context: { canManage: true, choices: {} } }), { status: 200, headers: { 'Content-Type': 'application/json' } });
	});
	const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, { fetcher });
	render(<LiveACLReview api={api} resource={resource}/>);
	expect(await screen.findByText('Policy revision 4')).toBeTruthy();
	expect(sent.map(({ url }) => url)).toEqual([
		'https://studio.example.com/v1/authz/sdk/policies.get', 'https://studio.example.com/v1/authz/sdk/policies.context',
	]);
	for (const request of sent) {
		expect(request.credentials).toBe('include');
		expect(request.authorization).toBeNull();
		expect(JSON.parse(request.body)).toEqual({resource});
	}
	expect(screen.getByText('Live · read-only')).toBeTruthy();
	expect(screen.getByLabelText('Access mode').disabled).toBe(true);
	expect(screen.getByLabelText('Permission action').value).toBe('retrieve');
	expect(screen.queryByRole('button', { name: 'Review changes' })).toBeNull();
	expect(fetcher).toHaveBeenCalledTimes(2);
});

test('live review URL requires a declared resource contract and exact identity', () => {
	expect(liveACLReviewResource(new URLSearchParams('mode=live&kind=component&id=ops&tenant=one&version=3'))).toEqual({ kind: 'component', id: 'ops', tenant: 'one', version: '3' });
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=report&id=ops&tenant=one&version=3'))).toThrow(/valid resource type/);
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=other&id=ops&tenant=one&version=3'))).toThrow(/valid resource type/);
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=component&id=&tenant=one&version=3'))).toThrow(/valid resource type/);
});

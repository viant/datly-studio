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

test('report preview scenario shows distinct preview and execute policies', async () => {
  const user = userEvent.setup();
  render(<ACLReviewWindow/>);
  await user.selectOptions(screen.getByLabelText('Resource'), 'report');
  expect(screen.getByTitle('ACL permissions preview').getAttribute('src')).toContain('kind=report');
  const review = render(<ACLReviewFrame scenario="editable" kind="report"/>);
  await screen.findByText('Policy revision 7');
  expect(screen.getByRole('button', { name: /preview Protected/ })).toBeTruthy();
  expect(screen.getByRole('button', { name: /execute Protected/ })).toBeTruthy();
  await user.selectOptions(screen.getByLabelText('Permission action'), 'preview');
  expect(screen.getByLabelText('Permission action').value).toBe('preview');
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
	const fetcher = vi.fn(async (url) => ({ ok: true, status: 200, json: async () => url.endsWith('access.get')
		? { resource, revision: 4, policies: { retrieve: { mode: 'public' } } }
		: { canManage: true, choices: {} } }));
	const api = new StudioAPI({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }, { fetcher });
	render(<LiveACLReview api={api} resource={resource}/>);
	expect(await screen.findByText('Policy revision 4')).toBeTruthy();
	expect(fetcher.mock.calls.map(([url]) => url)).toEqual([
		'https://studio.example.com/v1/studio/sdk/access.get', 'https://studio.example.com/v1/studio/sdk/access.context',
	]);
	for (const [, request] of fetcher.mock.calls) {
		expect(request.credentials).toBe('include');
		expect(request.headers.Authorization).toBeUndefined();
		expect(JSON.parse(request.body)).toEqual(resource);
	}
	expect(screen.getByText('Live · read-only')).toBeTruthy();
	expect(screen.getByLabelText('Access mode').disabled).toBe(true);
	expect(screen.getByLabelText('Permission action').value).toBe('retrieve');
	expect(screen.queryByRole('button', { name: 'Review changes' })).toBeNull();
	expect(fetcher).toHaveBeenCalledTimes(2);
});

test('live review URL requires a declared resource contract and exact identity', () => {
	expect(liveACLReviewResource(new URLSearchParams('mode=live&kind=report&id=ops&tenant=one&version=3'))).toEqual({ kind: 'report', id: 'ops', tenant: 'one', version: '3' });
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=other&id=ops&tenant=one&version=3'))).toThrow(/valid resource type/);
	expect(() => liveACLReviewResource(new URLSearchParams('mode=live&kind=report&id=&tenant=one&version=3'))).toThrow(/valid resource type/);
});

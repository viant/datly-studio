import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ACLReviewWindow, ACLReviewFrame } from './ACLReviewWindow.jsx';
import { createReviewFixture } from './aclReviewFixture.js';

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
  expect(screen.getByRole('button', { name: 'Review changes' }).disabled).toBe(true);
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

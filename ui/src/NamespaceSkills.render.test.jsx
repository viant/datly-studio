import React from 'react';
import { expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NamespaceSkills } from './NamespaceSkills.jsx';

const namespace = { namespaceId: 'a'.repeat(64), name: 'sales', title: 'Sales', canManage: true, mcpEnabled: false };
const content = '---\nname: sales-guide\ndescription: Query sales records.\n---\nUse sales tools.';
const skill = { skillId: 'guide', name: 'sales-guide', title: 'Sales guide', description: 'Query sales records.', latestRevision: 1, publishedRevision: 0, etag: 1 };
const apiFor = (items = []) => ({
  listNamespaceSkills: vi.fn().mockResolvedValue({ items }),
  createNamespaceSkill: vi.fn().mockResolvedValue({ skill, revision: { revision: 1, bundle: { content } } }),
  getNamespaceSkill: vi.fn().mockResolvedValue({ skill, revision: { revision: 1, bundle: { content, title: 'Sales guide', files: [] } } }),
  updateNamespaceSkill: vi.fn().mockResolvedValue({ skill: { ...skill, latestRevision: 2, etag: 2 }, revision: { revision: 2, bundle: { content } } }),
  publishNamespaceSkill: vi.fn().mockResolvedValue({ skill: { ...skill, publishedRevision: 1, etag: 2 }, publicationStatus: 'active' }),
  getRuntimeStatus: vi.fn().mockRejectedValue(new Error('offline')),
});

test('creates an independent namespace draft and supporting file while MCP is disabled', async () => {
  const api = apiFor(), dirty = vi.fn(), user = userEvent.setup();
  render(<NamespaceSkills api={api} namespace={namespace} onDirtyChange={dirty}/>);
  await screen.findByText(/No namespace skills found/);
  await user.click(screen.getByRole('button', { name: 'New namespace skill' }));
  await user.type(screen.getByLabelText('Display title'), 'Sales guide');
  await user.clear(screen.getByLabelText('SKILL.md'));
  await user.type(screen.getByLabelText('SKILL.md'), content);
  await user.click(screen.getByRole('button', { name: 'Add supporting file' }));
  await user.type(screen.getByLabelText('File path 1'), 'references/query.md');
  await user.type(screen.getByLabelText('File content 1'), 'Use namespace tools.');
  await user.click(screen.getByRole('button', { name: 'Save namespace skill' }));
  await screen.findByText('Saved sales-guide, revision 1.');
  expect(api.createNamespaceSkill).toHaveBeenCalledWith({ title: 'Sales guide', content, files: [{ path: 'references/query.md', content: 'Use namespace tools.' }] });
  expect(api.getRuntimeStatus).not.toHaveBeenCalled();
  expect(api.publishNamespaceSkill).not.toHaveBeenCalled();
  expect(dirty).toHaveBeenCalledWith(true, expect.anything());
});

test('conflicting edit preserves content and requires explicit reload', async () => {
  const api = apiFor([skill]), user = userEvent.setup();
  api.updateNamespaceSkill.mockRejectedValue(Object.assign(new Error('expected revision changed'), { code: 'conflict' }));
  render(<NamespaceSkills api={api} namespace={namespace}/>);
  await user.click(await screen.findByRole('button', { name: 'Edit sales-guide' }));
  await user.type(screen.getByLabelText('SKILL.md'), '\nKeep my changes.');
  await user.click(screen.getByRole('button', { name: 'Save namespace skill' }));
  await screen.findByText('Skill changed elsewhere');
  expect(screen.getByLabelText('SKILL.md').value).toContain('Keep my changes.');
  expect(screen.getByRole('button', { name: 'Save namespace skill' }).disabled).toBe(true);
  await user.click(screen.getByRole('button', { name: 'Discard edits and reload' }));
  await waitFor(() => expect(screen.getByLabelText('SKILL.md').value).toBe(content));
});

test('publication is explicit and uses the saved revision and etag', async () => {
  const api = apiFor([skill]), user = userEvent.setup();
  const view = render(<NamespaceSkills api={api} namespace={namespace}/>);
  expect((await screen.findByRole('button', { name: 'Publish sales-guide' })).disabled).toBe(true);
  view.rerender(<NamespaceSkills api={api} namespace={{ ...namespace, mcpEnabled: true }}/>);
  await user.click(screen.getByRole('button', { name: 'Publish sales-guide' }));
  await screen.findByText('Published sales-guide, revision 1.');
  expect(api.publishNamespaceSkill).toHaveBeenCalledWith('guide', 1, 1);
});

test('uses bounded authoring pages and prevents viewers from editing namespace drafts', async () => {
  const api = apiFor(Array.from({ length: 21 }, (_, index) => ({ ...skill, skillId: `skill-${index}`, name: `guide-${index}` }))), user = userEvent.setup();
  const view = render(<NamespaceSkills api={api} namespace={namespace}/>);
  await screen.findByRole('button', { name: 'Edit guide-0' });
  expect(screen.queryByRole('button', { name: 'Edit guide-20' })).toBeNull();
  await user.click(screen.getByRole('button', { name: 'Next' }));
  await waitFor(() => expect(api.listNamespaceSkills).toHaveBeenLastCalledWith({ query: '', limit: 21, offset: 20 }));
  view.unmount();
  api.listNamespaceSkills.mockClear();
  render(<NamespaceSkills api={api} namespace={{ ...namespace, canManage: false }}/>);
  expect(screen.queryByRole('button', { name: 'New namespace skill' })).toBeNull();
  expect(api.listNamespaceSkills).not.toHaveBeenCalled();
});

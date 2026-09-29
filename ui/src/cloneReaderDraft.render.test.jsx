import { expect, test, vi } from 'vitest';
import { cloneReaderDraft } from './cloneReaderDraft.js';

test('clones DQL, files, folder and skill into one editable version', async () => {
  const calls = [];
  const api = {
    getVersion: vi.fn().mockResolvedValue({ authoringMode: 'dql', authoredDql: '#package("reader")', componentSpec: { name: 'reader' } }),
    getResources: vi.fn().mockResolvedValue({
      files: [{ resourceId: 'file', namespace: 'owner.docs', resourcePath: 'skills/guide/SKILL.md', content: '---\nname: guide\n---\n', mediaType: 'text/markdown' }],
      folders: [{ folderId: 'folder', namespace: 'owner.docs', rootPath: 'skills/guide', uriPrefix: 'skill://guide/', ordinal: 0 }],
      skills: [{ skillId: 'skill', folderId: 'folder', skillRoot: '.', ordinal: 0 }],
    }),
    createVersion: vi.fn().mockResolvedValue({ versionNo: 8, sourceRevision: 1 }),
    upsertResourceFile: vi.fn().mockImplementation(async (value) => { calls.push(['file', value]); return { version: { versionNo: 8, sourceRevision: 2 } }; }),
    upsertResourceFolder: vi.fn().mockImplementation(async (value) => { calls.push(['folder', value]); return { version: { versionNo: 8, sourceRevision: 3 } }; }),
    upsertSkillRoot: vi.fn().mockImplementation(async (value) => { calls.push(['skill', value]); return { version: { versionNo: 8, sourceRevision: 4 } }; }),
  };
  const draft = await cloneReaderDraft(api, 'reader', 4);
  expect(api.createVersion).toHaveBeenCalledWith('reader', expect.objectContaining({ authoredDql: '#package("reader")', notes: 'Draft from v4' }));
  expect(calls.map(([kind]) => kind)).toEqual(['file', 'folder', 'skill']);
  expect(calls.map(([, input]) => input.expectedSourceRevision)).toEqual([1, 2, 3]);
  expect(calls[2][1]).toEqual(expect.objectContaining({ folderId: 'folder', skillRoot: '.', versionNo: 8 }));
  expect(draft).toEqual({ versionNo: 8, sourceRevision: 4 });
});

test('refuses to create a draft when source DQL is unavailable', async () => {
  const api = { getVersion: vi.fn().mockResolvedValue({}), getResources: vi.fn().mockResolvedValue({}), createVersion: vi.fn() };
  await expect(cloneReaderDraft(api, 'reader', 4)).rejects.toThrow('no DQL source');
  expect(api.createVersion).not.toHaveBeenCalled();
});

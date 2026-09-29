import { expect, test, vi } from 'vitest';
import { cloneReaderDraft } from './cloneReaderDraft.js';

test('clones the exact source through one server-owned operation', async () => {
  const api = {
    getVersion: vi.fn().mockResolvedValue({ sourceRevision: 7 }),
    cloneVersion: vi.fn().mockResolvedValue({ versionNo: 8, sourceRevision: 4 }),
    createVersion: vi.fn(), upsertResourceFile: vi.fn(), upsertResourceFolder: vi.fn(), upsertSkillRoot: vi.fn(),
  };
  expect(await cloneReaderDraft(api, 'reader', 4)).toEqual({ versionNo: 8, sourceRevision: 4 });
  expect(api.cloneVersion).toHaveBeenCalledWith('reader', 4, 7);
  for (const method of ['createVersion', 'upsertResourceFile', 'upsertResourceFolder', 'upsertSkillRoot']) expect(api[method]).not.toHaveBeenCalled();
});

test('requires the source revision before requesting a clone', async () => {
  const api = { getVersion: vi.fn().mockResolvedValue({}), cloneVersion: vi.fn() };
  await expect(cloneReaderDraft(api, 'reader', 4)).rejects.toThrow('Source version revision is unavailable');
  expect(api.cloneVersion).not.toHaveBeenCalled();
});

test('reports a server clone failure without inventing a partial draft', async () => {
  const error = new Error('Source version changed. Reload before cloning.');
  const api = { getVersion: vi.fn().mockResolvedValue({ sourceRevision: 7 }), cloneVersion: vi.fn().mockRejectedValue(error) };
  await expect(cloneReaderDraft(api, 'reader', 4)).rejects.toBe(error);
  expect(error.partialDraftVersionNo).toBeUndefined();
});

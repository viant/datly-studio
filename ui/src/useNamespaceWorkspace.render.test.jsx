import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { useNamespaceWorkspace } from './useNamespaceWorkspace.js';

const a = 'a'.repeat(64), b = 'b'.repeat(64);
const key = 'studio:namespace:http://studio.test:owner';
const rows = [{ namespaceId: a, name: 'alpha', status: 'active' }, { namespaceId: b, name: 'beta', status: 'active' }];
function client(items = rows) {
  return { config: { apiBaseURL: 'http://studio.test' }, setNamespace: vi.fn(), setNamespaceBlocked: vi.fn(), listNamespaces: vi.fn().mockResolvedValue({ items }) };
}
beforeEach(() => localStorage.clear());
afterEach(() => vi.restoreAllMocks());

test('retries an initial directory error without losing the saved selection', async () => {
  localStorage.setItem(key, b);
  const api = client();
  api.listNamespaces.mockRejectedValueOnce(new Error('Directory unavailable'));
  const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
  await waitFor(() => expect(result.current.error).toBe('Directory unavailable'));
  await act(() => result.current.refresh());
  expect(result.current.current.namespaceId).toBe(b);
  expect(result.current.ready).toBe(true);
  expect(result.current.error).toBe('');
});

test('retries a failed cross-window reload using its intended namespace', async () => {
  const api = client();
  const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
  await waitFor(() => expect(result.current.current?.namespaceId).toBe(a));
  api.listNamespaces.mockRejectedValueOnce(new Error('Directory disconnected'));
  localStorage.setItem(key,b);
  act(() => window.dispatchEvent(new StorageEvent('storage', { key, newValue: b })));
  await waitFor(() => expect(result.current.error).toBe('Directory disconnected'));
  await act(() => result.current.refresh());
  expect(result.current.current.namespaceId).toBe(b);
  expect(result.current.ready).toBe(true);
});

test('restores a visible selection and changes the SDK and shared selection together', async () => {
  localStorage.setItem(key, b);
  const api = client();
  const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
  await waitFor(() => expect(result.current.current?.namespaceId).toBe(b));
  await act(() => result.current.select(a));
  expect(api.setNamespace).toHaveBeenLastCalledWith(a);
  expect(localStorage.getItem(key)).toBe(a);
});

test('does not fall back to an unscoped catalog when saved visibility is lost', async () => {
  localStorage.setItem(key, b);
  const api = client(rows.slice(0, 1));
  const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
  await waitFor(() => expect(result.current.ready).toBe(true));
  expect(result.current.current).toBeNull();
  expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(true);
  expect(result.current.error).toContain('no longer available');
});

test('blocks switching while another window has an active editor', async () => {
  const api = client();
  const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
  await waitFor(() => expect(result.current.current?.namespaceId).toBe(a));
  localStorage.setItem(`${key}:edit:other-window`, String(Date.now() + 30000));
  await act(() => result.current.select(b));
  expect(result.current.current.namespaceId).toBe(a);
  expect(result.current.error).toContain('other Studio windows');
});

test('preserves an open editor and blocks requests after an unexpected external switch', async () => {
  const api = client();
  const { result, rerender } = renderHook(({ busy }) => useNamespaceWorkspace(api, 'owner', busy), { initialProps: { busy: true } });
  await waitFor(() => expect(result.current.current?.namespaceId).toBe(a));
  localStorage.setItem(key,b);
  act(() => window.dispatchEvent(new StorageEvent('storage', { key, newValue: b })));
  expect(result.current.current.namespaceId).toBe(a);
  expect(result.current.ready).toBe(false);
  expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(true);
  rerender({ busy: false });
  await waitFor(() => expect(result.current.current?.namespaceId).toBe(b));
  expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(false);
});

for (const operation of ['key', 'setItem']) {
  test(`keeps the current namespace when storage ${operation} fails and retries after recovery`, async () => {
    const api = client();
    const { result } = renderHook(() => useNamespaceWorkspace(api, 'owner', false));
    await waitFor(() => expect(result.current.current?.namespaceId).toBe(a));
    api.setNamespace.mockClear();
    const failure = vi.spyOn(Storage.prototype, operation).mockImplementation(() => { throw new DOMException('Storage unavailable', 'SecurityError'); });
    await act(() => result.current.select(b));
    expect(result.current.current.namespaceId).toBe(a);
    expect(result.current.ready).toBe(true);
    expect(result.current.error).toContain('then select the namespace again');
    expect(result.current.canRetry).toBe(false);
    expect(api.setNamespace).not.toHaveBeenCalled();
    expect(localStorage.getItem(key)).toBe(a);
    failure.mockRestore();
    await act(() => result.current.select(b));
    expect(result.current.current.namespaceId).toBe(b);
    expect(result.current.error).toBe('');
    expect(api.setNamespace).toHaveBeenLastCalledWith(b);
    expect(result.current.canRetry).toBe(true);
    expect(localStorage.getItem(key)).toBe(b);
  });
}

for (const eventName of ['focus', 'pageshow', 'visibilitychange']) {
  test(`reconciles a missed namespace switch on ${eventName} without discarding an editor`, async () => {
    const api = client();
    const { result, rerender } = renderHook(({busy}) => useNamespaceWorkspace(api, 'owner', busy), {initialProps:{busy:true}});
    await waitFor(() => expect(result.current.current?.namespaceId).toBe(a));
    localStorage.setItem(key,b);
    act(() => (eventName === 'visibilitychange' ? document : window).dispatchEvent(new Event(eventName)));
    expect(result.current.current.namespaceId).toBe(a);
    expect(result.current.ready).toBe(false);
    expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(true);
    rerender({busy:false});
    await waitFor(() => expect(result.current.current?.namespaceId).toBe(b));
    expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(false);
  });
}

test('blocks resumed requests when shared selection storage cannot be read', async () => {
 const api=client();const {result}=renderHook(()=>useNamespaceWorkspace(api,'owner',false));
 await waitFor(()=>expect(result.current.current?.namespaceId).toBe(a));
 const failure=vi.spyOn(Storage.prototype,'getItem').mockImplementation(()=>{throw new DOMException('Unavailable','SecurityError');});
 act(()=>window.dispatchEvent(new Event('focus')));
 expect(result.current.current.namespaceId).toBe(a);expect(result.current.ready).toBe(false);
 expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(true);
 failure.mockRestore();await act(()=>result.current.refresh());
 expect(result.current.ready).toBe(true);expect(result.current.current.namespaceId).toBe(a);
});

for (const saved of [null,'invalid']) {
 test(`allows valid reselection after resumed shared selection is ${saved}`, async()=>{
  const api=client();const {result}=renderHook(()=>useNamespaceWorkspace(api,'owner',false));
  await waitFor(()=>expect(result.current.current?.namespaceId).toBe(a));
  if(saved===null)localStorage.removeItem(key);else localStorage.setItem(key,saved);
  act(()=>window.dispatchEvent(new Event('focus')));
  expect(result.current.ready).toBe(true);expect(result.current.error).toContain('choose a namespace');
  expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(true);
  await act(()=>result.current.select(b));
  expect(result.current.current.namespaceId).toBe(b);expect(result.current.error).toBe('');
  expect(api.setNamespaceBlocked).toHaveBeenLastCalledWith(false);
 });
}

import { useEffect, useRef, useState } from 'react';

const validID = (value) => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

// Selection is navigation state only. The SDK independently verifies visibility
// and ownership on every request carrying X-Studio-Namespace.
export function useNamespaceWorkspace(api, subject, busy) {
  const enabled = typeof api.setNamespace === 'function';
  const [state, setState] = useState({ ready: !enabled, items: [], current: null, error: '' });
  const busyRef = useRef(busy);
  const reloadRef = useRef(null);
  const pendingRef = useRef(null);
  const intendedRef = useRef(null);
  const loadRevision = useRef(0);
  busyRef.current = busy;
  const windowID = useRef(globalThis.crypto.randomUUID());
  const key = `studio:namespace:${api.config?.apiBaseURL || ''}:${subject || ''}`;
  const lockKey = `${key}:edit:${windowID.current}`;

  useEffect(() => {
    if (!enabled) return;
    if (!busy && pendingRef.current && reloadRef.current) {
      const selected = pendingRef.current;
      pendingRef.current = null;
      reloadRef.current(selected).catch((cause) => setState((previous) => ({ ...previous, error: cause.message })));
    }
    const heartbeat = () => {
      try {
        if (busyRef.current) localStorage.setItem(lockKey, String(Date.now() + 30000));
        else localStorage.removeItem(lockKey);
      } catch { /* Storage may be unavailable; same-window guards still apply. */ }
    };
    heartbeat();
    const timer = setInterval(heartbeat, 10000);
    window.addEventListener('beforeunload', heartbeat);
    window.addEventListener('focus', heartbeat);
    window.addEventListener('pageshow', heartbeat);
    document.addEventListener('visibilitychange', heartbeat);
    return () => { clearInterval(timer); window.removeEventListener('beforeunload', heartbeat); window.removeEventListener('focus', heartbeat); window.removeEventListener('pageshow', heartbeat); document.removeEventListener('visibilitychange', heartbeat); try { localStorage.removeItem(lockKey); } catch {} };
  }, [enabled, lockKey, busy]);

  useEffect(() => {
    if (!enabled) return;
    api.setNamespaceBlocked?.(true);
    let cancelled = false;
    const load = async (selected) => {
      intendedRef.current = selected;
      const revision = ++loadRevision.current;
      const items = [];
      for (let offset = 0; ; offset += 100) {
        const page = await api.listNamespaces({ status: 'active', limit: 100, offset });
        if (cancelled || revision !== loadRevision.current) return;
        const rows = page?.items || [];
        items.push(...rows.filter((item) => validID(item.namespaceId)));
        if (rows.length < 100) break;
      }
      if (cancelled) return;
      const current = items.find((item) => item.namespaceId === selected) || (selected ? null : items[0]) || null;
      if (!selected && current) { try { localStorage.setItem(key, current.namespaceId); } catch {} }
      intendedRef.current = current?.namespaceId || selected || null;
      api.setNamespace(current?.namespaceId || null);
      api.setNamespaceBlocked?.(!current);
      setState({ ready: true, items, current, error: selected && !current ? 'The selected namespace is no longer available. Choose another namespace.' : '' });
    };
    reloadRef.current = load;
    let saved;
    try { saved = localStorage.getItem(key); } catch {}
    load(validID(saved) ? saved : null).catch((cause) => !cancelled && setState((previous) => ({ ...previous, ready: true, error: cause.message })));
    const changed = (event) => {
      if (event.key !== key || !validID(event.newValue)) return;
      intendedRef.current = event.newValue;
      if (busyRef.current) {
        // Do not discard an open editor on an unexpected concurrent change.
        // Invalidate requests so its old resource cannot be saved in a new scope.
        api.setNamespaceBlocked?.(true);
        pendingRef.current = event.newValue;
        setState((previous) => ({ ...previous, ready: false, error: 'Another window changed the namespace. Close this editor and reload before continuing.' }));
        return;
      }
      setState((previous) => ({ ...previous, ready: false }));
      api.setNamespaceBlocked?.(true);
      load(event.newValue).catch((cause) => !cancelled && setState((previous) => ({ ...previous, ready: false, error: cause.message })));
    };
    const resumed = () => {
      if (document.visibilityState === 'hidden') return;
      let saved;
      try { saved = localStorage.getItem(key); }
      catch {
        api.setNamespaceBlocked?.(true);
        setState((previous) => ({ ...previous, ready: false, error: 'Namespace synchronization is unavailable. Allow browser storage, then retry namespaces.' }));
        return;
      }
      if (saved === intendedRef.current) return;
      if (!validID(saved)) {
        api.setNamespaceBlocked?.(true);
        setState((previous) => ({ ...previous, ready: true, error: 'The shared namespace selection is unavailable. Close any editor and choose a namespace.' }));
        return;
      }
      changed({ key, newValue: saved });
    };
    window.addEventListener('storage', changed);
    window.addEventListener('focus', resumed);
    window.addEventListener('pageshow', resumed);
    document.addEventListener('visibilitychange', resumed);
    return () => { cancelled = true; window.removeEventListener('storage', changed); window.removeEventListener('focus', resumed); window.removeEventListener('pageshow', resumed); document.removeEventListener('visibilitychange', resumed); };
  }, [api, enabled, key]);

  const select = async (id) => {
    const change = () => {
      if (busyRef.current) { setState((previous) => ({ ...previous, error: 'Close the open editor or dialog before switching namespaces.' })); return; }
      try {
        for (let index = 0; index < localStorage.length; index++) {
          const name = localStorage.key(index);
          if (name?.startsWith(`${key}:edit:`) && Number(localStorage.getItem(name)) > Date.now()) {
            setState((previous) => ({ ...previous, error: 'Close editors and dialogs in other Studio windows before switching namespaces.' }));
            return;
          }
        }
      } catch {
        setState((previous) => ({ ...previous, synchronizationError: true, error: 'Namespace synchronization is unavailable. Allow browser storage, then select the namespace again.' }));
        return;
      }
      const current = state.items.find((item) => item.namespaceId === id);
      if (!current) return;
      // Persist before changing request scope. A failed write must leave every
      // local view and in-flight request pinned to the existing workspace.
      try { localStorage.setItem(key, id); }
      catch {
        setState((previous) => ({ ...previous, synchronizationError: true, error: 'Namespace synchronization is unavailable. Allow browser storage, then select the namespace again.' }));
        return;
      }
      ++loadRevision.current;
      intendedRef.current = id;
      api.setNamespace(id);
      api.setNamespaceBlocked?.(false);
      setState((previous) => ({ ...previous, ready: true, current, synchronizationError: false, error: '' }));
    };
    if (globalThis.navigator?.locks) await navigator.locks.request(key, change);
    else change();
  };
  const refresh = async () => {
    if (!reloadRef.current) return;
    api.setNamespaceBlocked?.(true);
    setState((previous) => ({ ...previous, ready: false, error: '' }));
    try {
      const saved = localStorage.getItem(key);
      if (validID(saved)) intendedRef.current = saved;
      await reloadRef.current(intendedRef.current);
    }
    catch (cause) { setState((previous) => ({ ...previous, ready: false, error: cause.message })); }
  };
  return { ...state, enabled, select, refresh, canRetry: !busy && !state.synchronizationError };
}

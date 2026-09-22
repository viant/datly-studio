import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

globalThis.ResizeObserver ??= class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

globalThis.matchMedia ??= () => ({
  matches: false,
  addEventListener() {},
  removeEventListener() {},
});

afterEach(cleanup);

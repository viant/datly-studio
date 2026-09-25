import assert from 'node:assert/strict';
import test from 'node:test';
import { validateConfig } from './config.js';

test('development mode requires a loopback API and explicit subject', () => {
  assert.deepEqual(validateConfig({
    mode: 'development', apiBaseURL: 'http://127.0.0.1:8080/', development: { subject: 'dev-user' },
  }), { mode: 'development', apiBaseURL: 'http://127.0.0.1:8080', development: { subject: 'dev-user' } });
  assert.throws(() => validateConfig({ mode: 'development', apiBaseURL: 'https://studio.example.com', development: { subject: 'dev-user' } }));
});

test('authenticated mode exposes only public BFF session routes', () => {
  const config = validateConfig({
    mode: 'authenticated', apiBaseURL: 'https://studio.example.com',
    authentication: { mode: 'bff' },
  });
  assert.deepEqual(config.authentication, { mode: 'bff', mePath: '/v1/studio/auth/me', loginPath: '/v1/studio/auth/login' });
  assert.equal(validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }).authentication.mode, 'bff');
  assert.equal(validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', brand: 'Acme Portal' }).brand, 'Acme Portal');
  assert.throws(() => validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', brand: 'bad\nname' }));
});

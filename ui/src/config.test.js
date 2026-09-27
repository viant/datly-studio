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
  assert.throws(() => validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com' }), /authentication.mode/);
  assert.equal(validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', authentication: { mode: 'bff' }, brand: 'Acme Portal' }).brand, 'Acme Portal');
  assert.throws(() => validateConfig({ mode: 'authenticated', apiBaseURL: 'https://studio.example.com', authentication: { mode: 'bff' }, brand: 'bad\nname' }));
});

test('identity-token mode separates auth, static Datly, and dynamic MCP origins', () => {
  const config = validateConfig({ mode: 'authenticated', apiBaseURL: 'https://static.example.com/',
    mcpBaseURL: 'https://dynamic.example.com/', authentication: { mode: 'identity-token', authBaseURL: 'https://studio.example.com/' } });
  assert.deepEqual(config.authentication, { mode: 'identity-token', authBaseURL: 'https://studio.example.com',
    loginPath: '/v1/studio/auth/login', tokenPath: '/v1/studio/auth/token' });
  assert.equal(config.apiBaseURL, 'https://static.example.com');
  assert.equal(config.mcpBaseURL, 'https://dynamic.example.com');
  assert.throws(() => validateConfig({ mode: 'authenticated', apiBaseURL: 'https://static.example.com',
    authentication: { mode: 'identity-token', authBaseURL: 'https://studio.example.com' } }), /mcpBaseURL/);
});

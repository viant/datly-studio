import assert from 'node:assert/strict';
import test from 'node:test';
import { sessionURL } from './session.js';

test('authenticated session routes remain relative to the configured BFF origin', () => {
  const config={apiBaseURL:'https://studio.example.com/base',authentication:{mePath:'/v1/studio/auth/me',loginPath:'/v1/studio/auth/login'}};
  assert.equal(sessionURL(config,config.authentication.mePath),'https://studio.example.com/v1/studio/auth/me');
  assert.equal(sessionURL(config,config.authentication.loginPath),'https://studio.example.com/v1/studio/auth/login');
});

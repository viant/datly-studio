import assert from 'node:assert/strict';
import test from 'node:test';
import { sessionURL, loginURL } from './session.js';

test('authenticated session routes remain relative to the configured BFF origin', () => {
  const config={apiBaseURL:'https://studio.example.com/base',authentication:{mePath:'/v1/studio/auth/me',loginPath:'/v1/studio/auth/login'}};
  assert.equal(sessionURL(config,config.authentication.mePath),'https://studio.example.com/v1/studio/auth/me');
  assert.equal(sessionURL(config,config.authentication.loginPath),'https://studio.example.com/v1/studio/auth/login');
});

test('login carries the current browser path, query and fragment without changing auth origin', () => {
  const config={apiBaseURL:'https://studio.example.com'};
  const result=new URL(loginURL(config,'/v1/studio/auth/login?prompt=login',{pathname:'/manage',search:'?section=components&query=a%20b',hash:'#catalog'}));
  assert.equal(result.origin,'https://studio.example.com');
  assert.equal(result.searchParams.get('returnTo'),'/manage?section=components&query=a%20b#catalog');
  assert.equal(result.searchParams.get('prompt'),'login');
  assert.equal(loginURL(config,'/v1/studio/auth/login',{pathname:'/',search:'',hash:''}),'https://studio.example.com/v1/studio/auth/login');
});

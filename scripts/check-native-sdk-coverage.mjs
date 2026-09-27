#!/usr/bin/env node
import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const sdkDir = path.join(root, 'sdk');
const operations = new Set();
const operationPattern = /\bOperation[A-Za-z0-9_]+\s*=\s*"([^"]+)"/g;
for (const name of readdirSync(sdkDir)) {
  if (!name.endsWith('.go') || name.endsWith('_test.go')) continue;
  for (const match of readFileSync(path.join(sdkDir, name), 'utf8').matchAll(operationPattern)) operations.add(match[1]);
}
for (const match of readFileSync(path.join(sdkDir, 'access/transport.go'), 'utf8').matchAll(operationPattern)) {
  operations.add(match[1]);
}

const document = JSON.parse(readFileSync(path.join(sdkDir, 'openapi/studio.json'), 'utf8'));
const prefix = '/v1/studio/sdk/';
const native = new Set();
const malformed = [];
for (const [route, methods] of Object.entries(document.paths || {})) {
  if (!route.startsWith(prefix) || !methods?.post) malformed.push(route);
  else native.add(route.slice(prefix.length));
}
const missing = [...operations].filter((operation) => !native.has(operation)).sort();
const unknown = [...native].filter((operation) => !operations.has(operation)).sort();
console.log(`Native Studio SDK OpenAPI routes: ${operations.size - missing.length}/${operations.size}`);
if (!operations.size) console.error('No SDK operation constants were found.');
if (missing.length) console.error(`Missing native SDK OpenAPI routes:\n${missing.map((value) => `  ${value}`).join('\n')}`);
if (unknown.length) console.error(`Generated routes without an SDK operation:\n${unknown.map((value) => `  ${value}`).join('\n')}`);
if (malformed.length) console.error(`Generated paths without a native POST SDK route:\n${malformed.map((value) => `  ${value}`).join('\n')}`);
if (!operations.size || missing.length || unknown.length || malformed.length) process.exitCode = 1;

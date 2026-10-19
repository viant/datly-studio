import test from 'node:test';
import assert from 'node:assert/strict';
import {downloadBytes} from './downloadBytes.js';
test('native byte arrays and base64 preserve binary ZIP content',()=>{
 const bytes=[80,75,3,4,0,128,255];
 assert.deepEqual([...downloadBytes(bytes)],bytes);
 assert.deepEqual([...downloadBytes(Buffer.from(bytes).toString('base64'))],bytes);
});
test('rejects malformed payloads without coercing or truncating bytes',()=>{
 for(const value of [[-1],[256],[1.5],['80'],null,{},'not base64!']) assert.throws(()=>downloadBytes(value));
});

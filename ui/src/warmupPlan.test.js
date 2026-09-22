import assert from 'node:assert/strict';
import test from 'node:test';
import { parseWarmupCases, validateWarmupCases, warmupCaseCount, warmupCaseExpression } from './warmupPlan.js';

test('warmup planner computes the complete Cartesian product', () => {
  const cases=parseWarmupCases('Period=today,yesterday\nGranularity=hour,day\nTenant=1,2,3,4');
  assert.equal(warmupCaseCount(cases),16);
  assert.equal(warmupCaseExpression(cases),'2 Period × 2 Granularity × 4 Tenant');
  assert.deepEqual(validateWarmupCases(cases),[]);
});

test('warmup planner rejects malformed, empty, and duplicate dimensions', () => {
  assert.match(validateWarmupCases(parseWarmupCases('Period'))[0],/Name=value/);
  assert.match(validateWarmupCases(parseWarmupCases('Period='))[0],/at least one/);
  assert.match(validateWarmupCases(parseWarmupCases('Period=today\nperiod=tomorrow'))[0],/duplicated/);
});

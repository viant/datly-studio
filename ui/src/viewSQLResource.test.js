import { test } from 'node:test';
import assert from 'node:assert/strict';
import { viewSQLResource } from './viewSQLResource.js';
const inspection = {structure:{component:{rootView:{name:'forecasts',source:{sql:'SELECT forecasts.* FROM (${embed:sql/forecasting.sql}) forecasts'}}}}};
test('embedded SQL resolves its actual resource, preserving authored text',()=>{
 const file={resourcePath:'sql/forecasting.sql',namespace:'actual',content:'SELECT publisher_id FROM fixture'};
 assert.deepEqual(viewSQLResource(inspection,'forecasts',[file]),{sql:file.content,file,path:file.resourcePath});
});
test('missing embedded SQL never falls back to a wrapper',()=>{
 const result=viewSQLResource(inspection,'forecasts',[]);
 assert.equal(result.sql,''); assert.match(result.error,/unavailable/);
});

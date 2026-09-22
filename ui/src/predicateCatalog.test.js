import assert from 'node:assert/strict';
import test from 'node:test';
import {filterPredicateGroups,filterPredicateRows,predicateGroups,predicatePage,predicateRows} from './predicateCatalog.js';

test('large predicate catalogs filter and paginate deterministically',()=>{
  const declarations=Array.from({length:240},(_,index)=>({parameter:{name:`${index%2?'Exclude':'Include'}Field${index}`,source:{kind:'form',name:`field_${index}`},typeExpr:'[]int'},predicates:[{ordinal:0,predicate:{group:0,name:index%3?'in':'handler',args:index%3?['v',`field_${index}`]:[`*predicate.Handler${index}`]}}]}));
  const rows=predicateRows({declarations,views:[{name:'forecasting'}],predicateExpansions:[{group:0,view:'forecasting'}]});
  assert.equal(rows.length,240);
  assert.equal(predicatePage(rows,1,25).pages,10);
  assert.equal(predicatePage(rows,10,25).items.length,15);
  assert.equal(filterPredicateRows(rows,{kind:'handler'}).length,80);
  assert.equal(filterPredicateRows(rows,{family:'include'}).length,120);
  assert.equal(filterPredicateRows(rows,{query:'field_239'}).length,1);
  assert.equal(filterPredicateRows(rows,{query:'forecasting'}).length,240);
  assert.equal(filterPredicateRows(rows,{view:'forecasting'}).length,240);
  assert.equal(filterPredicateRows(rows,{view:'missing'}).length,0);
  assert.equal(rows[0].groupOperator,'AND');
});

test('predicate rows preserve every compiled expansion view for a shared group',()=>{
  const structure={declarations:[{parameter:{name:'IDs',source:{kind:'query',name:'id'}},predicates:[{ordinal:0,predicate:{group:7,name:'in',args:['v','id']}}]}],views:[{name:'root'}],predicateExpansions:[{group:7,view:'root',operator:'OR'},{group:7,view:'child',operator:'OR'}]};
  const [row]=predicateRows(structure);
  assert.deepEqual(row.expansionViews,['root','child']);
  assert.equal(row.groupOperator,'OR');
  assert.equal(filterPredicateRows([row],{view:'child'}).length,1);
  assert.equal(filterPredicateRows([row],{query:'child'}).length,1);
});

test('predicate groups preserve per-view composition and complete shared scope',()=>{
  const structure={declarations:[{parameter:{name:'IDs',source:{kind:'query',name:'id'}},predicates:[{ordinal:0,predicate:{group:7,name:'in',args:['v','id']}}]}],views:[{name:'root'}],predicateExpansions:[{group:7,view:'root',operator:'OR'},{group:7,view:'child',operator:'OR'}]};
  const groups=predicateGroups(structure);
  assert.equal(groups.length,1);
  assert.deepEqual(groups[0].views,['root','child']);
  assert.equal(groups[0].predicates[0].field,'IDs');
  assert.equal(filterPredicateGroups(groups,{view:'child'}).length,1);
  assert.equal(filterPredicateGroups(groups,{query:'IDs'}).length,1);
});

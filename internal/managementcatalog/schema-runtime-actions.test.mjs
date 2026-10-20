import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

const root=new URL('./windows/',import.meta.url);
function harness(name){
 const calls=[];
 const sources=new Map();
 const source=id=>{
  if(!sources.has(id))sources.set(id,{
   form:{},selection:null,collection:[],
   getFormData(){return this.form},
   setFormData({values}){this.form=values},
   setFormField({item,value}){this.form[item.dataField]=value},
   setInputParameters(input){this.input=input;calls.push({id,operation:'parameters',input})},
   async fetchCollection(){calls.push({id,operation:'collection'});return {rows:[]}},
   async fetchRecords(options){this.lastOptions=options;calls.push({id,operation:'records'});return this.record||{rows:[]}},
   setCollection(rows){this.collection=rows},
   peekSelection(){return {selected:this.selection}},
  });
  return sources.get(id);
 };
 const context={Context:id=>({handlers:{dataSource:source(id)}})};
 const code=fs.readFileSync(new URL(`${name}/shared/main.js`,root),'utf8');
 return {actions:vm.runInNewContext(code,{context}),source,calls};
}

test('schema browser passes selected connector, schema, and table to SDK tools',async()=>{
 const h=harness('schema');
 h.source('schema_state').form={query:'report'};
 h.source('schema_connectors').selection={name:'ar_metadata'};
 h.source('schema_names').selection={name:'adelphic_ar'};
 h.source('schema_tables').selection={name:'report_suite'};
 h.source('schema_table').record={rows:[{columns:[{name:'id',type:'bigint'}]}]};
 await h.actions.initialize();
 await h.actions.loadSchemas();
 assert.equal(h.source('schema_names').input.name,'ar_metadata');
 await h.actions.loadTables();
 assert.equal(h.source('schema_tables').input['input.schema'],'adelphic_ar');
 assert.equal(h.source('schema_tables').input['input.limit'],25);
 await h.actions.inspectTable();
 assert.equal(h.source('schema_table').lastOptions.parameters['input.table'],'report_suite');
 assert.equal(h.source('schema_columns').collection[0].name,'id');
});

test('runtime screen reads status and pages generations through SDK MCP',async()=>{
 const h=harness('runtime');
 h.source('runtime_status').record={rows:[{status:'ready',activeGeneration:45,reportCount:33,host:{status:'ready'}}]};
 await h.actions.refreshRuntime();
 assert.equal(h.source('runtime_state').form.activeGeneration,45);
 assert.equal(h.source('runtime_state').form.hostStatus,'ready');
 assert.equal(h.source('runtime_generations').input.Limit,25);
 await h.actions.nextGenerations();
 assert.equal(h.source('runtime_generations').input.Offset,25);
});

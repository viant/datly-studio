(() => {
 const ds = id => context.Context(id).handlers.dataSource;
 const state = () => ds('schema_state');
 const selected = id => ds(id).peekSelection()?.selected;
 let busy = false;
 const run = async action => {
  if (busy) return;
  busy = true;
  try { await action(); }
  catch (error) { state().setFormField({item:{dataField:'outcome'},value:error.message}); }
  finally { busy = false; }
 };
 const form = () => state().getFormData() || {};
 const update = values => state().setFormData({values:{...form(),...values}});
 const pageTables = async offset => {
  const {connectorName,schemaName,query}=form();
  if (!connectorName || !schemaName) throw new Error('Select a connector and schema.');
  ds('schema_tables').setInputParameters({name:connectorName,'input.schema':schemaName,'input.query':(query||'').trim(),'input.limit':25,'input.offset':offset});
  await ds('schema_tables').fetchCollection();
  update({tableOffset:offset,outcome:''});
 };
 return {
  initialize: () => run(async () => { await ds('schema_connectors').fetchCollection(); update({outcome:''}); }),
  loadSchemas: () => run(async () => {
   const connector=selected('schema_connectors');
   if (!connector) throw new Error('Select a connector.');
   update({connectorName:connector.name,schemaName:'',tableOffset:0,outcome:''});
   ds('schema_names').setInputParameters({name:connector.name,'input.query':(form().query||'').trim()});
   await ds('schema_names').fetchCollection();
   ds('schema_tables').setCollection([]);ds('schema_columns').setCollection([]);
  }),
  loadTables: () => run(async () => {
   const schema=selected('schema_names');
   if (!schema) throw new Error('Select a schema.');
   update({schemaName:schema.name,tableOffset:0,outcome:''});
   await pageTables(0);
   ds('schema_columns').setCollection([]);
  }),
  nextTables: () => run(async () => pageTables((Number(form().tableOffset)||0)+25)),
  inspectTable: () => run(async () => {
   const table=selected('schema_tables');
   if (!table) throw new Error('Select a table.');
   const {connectorName,schemaName}=form();
   const result=await ds('schema_table').fetchRecords({parameters:{name:connectorName,'input.schema':schemaName,'input.table':table.name}});
   const value=result?.rows?.[0]??result?.data?.[0]??result?.data??result;
   ds('schema_columns').setCollection(Array.isArray(value?.columns)?value.columns:[]);
   update({outcome:table.name+' columns loaded.'});
  }),
 };
})()

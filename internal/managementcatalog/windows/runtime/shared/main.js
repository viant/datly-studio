(() => {
 const ds = id => context.Context(id).handlers.dataSource;
 const state = () => ds('runtime_state');
 let busy = false;
 const form = () => state().getFormData() || {};
 const update = values => state().setFormData({values:{...form(),...values}});
 const run = async action => {
  if (busy) return;
  busy = true;
  try { await action(); }
  catch (error) { update({outcome:error.message}); }
  finally { busy = false; }
 };
 const record = result => result?.rows?.[0]??result?.data?.[0]??result?.data??result;
 const generations = async offset => {
  ds('runtime_generations').setInputParameters({Limit:25,Offset:offset});
  await ds('runtime_generations').fetchCollection();
  update({generationOffset:offset});
 };
 return {
  refreshRuntime: () => run(async () => {
   const [status] = await Promise.all([ds('runtime_status').fetchRecords(),generations(0)]);
   const value=record(status);
   update({status:value.status||'',activeGeneration:value.activeGeneration||0,reportCount:value.reportCount||0,hostStatus:value.host?.status||'',outcome:''});
  }),
  previousGenerations: () => run(async () => generations(Math.max(0,(Number(form().generationOffset)||0)-25))),
  nextGenerations: () => run(async () => generations((Number(form().generationOffset)||0)+25)),
 };
})()

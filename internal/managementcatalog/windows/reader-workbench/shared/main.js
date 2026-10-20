(() => {
 const draft=()=>context.Context('workbench_context').handlers.dataSource;
 const set=values=>draft().setFormData({values:{...draft().getFormData(),...values}});
 return {
  initialize:()=>{const params=context.windowState?.parameters||{};set({report:params.report,namespaceId:params.currentNamespace?.namespaceId||params.report?.namespaceId||'',openResources:params.report?.openResources===true,resourceAction:params.report?.resourceAction||''});},
  back:()=>{context.services.hostWindow.setDirty?.(false);context.services.hostWindow.navigate('components',{discardConfirmed:true});},
  reportUpdated:props=>set({report:props.value}),
  resourcesClosed:()=>set({openResources:false,resourceAction:''}),
 };
})()

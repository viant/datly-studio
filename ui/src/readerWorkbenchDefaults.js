// Studio-owned binding for the runtime's existing verified row scope contract.
export const readerWorkbenchDefaults = {
 routePrefix:'/v1/studio/readers/',
 rowAccessScope:{
  defaultEntity:'customer',defaultColumn:'id',
  contextField:{name:'Auth',type:'*github.com/viant/datly-studio/runtime/accesscontext.Output',sourceKind:'component',sourceName:'GET:/_studio/access/context/{{report.id}}/{{entity}}',required:true},
  idsField:{name:'AuthorizedEntityIDs',sourceKind:'param',sourceName:'Auth.Scope.IDs',required:true},
  idTypes:[
   {label:'Integer',field:{type:'[]string,[]int',codec:{name:'EntityIDs'}}},
   {label:'Text',field:{type:'[]string'}},
  ],
  predicate:{name:'in',group:99,groupOperator:'AND',applyWhenAbsent:true},
  helpText:'Set the matching required entity scope in the component’s consumption policy. The runtime denies execution when the policy cannot supply that scope.',
 },
};

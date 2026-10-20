import React,{useMemo,useId} from 'react';
import {parse} from 'yaml';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/core';
import 'forge/packs/blueprint';
import previewSource from '../../internal/managementcatalog/windows/access-preview/shared/main.yaml?raw';
import policySource from '../../internal/managementcatalog/windows/access-review/shared/main.yaml?raw';
import {ManagementWindow} from './ManagementWindow.jsx';
import {createReviewFixture} from './aclReviewFixture.js';
import './aclReview.css';

const previewMetadata=parse(previewSource).window;
const policyDefinition=parse(policySource);
export function LiveACLReview({api,resource}) {
 return <ManagementWindow api={api} uri="window://datastudio/access-review" instanceKey={'access-review:'+JSON.stringify(resource)} parameters={{resource}}/>;
}

export function ACLReviewWindow() {
 const id=useId();
 return <SettingProvider connectorConfig={{window:{service:{uri:'local:access-preview'}}}} services={{dataConnectorTransport:async()=>({data:[]})}}><WindowContent window={{windowId:'access-preview:'+id,windowKey:'access-preview'}} selectedMetadata={previewMetadata}/></SettingProvider>;
}

export function ACLReviewFrame({scenario,kind}) {
 const id=useId(),fixture=useMemo(()=>createReviewFixture(scenario,kind),[scenario,kind]);
 const metadata=useMemo(()=>{
  const value=structuredClone(policyDefinition.window);delete value.on;value.namespace='datastudio';value.windowKey='access-preview-frame';
  value.dataSource=structuredClone(policyDefinition.dataSources);
  value.dataSource.security_selection.backend.rows=[{resource:fixture.resource,readOnly:false}];
  const panel=value.view.content.containers.find(node=>node.resourcePolicyEditor);panel.resourcePolicyEditor.readOnly=false;panel.resourcePolicyEditor.actions=fixture.actions;
  if(!fixture.api.checkCurrentAccess)delete panel.resourcePolicyEditor.operations.check;
  value.view.content={id:'synthetic-access-frame',className:'acl-review-frame-content',dataSourceRef:'security_selection',fetchData:true,containers:[{id:'synthetic-notice',items:[{id:'synthetic-notice-text',type:'label',value:'Synthetic ACL preview · changes stay in this window'}]},panel]};
  return value;
 },[fixture]);
 const services=useMemo(()=>({dataConnectorTransport:async({dataSource,options})=>{
  if(dataSource.backend.kind==='inline')return {data:dataSource.backend.rows||[]};
  const input=options.inputParameters||{};
  switch(dataSource.backend.method){
   case 'authz.sdk.policies.get':return {document:await fixture.api.getResourceAccess(input.resource)};
   case 'authz.sdk.policies.context':return {context:fixture.api.getResourceAccessContext?await fixture.api.getResourceAccessContext(input.resource):{canManage:true,choices:fixture.choices||{}}};
   case 'authz.sdk.policies.replace':return {document:await fixture.api.replaceResourceAccess(input.document)};
   case 'authz.sdk.policies.create':return {document:await fixture.api.createResourceAccess(input.document)};
   case 'authz.sdk.authorization.check':{const decision=await fixture.api.checkCurrentAccess(input.resource,input.action);if(decision.effect==='deny')throw Object.assign(new Error('Synthetic denial'),{code:'forbidden'});return {decision};}
   default:throw new Error('Synthetic preview operation is not configured.');
  }
 }}),[fixture]);
 return <SettingProvider connectorConfig={{window:{service:{uri:'local:access-preview-frame'}}}} services={services}><WindowContent window={{windowId:'access-preview-frame:'+id,windowKey:'access-preview-frame'}} selectedMetadata={metadata}/></SettingProvider>;
}

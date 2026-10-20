import React,{useId,useMemo,useRef} from 'react';
import {parse} from 'yaml';
import WindowContent from 'forge/window';
import {SettingProvider} from 'forge/core';
import 'forge/packs/blueprint';
import source from '../../internal/managementcatalog/windows/authentication/shared/main.yaml?raw';

const definition=parse(source).window;
// Authentication and login destinations remain host-owned. Local metadata needs
// no authenticated resource request and receives no credentials.
export function AuthenticationWindow({state,brand='Data Studio',message,identity,login,onRetry}) {
 const id=useId(),retry=useRef(onRetry);retry.current=onRetry;
 const metadata=useMemo(()=>({...definition,namespace:'datastudio',actionAliases:['authentication'],actions:{code:'({retry:()=>context.services.authentication.retry()})'},dataSource:{authentication:{id:'authentication',cardinality:'object',backend:{kind:'inline',rows:[{state,heading:state==='loading'?(brand==='Data Studio'?'Checking Studio session':`Checking ${brand} session`):state==='error'?'Studio authentication is unavailable':`Sign in to ${brand}`,description:state==='loading'?(identity?'Restoring your in-memory identity token.':'Restoring your secure server-side session.'):state==='unauthenticated'?(identity?'Your identity token stays in this page only. Sign in to restore it.':'Your session is missing or expired. Authentication continues through the deployment identity provider; tokens are never stored in this browser.'):'',error:message||'',loginEnabled:state==='unauthenticated'&&Boolean(login),browserLoginDisabled:state==='unauthenticated'&&!login,login:login?{...login,target:'_self'}:undefined}]},cache:{enabled:false}}}}),[state,brand,message,identity,login]);
 const services=useMemo(()=>({authentication:{retry:()=>retry.current?.()},dataConnectorTransport:async({dataSource})=>({data:dataSource.backend.rows})}),[]);
 return <main><SettingProvider services={services} connectorConfig={{window:{service:{uri:'local:authentication'}}}}><WindowContent window={{windowId:'authentication:'+id,windowKey:'authentication'}} selectedMetadata={metadata}/></SettingProvider></main>;
}

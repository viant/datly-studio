import React from 'react';
import {expect,test,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {StudioApp,createStudioSDK,defineStudioExtension,loadConfig,validateConfig} from './embed.jsx';

test('standalone embeddings reuse the public Studio config loader and guards',async()=>{
 const value={mode:'development',apiBaseURL:'http://127.0.0.1:18532',development:{subject:'synthetic'}};
 const fetcher=vi.fn().mockResolvedValue({ok:true,json:async()=>value});
 expect(await loadConfig('/studio-config.json',fetcher)).toEqual(validateConfig(value));
 expect(fetcher).toHaveBeenCalledWith('/studio-config.json',{headers:{Accept:'application/json'}});
 expect(()=>validateConfig({...value,apiBaseURL:'https://remote.example'})).toThrow(/loopback|localhost/);
 await expect(loadConfig('/missing',async()=>({ok:false,status:404}))).rejects.toThrow(/404/);
});

test('installed primitive workspace shares the original Studio namespace and authoring shell',async()=>{
 localStorage.clear();
 const user=userEvent.setup();
 const rows=[
  {name:'alpha',title:'Alpha',ownerId:'owner',namespaceId:'a'.repeat(64),status:'active',canManage:false},
  {name:'beta',title:'Beta',ownerId:'owner',namespaceId:'b'.repeat(64),status:'active',canManage:true},
 ];
 const api={config:{apiBaseURL:'http://studio-extension.test'},setNamespace:vi.fn(),setNamespaceBlocked:vi.fn(),listNamespaces:vi.fn().mockResolvedValue({items:rows}),listComponents:vi.fn().mockResolvedValue({items:[]}),listConnectors:vi.fn().mockResolvedValue({items:[]}),getRuntimeStatus:vi.fn().mockResolvedValue({})};
 let lastContext;
 const sdk=createStudioSDK().register(defineStudioExtension({id:'example.primitives',label:'Resources',render:context=>{
  lastContext=context;
  return <section aria-label="Primitive resources"><h1>Primitive resources</h1><output data-testid="extension-namespace">{context.currentNamespace.name}</output></section>;
 }}));
 render(<StudioApp api={api} mode="authenticated" subject="viewer" extensions={sdk.extensions()}/>);
 await waitFor(()=>expect(api.setNamespace).toHaveBeenLastCalledWith(rows[0].namespaceId));
 expect(await screen.findByRole('heading',{name:'Overview',level:1})).toBeTruthy();
 await user.click(screen.getByText('Resources'));
 expect(await screen.findByRole('heading',{name:'Primitive resources'})).toBeTruthy();
 expect(lastContext.api).toBe(api);
 expect(lastContext.currentNamespace).toEqual(rows[0]);
 expect(lastContext.namespaceItems).toEqual(rows);
 expect(lastContext.subject).toBe('viewer');
 expect(lastContext.currentNamespace.canManage).toBe(false);
 await user.selectOptions(screen.getByRole('combobox',{name:'Current namespace'}),rows[1].namespaceId);
	// The original shell blocks changing scope while an extension/editor is
	// open. Leave that workspace before selecting another namespace.
	expect(await screen.findByText('Close the open editor or dialog before switching namespaces.')).toBeTruthy();
	expect(lastContext.currentNamespace).toEqual(rows[0]);
	await user.click(screen.getByText('Connectors'));
	await user.selectOptions(screen.getByRole('combobox',{name:'Current namespace'}),rows[1].namespaceId);
	await user.click(screen.getByText('Resources'));
 await waitFor(()=>expect(lastContext.currentNamespace).toEqual(rows[1]));
 expect(api.setNamespace).toHaveBeenLastCalledWith(rows[1].namespaceId);
 expect(screen.getByTestId('extension-namespace').textContent).toBe('beta');
 await user.click(screen.getByText('Connectors'));
 expect(await screen.findByRole('heading',{name:'Connectors',level:1})).toBeTruthy();
 expect(screen.queryByRole('heading',{name:'Primitive resources'})).toBeNull();
});

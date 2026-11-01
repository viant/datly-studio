// @vitest-environment jsdom
import {expect,it,vi} from 'vitest';
import {installStudioPerfDiagnostics} from './perfDiagnostics.js';

it('classifies direct Studio MCP metadata separately from SDK data without recording inputs',async()=>{
  const fetch=vi.fn().mockResolvedValue({status:200});
  const host={location:{href:'http://fixture/?diagnostics=timing'},fetch,getComputedStyle:window.getComputedStyle};
  const trace=installStudioPerfDiagnostics({window:host,document,performance});
  document.dispatchEvent(new Event('DOMContentLoaded'));
  document.querySelector('#studio-timing-arm').click();
  const button=document.createElement('button');button.className='studio-navigation-label';button.textContent='Catalog';document.body.append(button);button.click();
  await host.fetch('/v1/studio/sdk-mcp/mcp?token=secret',{headers:{'Mcp-Method':'resources/read'},body:'private metadata'});
  await host.fetch('/v1/studio/sdk/connectors/list?token=secret',{method:'POST',body:'private inputs'});
  const panel=document.createElement('div');panel.className='studio-workspace-panel';panel.setAttribute('role','tabpanel');panel.setAttribute('aria-label','components');
  panel.innerHTML='<div class="basic-table-wrapper"><table><tbody><tr class="row"><td>private row</td></tr></tbody></table></div>';
  document.body.append(panel);
  await new Promise(resolve=>setTimeout(resolve,0));
  const result=trace.snapshot();
  expect(result.openToMetadataEndMs).not.toBeNull();
  expect(result.openToDataEndMs).not.toBeNull();
  expect(result.events.find(entry=>entry.kind==='rows-dom').rowCount).toBe(1);
  expect(document.querySelector('#studio-timing-output').textContent).not.toMatch(/secret|private/);
});

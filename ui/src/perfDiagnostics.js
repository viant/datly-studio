import {installForgePerfDiagnostics} from 'forge/runtime/perfDiagnostics';

const methodHeader = (input,init,name) => {
  try {return new Headers(init?.headers || input?.headers).get(name) || '';}
  catch {return '';}
};
const sectionForLabel = label => ({Catalog:'components',Explorer:'schema','Audit log':'audit'})[label]
  || String(label || '').trim().toLowerCase();

// Direct Studio uses the same Forge timing collector as Steward, with its own
// MCP/SDK routes and selected workspace panel. No request body is inspected.
export function installStudioPerfDiagnostics(env=globalThis) {
  return installForgePerfDiagnostics({
    env,panelId:'studio-timing-diagnostics',armId:'studio-timing-arm',
    serverId:'studio-timing-server',outputId:'studio-timing-output',
    globalKey:'__studioPerfDiagnostics',panelTitle:'Data Studio timing diagnostics',
    classifyFetch:(path,input,init)=>{
      if (path.endsWith('/mcp')) {
        const method=methodHeader(input,init,'Mcp-Method');
        const name=methodHeader(input,init,'Mcp-Name');
        if (method==='resources/read' || method==='tools/call' && name==='resource/get') return 'metadata';
        if (method==='tools/call') return 'data';
      }
      return path.startsWith('/v1/studio/sdk/') ? 'data' : '';
    },
    resourcePath:path=>path.startsWith('/v1/studio/') || path.endsWith('/mcp'),
    clickLabel:event=>event.target?.closest?.('button.studio-navigation-label')?.textContent?.trim(),
    findVisiblePanel:(document,label)=>[...document.querySelectorAll('.studio-workspace-panel[role="tabpanel"]:not([hidden]):not([aria-hidden="true"])')]
      .find(panel=>panel.getAttribute('aria-label')===sectionForLabel(label)),
  });
}

installStudioPerfDiagnostics();

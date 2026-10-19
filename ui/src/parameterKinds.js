// Built-in request and component binding kinds. Host-defined kinds already
// present in the reader are retained so editing never silently changes them.
export const inputKinds = ['query', 'path', 'header', 'cookie', 'form', 'body', 'http_request', 'const', 'param', 'component', 'view', 'data_view', 'input', 'caller_output', 'generator', 'cache', 'http_client', 'mcp_client', 'async'];
export function parameterKinds(structure, selected = '') {
  return [...new Set([...inputKinds, ...(structure?.declarations ?? []).map(({parameter}) => parameter?.source?.kind), selected].filter((kind) => kind && kind !== 'output'))];
}
export const allowsEmptyKindSource = (kind) => ['body', 'http_request', 'input'].includes(kind);

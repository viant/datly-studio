// Adapts the shared authz Datly MCP tools to GateRequirementsEditor. The host
// supplies an authenticated MCP call function; no principal or provider URL is
// accepted from browser input.
export function gateMCPAdapter(callTool) {
  if (typeof callTool !== 'function') throw new TypeError('trusted MCP call function is required');
  const invoke = async (name, input) => {
    const result = await callTool(`authz.sdk.gates.${name}`, input);
    if (!result || typeof result !== 'object') throw new Error('Gate service returned no result');
    return result;
  };
  return {
    async checkCurrentGate(resource, action, selected = []) {
      const {decision} = await invoke('check', {resource, action, selected});
      if (!decision || typeof decision.requestId !== 'string' || !decision.requestId || decision.resourceKind !== resource?.kind || decision.resourceId !== resource?.id || decision.resourceVersion !== resource?.version || decision.action !== action || !['allow', 'deny'].includes(decision.effect) || !decision.requirementsRevision || !Number.isFinite(Date.parse(decision.validUntil)) || Date.parse(decision.validUntil) <= Date.now()) {
        throw new Error('Gate decision bindings are invalid');
      }
      return decision;
    },
    async getGateRequirements(resource, action) {
      const {document} = await invoke('get', {resource, action});
      if (!document?.revision) throw new Error('Gate document revision is missing');
      return document;
    },
    async getGateRequirementsContext(resource, action) {
      const {context} = await invoke('context', {resource, action});
      if (typeof context?.canManage !== 'boolean') throw new Error('Gate editor context is missing');
      return context;
    },
    async replaceGateRequirements({resource, action, expectedRevision, requirements}) {
      const {document} = await invoke('replace', {resource, action, expectedRevision, requirements});
      if (!document?.revision) throw new Error('Gate document revision is missing');
      return document;
    },
  };
}

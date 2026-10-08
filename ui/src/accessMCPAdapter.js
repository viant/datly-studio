// Reuses the shared authz Datly MCP SDK for ResourceAccessEditor. Host code
// supplies the authenticated MCP caller and resource/action catalog.
export function accessMCPAdapter(callTool) {
  if (typeof callTool !== 'function') throw new TypeError('trusted MCP call function is required');
  const invoke = async (name, input, field) => {
    const result = await callTool(`authz.sdk.policies.${name}`, input);
    const value = result?.[field];
    if (!value || typeof value !== 'object') throw new Error(`Policy ${field} is missing`);
    return value;
  };
  return {
    getResourceAccess(resource) { return invoke('get', {resource}, 'document'); },
    getResourceAccessContext(resource) { return invoke('context', {resource}, 'context'); },
    replaceResourceAccess(document) { return invoke('replace', {document}, 'document'); },
    async checkCurrentAccess(resource, action) {
      if (!resource?.kind || !resource?.id || !action) throw new Error('Exact resource and action are required');
      try {
        const result = await callTool('authz.sdk.authorization.check', {resource, action});
        const decision = result?.decision;
        const bounded = decision?.bounded ?? decision?.Bounded;
        const entities = decision?.entities ?? decision?.Entities ?? [];
        if (typeof bounded !== 'boolean' || !Array.isArray(entities)) throw new Error('Authorization decision is invalid');
        return {effect: 'allow', bounded, entities};
      } catch (error) {
        if (error?.status === 403 || error?.code === 'forbidden') return {effect: 'deny', bounded: false, entities: []};
        throw error;
      }
    },
  };
}

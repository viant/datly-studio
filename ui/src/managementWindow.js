const authorizationTools = new Set(['authz.sdk.namedPolicies.manage','authz.sdk.policies.get','authz.sdk.policies.context','authz.sdk.policies.create','authz.sdk.policies.replace','authz.sdk.authorization.check']);
export const isManagedMCPTool = method => typeof method === 'string' && (method.startsWith('studio.sdk.') || authorizationTools.has(method));

export {resourceWindowMetadata as managementWindowMetadata} from 'forge/resource-window/metadata';

export function managementWindowTransport(api, metadata, onOperationCompleted) {
  return async ({dataSourceRef, options = {}}) => {
    const source = metadata.dataSource[dataSourceRef];
    if (!source) throw new Error('Unknown management datasource.');
    if (source.backend?.kind === 'inline') return {data: source.backend.rows || []};
    const method = source.backend?.method;
    if (source.backend?.kind === 'mcp_protocol') {
      // Catalog discovery stays on the host-selected namespace runtime; window
      // metadata cannot supply a URL, credentials, or an arbitrary RPC method.
      if (source.backend.service !== 'runtime' || source.backend.method !== 'tools/list') throw new Error('Unsupported management MCP catalog.');
      if (typeof api.invokeManagementProtocol === 'function') return api.invokeManagementProtocol('runtime', 'tools/list', {signal: options.signal});
      return {tools: await api.listMCPTools()};
    }

    if (source.backend?.kind !== 'mcp_tool' || typeof method !== 'string' || !method) throw new Error('Unsupported management datasource operation.');
    if (typeof api.invokeManagementTool === 'function') {
      const result = await api.invokeManagementTool(method, options.inputParameters || {}, {signal: options.signal});
      await onOperationCompleted?.(method);
      return result;
    }
    if (!isManagedMCPTool(method)) throw new Error('This host has no adapter for the management tool.');
    // Only the host-selected Studio SDK receives requests; a window cannot
    // replace its endpoint or the signed-in identity.
    const operation = method.startsWith('studio.sdk.') ? method.slice('studio.sdk.'.length) : method;
    const result = typeof api.fetchMCPDataSource === 'function'
      ? await api.fetchMCPDataSource({...source, selectors:undefined}, options.inputParameters || {})
      : await (api.nativeCallTool ? api.nativeRequest(null, operation, options.inputParameters || {}) : api.invoke(operation, options.inputParameters || {}));
    await onOperationCompleted?.(operation,result);
    return result;
  };
}

// Project the producer's canonical resource into Forge's renderer. This module
// contains no screen layout, action implementation, or connector field rules.
export function managementWindowMetadata(envelope, target = {platform: 'web', formFactor: 'desktop'}) {
  if (envelope?.schemaVersion !== 3 || !envelope.variants || !Array.isArray(envelope.targets)) throw new Error('Invalid management window resource.');
  const binding = envelope.targets.find(({target: candidate}) => candidate?.platform === target.platform && candidate?.formFactor === target.formFactor)
    || envelope.targets.find(({target: candidate}) => !candidate || Object.keys(candidate).length === 0);
  const variant = envelope.variants[binding?.variant];
  if (!variant?.window) throw new Error('Management window has no matching target.');
  const sources = Object.fromEntries(Object.entries(variant.dataSources || {}).map(([id, source]) => [id, typeof source === 'string' ? JSON.parse(source) : source]));
  return {...variant.window, dataSource: Object.fromEntries(Object.entries(variant.window.dataSource || {}).map(([id, source]) => {
    if (!sources[id]) throw new Error(`Management datasource ${id} is missing.`);
    return [id, {...sources[id], ...source}];
  }))};
}

export function managementWindowTransport(api, metadata, onOperationCompleted) {
  return async ({dataSourceRef, options = {}}) => {
    const source = metadata.dataSource[dataSourceRef];
    if (!source) throw new Error('Unknown management datasource.');
    if (source.backend?.kind === 'inline') return {data: source.backend.rows || []};
    const method = source.backend?.method;
    if (source.backend?.kind !== 'mcp_tool' || !method?.startsWith('studio.sdk.')) throw new Error('Unsupported management datasource operation.');
    // Only the host-selected Studio SDK receives requests; a window cannot
    // replace its endpoint or the signed-in identity.
    const operation = method.slice('studio.sdk.'.length);
    const result = await (api.nativeCallTool ? api.nativeRequest(null, operation, options.inputParameters || {}) : api.invoke(operation, options.inputParameters || {}));
    await onOperationCompleted?.(operation);
    return result;
  };
}

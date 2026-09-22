export function baseRoute(structure) {
  return structure?.component?.routes?.[0] ?? null;
}

export function toolExposure(structure) {
  return baseRoute(structure)?.mcp?.find((item) => !item?.kind || item.kind === 'tool') ?? null;
}

export function suggestExposureName(component, ownerPackage = '') {
  const routePath = component?.routes?.[0]?.path ?? '';
  const routeIdentity = String(routePath)
    .split('/')
    .filter((segment) => segment && !segment.startsWith('{'))
    .at(-1);
  const viewIdentity = component?.rootView?.namespace || component?.rootView?.name;
  const componentIdentity = !/^reader$/i.test(String(component?.name ?? '')) ? component?.name : '';
  const identity = String(routeIdentity || viewIdentity || componentIdentity || component?.key?.name || 'reader')
    .replace(/[^A-Za-z0-9]+/g, '.')
    .replace(/^\.+|\.+$/g, '');
  return canonicalMCPName([ownerPackage, identity || 'reader', 'read'].filter(Boolean).join('.'));
}

// MCP names are durable public identities, not display labels. Keep them
// predictable for clients, URLs, and catalogue search.
export function canonicalMCPName(value) {
  return String(value ?? '')
    .trim()
    .replace(/[^A-Za-z0-9]+/g, '.')
    .replace(/^\.+|\.+$/g, '')
    .slice(0, 117);
}

export function exposureSetting({ enabled, name, description, descriptionPath }) {
  if (!enabled) return { name: 'mcp', remove: true };
  const args = [quoteDQL(name.trim())];
  const cleanDescription = description.trim().replace(/\s+/g, ' ');
  const cleanPath = descriptionPath.trim();
  if (cleanDescription || cleanPath) args.push(quoteDQL(cleanDescription));
  if (cleanPath) args.push(quoteDQL(cleanPath));
  return { name: 'mcp', args };
}

export function componentSettingsOperation({ connector, cubeEnabled, cubeMCP = true, composeEnabled, composeMCP, maxCubes, maxLimit, timeoutMs, exposure }) {
  const cubeArgs = cubeMCP ? [] : ["''", "''", "''", "''", "''", "''", "''", 'false'];
  const cube = { type: 'setSetting', setting: cubeEnabled ? { name: 'cube', args: cubeArgs } : { name: 'cube', remove: true } };
  const composition = {
    type: 'setSetting',
    setting: composeEnabled
      ? { name: 'cubeCompose', args: ['true', String(Boolean(composeMCP)), String(maxCubes), String(maxLimit), String(timeoutMs)] }
      : { name: 'cubeCompose', remove: true },
  };
  const operations = [{ type: 'setSetting', setting: { name: 'connector', args: [quoteDQL(connector)] } }, ...(cubeEnabled ? [cube, composition] : [composition, cube])];
  operations.push({ type: 'setSetting', setting: exposureSetting(exposure) });
  operations.push({ type: 'setSetting', setting: exposure?.enabled && exposure?.mcpOnly ? { name: 'mcpOnly', args: ['true'] } : { name: 'mcpOnly', remove: true } });
  return { type: 'batch', operations };
}

export function validateExposure({ enabled, name, description, descriptionPath, route, ownerPackage }) {
  if (!enabled) return '';
  if (!route?.path || !route?.method) return 'The component needs one compiled HTTP route before it can be exposed as an MCP tool.';
  if (!/^[A-Za-z][A-Za-z0-9_.-]{0,116}$/.test(name.trim())) {
    return 'Tool name must be canonical: start with a letter, use only letters, numbers, dots, dashes, or underscores, and fit generated Cube names.';
  }
  if (ownerPackage && !name.trim().startsWith(`${ownerPackage}.`)) {
    return `Tool name must use the owner prefix ${ownerPackage}.`;
  }
  if (description.trim().length > 512) return 'Description must be 512 characters or fewer.';
  const path = descriptionPath.trim();
  if (path && (path.length > 500 || path.startsWith('/') || path.includes('\\') || path.split('/').includes('..'))) {
    return 'Description resource must be a relative resource path without traversal.';
  }
  return '';
}

export function derivedExposureState(structure) {
  const report = structure?.component?.settings?.report;
  return {
    cube: Boolean(report?.enabled && (report?.mcpTool ?? true)),
    compose: Boolean(report?.compose?.enabled && report?.compose?.mcpTool),
  };
}

function quoteDQL(value) {
  return JSON.stringify(String(value));
}

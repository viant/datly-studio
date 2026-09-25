export function overviewIssues(input = {}) {
  const { reports = [], connectors = [], runtime = null, errors = {} } = input || {};
  const issues = [];
  if (!runtime || runtime.host?.status !== 'ready') {
    issues.push({ key: 'runtime-host', intent: 'danger', title: 'Live runtime could not be verified', detail: 'The live HTTP and MCP host did not pass a runtime check. A published generation record alone does not prove it is serving.', section: 'runtime' });
  }
  if (errors.connectors) {
    issues.push({ key: 'connector-catalog', intent: 'warning', title: 'Connector readiness is unavailable', detail: errors.connectors, section: 'connectors' });
  }
  for (const connector of errors.connectors ? [] : connectors) {
    if (connector.status === 'active' && connector.lastTestStatus !== 'passed') {
      issues.push({ key: `connector:${connector.name}`, intent: 'warning', title: `${connector.name} needs a connection test`, detail: connector.lastTestErrorCode || 'No successful probe is recorded.', section: 'connectors' });
    }
  }
  const drafts = reports.filter((report) => report.status === 'draft');
  if (drafts.length > 0) {
    issues.push({ key: 'drafts', intent: 'primary', title: `${drafts.length} draft ${drafts.length === 1 ? 'component' : 'components'} need review`, detail: drafts.slice(0, 3).map((report) => report.title).join(', '), section: 'reports' });
  }
  return issues;
}

export function environmentChecks(input = {}) {
  const { namespaces = [], connectors = [], runtime = null, errors = {} } = input || {};
  const activeConnectors = connectors.filter((connector) => connector.status === 'active');
  const testedConnectors = activeConnectors.filter((connector) => connector.lastTestStatus === 'passed');
  return [
    { key: 'connectors', label: 'Connectors', value: errors.connectors || `${testedConnectors.length}/${activeConnectors.length} active connectors tested`, ready: !errors.connectors && activeConnectors.length > 0 && testedConnectors.length === activeConnectors.length, section: 'connectors' },
    { key: 'namespaces', label: 'Namespaces', value: errors.namespaces || `${namespaces.length} governed`, ready: !errors.namespaces && namespaces.length > 0, section: 'namespaces' },
    { key: 'runtime', label: 'Live runtime', value: errors.runtime || (runtime?.host?.status === 'ready' ? `Ready · revision ${runtime.host.revision}` : runtime?.host?.status === 'unavailable' ? 'Check failed' : 'Not checked'), ready: !errors.runtime && runtime?.host?.status === 'ready', section: 'runtime' },
  ];
}

export function overviewFromSettled(results) {
  const names = ['reports', 'connectors', 'namespaces', 'runtime'];
  const output = { reports: [], connectors: [], namespaces: [], runtime: null, errors: {} };
  results.forEach((result, index) => {
    const name = names[index];
    if (result.status === 'fulfilled') {
      output[name] = name === 'runtime' ? result.value : result.value?.items ?? [];
      return;
    }
    output.errors[name] = result.reason?.message || `${name} request failed`;
  });
  return output;
}

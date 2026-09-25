// StudioAPI is deliberately a client-side counterpart of sdk.Transport: every
// UI interaction names an SDK operation and sends an SDK DTO. It has no direct
// SQL, DQL, Datly component, or storage knowledge.
import { postV1StudioSdkAclList, postV1StudioSdkConnectorsGet, postV1StudioSdkConnectorsList, postV1StudioSdkNamespacesGet, postV1StudioSdkNamespacesList, postV1StudioSdkPublicationsEventsList, postV1StudioSdkPublicationsGet, postV1StudioSdkReportsGet, postV1StudioSdkReportsList } from './generated/studioClient.gen.js';

const nativeErrorCode = { 400: 'invalid_argument', 401: 'unauthorized', 403: 'forbidden', 404: 'not_found', 409: 'conflict', 422: 'invalid_argument', 502: 'unavailable', 503: 'unavailable' };

export class StudioAPI {
  constructor(config, options = {}) {
    this.config = config;
    // Window.fetch requires its original receiver in browsers. Test doubles do
    // not, so normalize both forms into an ordinary callable function.
    this.fetcher = options.fetcher ?? globalThis.fetch.bind(globalThis);
    this.onUnauthorized = options.onUnauthorized;
  }

  async invoke(operation, input = {}) {
    const headers = { Accept: 'application/json', 'Content-Type': 'application/json' };
    if (this.config.mode === 'development') {
      headers['X-Studio-Development-Subject'] = this.config.development.subject;
    }
    const response = await this.fetcher(`${this.config.apiBaseURL}/v1/studio/sdk/${encodeURIComponent(operation)}`, {
      method: 'POST', headers, credentials: this.config.mode === 'authenticated' ? 'include' : 'same-origin', body: JSON.stringify(input),
    });
    const payload = response.status === 204 ? null : await response.json();
    if (!response.ok) {
      if (response.status === 401) this.onUnauthorized?.();
      const requestId = response.headers?.get?.('X-Request-ID') || '';
      const baseMessage = payload?.message || `Studio SDK operation ${operation} failed (${response.status})`;
      const message = requestId ? `${baseMessage} · request ${requestId}` : baseMessage;
      const error = new Error(message);
      error.code = payload?.code || '';
      error.status = response.status;
      error.field = payload?.field || '';
      error.violations = payload?.violations ?? [];
      error.requestId = requestId;
      throw error;
    }
    return payload;
  }

  listReports(input = {}) { return this.nativeRequest(postV1StudioSdkReportsList, 'reports.list', input); }
  getReport(reportId) { return this.nativeRequest(postV1StudioSdkReportsGet, 'reports.get', { id: reportId }); }
  createReport(input) { return this.invoke('reports.create', input); }
  updateReport(reportId, input) { return this.invoke('reports.update', { id: reportId, input }); }
  listVersions(reportId, input = {}) { return this.invoke('versions.list', { reportId, input }); }
  createVersion(reportId, input) { return this.invoke('versions.create', { reportId, input }); }
  loadDQL(reportId, input) { return this.invoke('versions.load_dql', { reportId, input }); }
  loadArchive(reportId, input) { return this.invoke('versions.load_archive', { reportId, input }); }
  downloadComponent(reportId, versionNo) { return this.invoke('versions.download', { reportId, versionNo }); }
  inspectVersion(reportId, versionNo) { return this.invoke('versions.inspect', { reportId, versionNo }); }
  validateVersion(reportId, versionNo, expectedSourceRevision) { return this.invoke('versions.validate', { reportId, versionNo, expectedSourceRevision }); }
  publishReader(reportId, versionNo, expectedSourceRevision, reason = '') { return this.invoke('publications.publish', { reportId, versionNo, input: { expectedSourceRevision, reason } }); }
  getPublication(reportId) { return this.nativeRequest(postV1StudioSdkPublicationsGet, 'publications.get', { reportId }); }
  listPublicationEvents(reportId, input = {}) { return this.nativeRequest(postV1StudioSdkPublicationsEventsList, 'publications.events.list', { reportId, input }); }
  unpublishReader(reportId, expectedActiveGeneration, reason = '') { return this.invoke('publications.unpublish', { reportId, input: { expectedActiveGeneration, reason } }); }
  rollbackReader(reportId, versionNo, expectedSourceRevision, reason = '') { return this.invoke('publications.rollback', { reportId, versionNo, input: { expectedSourceRevision, reason } }); }
  getRuntimeStatus() { return this.invoke('runtime.status'); }
  async listMCPTools() {
    return this.mcpRequest('tools/list', 'tools');
  }
  async listMCPSkills() {
    return this.mcpRequest('skills/list', 'skills');
  }
  async mcpRequest(method, resultKey) {
    const response = await this.fetcher(`${this.config.apiBaseURL}/v1/studio/mcp/mcp`, {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json', 'Mcp-Protocol-Version': '2026-07-28', 'Mcp-Method': method },
      credentials: this.config.mode === 'authenticated' ? 'include' : 'same-origin',
      body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params: { _meta: { 'io.modelcontextprotocol/protocolVersion': '2026-07-28', 'io.modelcontextprotocol/clientCapabilities': {} } } }),
    });
    const body = await response.text();
    let payload = null;
    if (body.trim()) {
      try { payload = JSON.parse(body); }
      catch {
        if (response.ok) throw new Error(`Datly MCP returned an invalid response for ${method}`);
      }
    }
    if (!response.ok || payload?.error || !payload) {
      if (response.status === 401) this.onUnauthorized?.();
      const fallback = response.status === 502 || response.status === 503
        ? 'The dedicated Datly MCP server is unavailable. Start or reconnect the dynamic runtime, then retry.'
        : response.status === 401
          ? 'The Datly MCP server rejected this session. Check that Studio and the runtime use the same authentication mode.'
          : `Datly MCP ${method} failed (${response.status})`;
      const error = new Error(payload?.error?.message || payload?.message || body.trim() || fallback);
      error.status = response.status;
      throw error;
    }
    return payload?.result?.[resultKey] ?? [];
  }
  getResourceAccess(resource) { return this.invoke('access.get', resource); }
  getResourceAccessContext(resource) { return this.invoke('access.context', resource); }
  replaceResourceAccess(document) { return this.invoke('access.replace', document); }
  async nativeRequest(call, operation, body) {
    const headers = { Accept: 'application/json' };
    if (this.config.mode === 'development') headers['X-Studio-Development-Subject'] = this.config.development.subject;
    const { data, error, response } = await call({
      body, baseUrl: this.config.apiBaseURL, fetch: this.fetcher,
      credentials: this.config.mode === 'authenticated' ? 'include' : 'same-origin', headers,
      parseAs: 'json',
    });
    if (error) {
      if (response?.status === 401) this.onUnauthorized?.();
      const requestId = response?.headers?.get?.('X-Request-ID') || '';
      const baseMessage = error?.message || `Studio SDK operation ${operation} failed (${response?.status ?? 'network'})`;
      const failure = new Error(requestId ? `${baseMessage} · request ${requestId}` : baseMessage);
      failure.code = error?.code || nativeErrorCode[response?.status] || 'internal';
      failure.status = response?.status;
      failure.field = error?.field || '';
      failure.violations = error?.violations ?? [];
      failure.requestId = requestId;
      throw failure;
    }
    return data;
  }
  async listACL(reportId) { return (await this.nativeRequest(postV1StudioSdkAclList, 'acl.list', { reportId }))?.items ?? []; }
  upsertACL(input) { return this.invoke('acl.upsert', input); }
  deleteACL(reportId, subjectType, subjectId, etag) { return this.invoke('acl.delete', { reportId, subjectType, subjectId, etag }); }
  getResources(reportId, versionNo) { return this.invoke('resources.get', { reportId, versionNo }); }
  upsertResourceFile(input) { return this.invoke('resources.upsert_file', input); }
  deleteResourceFile(reportId, versionNo, resourceId, expectedSourceRevision) { return this.invoke('resources.delete_file', { reportId, versionNo, resourceId, expectedSourceRevision }); }
  upsertResourceFolder(input) { return this.invoke('resources.upsert_folder', input); }
  deleteResourceFolder(reportId, versionNo, folderId, expectedSourceRevision) { return this.invoke('resources.delete_folder', { reportId, versionNo, folderId, expectedSourceRevision }); }
  upsertSkillRoot(input) { return this.invoke('resources.upsert_skill', input); }
  deleteSkillRoot(reportId, versionNo, skillId, expectedSourceRevision) { return this.invoke('resources.delete_skill', { reportId, versionNo, skillId, expectedSourceRevision }); }
  applyReaderCommand(reportId, versionNo, command) { return this.invoke('versions.builder', { reportId, versionNo, command }); }
  testReaderView(reportId, versionNo, view, input = {}, limit = 50) { return this.invoke('versions.test_view', { reportId, versionNo, view, input: { input, limit } }); }
  testReaderRelation(reportId, versionNo, relation, input = {}, limit = 50) { return this.invoke('versions.test_relation', { reportId, versionNo, relation, input: { input, limit } }); }
  testCubeCompose(reportId, versionNo, input) { return this.invoke('versions.test_compose', { reportId, versionNo, input }); }
  warmupReader(reportId, versionNo) { return this.invoke('versions.warmup', { reportId, versionNo }); }
  getWarmupRun(reportId, runId) { return this.invoke('versions.warmup_get', { reportId, runId }); }
  listWarmupRuns(reportId, versionNo, input = {}) { return this.invoke('versions.warmup_list', { reportId, versionNo, input }); }
  previewReader(reportId, versionNo, input = {}, limit = 50) { return this.invoke('preview.execute', { reportId, versionNo, input: { input, limit } }); }
  listConnectors(input = {}) { return this.nativeRequest(postV1StudioSdkConnectorsList, 'connectors.list', input); }
  getConnector(name) { return this.nativeRequest(postV1StudioSdkConnectorsGet, 'connectors.get', { name }); }
  createConnector(input) { return this.invoke('connectors.create', input); }
  updateConnector(name, input) { return this.invoke('connectors.update', { name, input }); }
  testConnector(name) { return this.invoke('connectors.test', { name }); }
  listSchemas(name, input = {}) { return this.invoke('connectors.schemas', { name, input }); }
  listTables(name, input = {}) { return this.invoke('connectors.tables', { name, input }); }
  getTable(name, input) { return this.invoke('connectors.table', { name, input }); }
  testSQL(name, input) { return this.invoke('connectors.test_sql', { name, input }); }
  activateConnector(name, etag) { return this.invoke('connectors.activate', { name, etag }); }
  disableConnector(name, etag) { return this.invoke('connectors.disable', { name, etag }); }
  deleteConnector(name, etag) { return this.invoke('connectors.delete', { name, etag }); }
  listNamespaces(input = {}) { return this.nativeRequest(postV1StudioSdkNamespacesList, 'namespaces.list', input); }
  getNamespace(name) { return this.nativeRequest(postV1StudioSdkNamespacesGet, 'namespaces.get', { name }); }
  createNamespace(input) { return this.invoke('namespaces.create', input); }
  updateNamespace(name, input) { return this.invoke('namespaces.update', { name, input }); }
  deleteNamespace(name, etag) { return this.invoke('namespaces.delete', { name, etag }); }
  listAuthorizationPredicates(input = {}) { return this.invoke('authorization_predicates.list', input); }
  listAuthorizationPredicateTypes() { return this.invoke('authorization_predicates.types').then((page) => page?.items ?? []); }
  createAuthorizationPredicate(input) { return this.invoke('authorization_predicates.create', input); }
  updateAuthorizationPredicate(name, input) { return this.invoke('authorization_predicates.update', { name, input }); }
  deleteAuthorizationPredicate(name, etag) { return this.invoke('authorization_predicates.delete', { name, etag }); }
}

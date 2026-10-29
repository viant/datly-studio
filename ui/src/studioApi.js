import {isManagedMCPTool} from './managementWindow.js';
import { postV1StudioSdkNamespaceSkillsList, postV1StudioSdkNamespaceSkillsGet, postV1StudioSdkNamespaceSkillsCreate, postV1StudioSdkNamespaceSkillsUpdate, postV1StudioSdkNamespaceSkillsDelete, postV1StudioSdkNamespaceSkillsRevisions, postV1StudioSdkNamespaceSkillsPublish, postV1StudioSdkNamespaceSkillsUnpublish } from './generated/studioClient.gen.js';
// StudioAPI is deliberately a client-side counterpart of sdk.Transport: every
// UI interaction names an SDK operation and sends an SDK DTO. It has no direct
// SQL, DQL, Datly component, or storage knowledge.
import { postV1StudioSdkAclDelete, postV1StudioSdkAclList, postV1StudioSdkAclUpsert, postV1StudioSdkAuthorizationPredicatesTypes, postV1StudioSdkConnectorsActivate, postV1StudioSdkConnectorsCreate, postV1StudioSdkConnectorsDelete, postV1StudioSdkConnectorsDisable, postV1StudioSdkConnectorsGet, postV1StudioSdkConnectorsList, postV1StudioSdkConnectorsUpdate, postV1StudioSdkNamespacesCreate, postV1StudioSdkNamespacesDelete, postV1StudioSdkNamespacesGet, postV1StudioSdkNamespacesList, postV1StudioSdkNamespacesUpdate, postV1StudioSdkPublicationsEventsList, postV1StudioSdkPublicationsGet, postV1StudioSdkComponentsCreate, postV1StudioSdkComponentsGet, postV1StudioSdkComponentsList, postV1StudioSdkComponentsUpdate, postV1StudioSdkResourcesGet, postV1StudioSdkVersionsApply, postV1StudioSdkVersionsCreate, postV1StudioSdkVersionsClone, postV1StudioSdkVersionsDescriptor, postV1StudioSdkVersionsDownload, postV1StudioSdkVersionsExportDql, postV1StudioSdkVersionsGet, postV1StudioSdkVersionsInspect, postV1StudioSdkVersionsList, postV1StudioSdkVersionsLoadArchive, postV1StudioSdkVersionsLoadDql, postV1StudioSdkVersionsWarmup, postV1StudioSdkVersionsWarmupGet, postV1StudioSdkVersionsWarmupList } from './generated/studioClient.gen.js';
import { postV1StudioSdkAuthorizationPredicatesGet, postV1StudioSdkAuthorizationPredicatesList } from './generated/studioClient.gen.js';
import { postV1StudioSdkAuthorizationPredicatesCreate } from './generated/studioClient.gen.js';
import { postV1StudioSdkAuthorizationPredicatesUpdate, postV1StudioSdkAuthorizationPredicatesDelete } from './generated/studioClient.gen.js';
import { postV1StudioSdkConnectorsDiscover, postV1StudioSdkConnectorsSchemas, postV1StudioSdkConnectorsTables, postV1StudioSdkConnectorsTable } from './generated/studioClient.gen.js';
import { postV1StudioSdkConnectorsTest } from './generated/studioClient.gen.js';
import { postV1StudioSdkConnectorsTestSql } from './generated/studioClient.gen.js';
import { postV1StudioSdkPreviewExecute } from './generated/studioClient.gen.js';
import { postV1StudioSdkVersionsValidate } from './generated/studioClient.gen.js';
import { postV1StudioSdkVersionsTestView, postV1StudioSdkVersionsTestRelation, postV1StudioSdkVersionsTestCompose } from './generated/studioClient.gen.js';
import { postV1StudioSdkResourcesUpsertFile, postV1StudioSdkResourcesDeleteFile, postV1StudioSdkResourcesUpsertFolder, postV1StudioSdkResourcesDeleteFolder, postV1StudioSdkResourcesUpsertSkill, postV1StudioSdkResourcesDeleteSkill } from './generated/studioClient.gen.js';
import { postV1StudioSdkVersionsBuilder } from './generated/studioClient.gen.js';
import { postV1StudioSdkRuntimeStatus } from './generated/studioClient.gen.js';
import { postV1StudioSdkPublicationsPublish, postV1StudioSdkPublicationsRollback, postV1StudioSdkPublicationsUnpublish } from './generated/studioClient.gen.js';
import { postV1StudioSdkAccessList, postV1AuthzSdkPoliciesGet, postV1AuthzSdkPoliciesContext, postV1AuthzSdkPoliciesReplace, postV1AuthzSdkPoliciesCreate, postV1AuthzSdkAuthorizationCheck } from './generated/studioClient.gen.js';
import {postNamedPolicies} from './namedPolicySDK.js';
import { accessMCPAdapter } from './accessMCPAdapter.js';
import { BrowserIdentity } from './browserIdentity.js';
import {releaseMethods, supportsReleaseTools} from 'forge/release-manager/adapter';

const nativeErrorCode = { 400: 'invalid_argument', 401: 'unauthorized', 403: 'forbidden', 404: 'not_found', 409: 'conflict', 422: 'invalid_argument', 502: 'unavailable', 503: 'unavailable' };

export class StudioAPI {
  constructor(config, options = {}) {
    this.config = config;
    // Window.fetch requires its original receiver in browsers. Test doubles do
    // not, so normalize both forms into an ordinary callable function.
    this.fetcher = options.fetcher ?? globalThis.fetch.bind(globalThis);
    this.authRevision = 0;
    this.verifiedSubject = '';
    this.onUnauthorized = () => { this.clearVerifiedSubject(); options.onUnauthorized?.(); };
    this.nativeCallTool = options.nativeCallTool;
    this.resourceAccess = accessMCPAdapter((name, input) => {
      const calls = {
        'authz.sdk.namedPolicies.manage': postNamedPolicies,
        'authz.sdk.policies.get': postV1AuthzSdkPoliciesGet,
        'authz.sdk.policies.context': postV1AuthzSdkPoliciesContext,
        'authz.sdk.policies.replace': postV1AuthzSdkPoliciesReplace,
        'authz.sdk.policies.create': postV1AuthzSdkPoliciesCreate,
        'authz.sdk.authorization.check': postV1AuthzSdkAuthorizationCheck,
      };
      const call = calls[name];
      if (!call) throw new Error('Authorization operation is not supported');
      return this.nativeRequest(call, name, input);
    });
    this.namespaceId = null;
    this.namespaceRevision = 0;
    this.namespaceBlocked = false;
    this.releaseToolAvailability = null;
    this.identity = this.config.authentication?.mode === 'identity-token'
      ? options.identity ?? new BrowserIdentity(config, this.fetcher) : null;
  }

  setNamespace(namespaceId) {
    if (namespaceId !== null && (typeof namespaceId !== 'string' || !/^[a-f0-9]{64}$/.test(namespaceId))) throw new TypeError('A valid namespace ID is required');
    if (this.namespaceId === namespaceId) return;
    this.namespaceId = namespaceId;
    this.namespaceRevision += 1;
    this.prepareNativeTools();
  }

  // These values partition transport reuse only. The producer still verifies
  // every call; a browser-supplied subject never becomes an authorization fact.
  setVerifiedSubject(subject) {
    const next = String(subject || '').trim();
    if (next && next !== this.verifiedSubject) { this.verifiedSubject = next; this.authRevision++; this.prepareNativeTools(); }
  }

  clearVerifiedSubject() { this.verifiedSubject = ''; this.authRevision++; }

  prepareNativeTools() {
    if (!this.namespaceId || this.namespaceBlocked || typeof this.nativeCallTool?.prepare !== 'function'
      || this.config.mode === 'authenticated' && !this.verifiedSubject) return;
    const snapshot=this.namespaceSnapshot('release.management');
    const headers={'X-Studio-Namespace':snapshot.id};
    if(this.config.mode==='development')headers['X-Studio-Development-Subject']=this.config.development.subject;
    try { void Promise.resolve(this.nativeCallTool.prepare(this.nativeToolOptions(snapshot,headers))).catch(()=>{}); }
    catch { /* Explicit calls surface setup errors; eager preparation is optional. */ }
  }

  nativeScopeKey(snapshot) {
    return JSON.stringify([snapshot?.id || '', snapshot?.revision ?? this.namespaceRevision, this.authRevision, this.identity?.epoch ?? 0]);
  }

  nativeToolOptions(snapshot, headers) {
    const scopeKey = this.nativeScopeKey(snapshot);
    return {headers, owner: this, scopeKey, send: async (url, init) => {
      this.assertNamespace(snapshot);
      if (scopeKey !== this.nativeScopeKey(snapshot)) throw Object.assign(new Error('The Studio identity changed while this request was running.'), {name:'AbortError',code:'identity_changed'});
      const response = await this.send(url, {...init, credentials: this.identity ? 'omit' : 'include'}, snapshot);
      this.assertNamespace(snapshot);
      if (scopeKey !== this.nativeScopeKey(snapshot)) throw Object.assign(new Error('The Studio identity changed while this request was running.'), {name:'AbortError',code:'identity_changed'});
      return response;
    }};
  }

  namespaceSnapshot(operation) {
    if (operation.startsWith('namespaces.') || operation.startsWith('connectors.') || (operation === 'authorization_predicates.types' || operation === 'codecs.types')) return null;
    if (this.namespaceBlocked) {
      const error = new Error('Choose an available namespace before continuing.');
      error.code = 'namespace_unavailable';
      throw error;
    }
    return { id: this.namespaceId, revision: this.namespaceRevision };
  }

  setNamespaceBlocked(blocked) {
    if (this.namespaceBlocked === blocked) return;
    this.namespaceBlocked = blocked;
    this.namespaceRevision += 1;
    if(!blocked)this.prepareNativeTools();
  }

  assertNamespace(snapshot) {
    if (snapshot && snapshot.revision !== this.namespaceRevision) {
      const error = new Error('The namespace changed while this request was running.');
      error.name = 'AbortError';
      error.code = 'namespace_changed';
      throw error;
    }
  }

  async send(value, init, snapshot = null) {
    if (!this.identity) return this.fetcher(value, init);
    const token = await this.identity.token();
    this.assertNamespace(snapshot);
    const first = new Request(value, init);
    const retry = first.clone();
    const headers = new Headers(first.headers);
    headers.set('Authorization', `Bearer ${token}`);
    const response = await this.fetcher(new Request(first, { headers, credentials: 'omit' }));
    this.assertNamespace(snapshot);
    if (response.status !== 401) return response;
    let renewed;
    try { renewed = await this.identity.token(true); }
    catch { this.assertNamespace(snapshot); this.onUnauthorized?.(); return response; }
    this.assertNamespace(snapshot);
    const retryHeaders = new Headers(retry.headers);
    retryHeaders.set('Authorization', `Bearer ${renewed}`);
    return this.fetcher(new Request(retry, { headers: retryHeaders, credentials: 'omit' }));
  }

  async readManagementResource(uri) {
    const endpoint = this.identity ? `${this.config.mcpBaseURL}/mcp` : `${this.config.apiBaseURL}/v1/studio/sdk-mcp/mcp`;
    const response = await this.send(endpoint, {
      method: 'POST', credentials: this.identity ? 'omit' : 'include',
      headers: {'Content-Type': 'application/json', Accept: 'application/json', 'Mcp-Method': 'resources/read', 'Mcp-Name': uri, 'Mcp-Protocol-Version': '2026-07-28'},
      body: JSON.stringify({jsonrpc:'2.0',id:1,method:'resources/read',params:{uri,_meta:{'io.modelcontextprotocol/protocolVersion':'2026-07-28','io.modelcontextprotocol/clientCapabilities':{}}}}),
    });
    const value = await response.json();
    // Older hosts expose the authorized namespace resource tool but do not yet
    // bridge virtual resources into resources/read.
    if (response.ok && value.error?.code === -32002) {
      const fallback = await this.send(endpoint, {method:'POST',credentials:this.identity?'omit':'include',headers:{'Content-Type':'application/json',Accept:'application/json','Mcp-Method':'tools/call','Mcp-Name':'resource/get','Mcp-Protocol-Version':'2026-07-28'},body:JSON.stringify({jsonrpc:'2.0',id:2,method:'tools/call',params:{name:'resource/get',arguments:{uri},_meta:{'io.modelcontextprotocol/protocolVersion':'2026-07-28','io.modelcontextprotocol/clientCapabilities':{}}}})});
      const payload=await fallback.json();
      if(!fallback.ok||payload.error||payload.result?.isError)throw new Error(payload.error?.message||payload.result?.content?.find(item=>item.type==='text')?.text||'Management resource could not be loaded.');
      if(payload.result?.structuredContent!==undefined)return payload.result.structuredContent;
      const text=payload.result?.content?.find(item=>item.type==='text')?.text;
      if(!text)throw new Error('Management window resource is missing.');
      return JSON.parse(text);
    }
    if (!response.ok || value.error) {
      if (response.status === 401) this.onUnauthorized?.();
      throw new Error(value?.error?.message || value?.message || 'Management window could not be loaded.');
    }
    const content = value.result?.contents?.find(item => item.uri === uri && typeof item.text === 'string');
    if (!content) throw new Error('Management window resource is missing.');
    return JSON.parse(content.text);
  }

  async fetchMCPDataSource(source, input = {}) {
    const method = source?.backend?.method;
    if (source?.backend?.kind !== 'mcp_tool' || !isManagedMCPTool(method)) throw new Error('A Studio MCP datasource is required.');
    const operation = method.startsWith('studio.sdk.') ? method.slice('studio.sdk.'.length) : method;
    const snapshot = this.namespaceSnapshot(operation);
    const args = {...input, ...(method.startsWith('studio.sdk.') && snapshot?.id ? {namespaceId:snapshot.id} : {})};
    let result;
    if (this.nativeCallTool) result = await this.nativeRequest(null, operation, args);
    else {
      const endpoint = this.identity ? `${this.config.mcpBaseURL}/mcp` : `${this.config.apiBaseURL}/v1/studio/sdk-mcp/mcp`;
      const headers = {'Content-Type':'application/json',Accept:'application/json','Mcp-Method':'tools/call','Mcp-Name':method,'Mcp-Protocol-Version':'2026-07-28'};
      if (snapshot?.id) headers['X-Studio-Namespace'] = snapshot.id;
      if (this.config.mode === 'development') headers['X-Studio-Development-Subject'] = this.config.development.subject;
      const response = await this.send(endpoint,{method:'POST',credentials:this.identity?'omit':'include',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'tools/call',params:{name:method,arguments:args,_meta:{'io.modelcontextprotocol/protocolVersion':'2026-07-28','io.modelcontextprotocol/clientCapabilities':{}}}})},snapshot);
      const payload = await response.json();
      this.assertNamespace(snapshot);
      const tool = payload?.result;
      if (!response.ok || payload?.error || tool?.isError) {
        if(response.status===401)this.onUnauthorized?.();
        const text=(tool?.content??[]).filter((item)=>item.type==='text').map((item)=>item.text).join(' ');
        const error=new Error(payload?.error?.message||tool?.structuredContent?.message||text||'Reader lookup datasource failed.');
        error.code=tool?.structuredContent?.code||nativeErrorCode[response.status];error.status=response.status;error.providerError=tool?.structuredContent?.providerError;error.diagnostics=tool?.structuredContent?.diagnostics;throw error;
      }
      result=tool?.structuredContent;
      if(result===undefined){const text=tool?.content?.find((item)=>item.type==='text')?.text;if(text)result=JSON.parse(text);}
      if(result===undefined)throw new Error('Reader lookup datasource returned no data.');
    }
    this.assertNamespace(snapshot);
    return source.selectors?.data ? source.selectors.data.split('.').reduce((value,key)=>value?.[key],result) ?? [] : result;
  }

  async invoke(operation, input = {}) {
    const snapshot = this.namespaceSnapshot(operation);
    const headers = { Accept: 'application/json', 'Content-Type': 'application/json' };
    if (this.config.mode === 'development') {
      headers['X-Studio-Development-Subject'] = this.config.development.subject;
    }
    if (snapshot?.id) headers['X-Studio-Namespace'] = snapshot.id;
    const response = await this.send(`${this.config.apiBaseURL}/v1/studio/sdk/${encodeURIComponent(operation)}`, {
      method: 'POST', headers, credentials: this.identity ? 'omit' : this.config.mode === 'authenticated' ? 'include' : 'same-origin', body: JSON.stringify(input),
    }, snapshot);
    const payload = response.status === 204 ? null : await response.json();
    this.assertNamespace(snapshot);
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
      error.providerError = payload?.providerError;
      error.diagnostics = payload?.diagnostics;
      error.requestId = requestId;
      throw error;
    }
    return payload;
  }

  listComponents(input = {}) { return this.nativeRequest(postV1StudioSdkComponentsList, 'components.list', input); }
  getComponent(reportId) { return this.nativeRequest(postV1StudioSdkComponentsGet, 'components.get', { id: reportId }); }
  createComponent(input) { return this.nativeRequest(postV1StudioSdkComponentsCreate, 'components.create', input); }
  updateComponent(reportId, input) { return this.nativeRequest(postV1StudioSdkComponentsUpdate, 'components.update', { id: reportId, input }); }
  listVersions(reportId, input = {}) { return this.nativeRequest(postV1StudioSdkVersionsList, 'versions.list', { reportId, input }); }
  getVersion(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkVersionsGet, 'versions.get', { reportId, versionNo }); }
  exportDQL(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkVersionsExportDql, 'versions.export_dql', { reportId, versionNo }); }
  getVersionDescriptor(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkVersionsDescriptor, 'versions.descriptor', { reportId, versionNo }); }
  createVersion(reportId, input) { return this.nativeRequest(postV1StudioSdkVersionsCreate, 'versions.create', { reportId, input }); }
  cloneVersion(reportId, versionNo, expectedSourceRevision) { return this.nativeRequest(postV1StudioSdkVersionsClone, 'versions.clone', { reportId, versionNo, expectedSourceRevision }); }
  loadDQL(reportId, input) { return this.nativeRequest(postV1StudioSdkVersionsLoadDql, 'versions.load_dql', { reportId, input }); }
  loadArchive(reportId, input) { return this.nativeRequest(postV1StudioSdkVersionsLoadArchive, 'versions.load_archive', { reportId, input }); }
  downloadComponent(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkVersionsDownload, 'versions.download', { reportId, versionNo }); }
  inspectVersion(reportId, versionNo, { discoverColumns } = {}) { return this.nativeRequest(postV1StudioSdkVersionsInspect, 'versions.inspect', discoverColumns === undefined ? { reportId, versionNo } : { reportId, versionNo, discoverColumns }); }
  validateVersion(reportId, versionNo, expectedSourceRevision) { return this.nativeRequest(postV1StudioSdkVersionsValidate, 'versions.validate', { reportId, versionNo, expectedSourceRevision }); }
  publishReader(reportId, versionNo, expectedSourceRevision, reason = '') { return this.nativeRequest(postV1StudioSdkPublicationsPublish, 'publications.publish', { reportId, versionNo, input: { expectedSourceRevision, reason } }); }
  getPublication(reportId) { return this.nativeRequest(postV1StudioSdkPublicationsGet, 'publications.get', { reportId }); }
  listPublicationEvents(reportId, input = {}) { return this.nativeRequest(postV1StudioSdkPublicationsEventsList, 'publications.events.list', { reportId, input }); }
  unpublishReader(reportId, expectedActiveGeneration, reason = '') { return this.nativeRequest(postV1StudioSdkPublicationsUnpublish, 'publications.unpublish', { reportId, input: { expectedActiveGeneration, reason } }); }
  rollbackReader(reportId, versionNo, expectedSourceRevision, reason = '') { return this.nativeRequest(postV1StudioSdkPublicationsRollback, 'publications.rollback', { reportId, versionNo, input: { expectedSourceRevision, reason } }); }
  getRuntimeStatus() { return this.nativeRequest(postV1StudioSdkRuntimeStatus, 'runtime.status', this.namespaceId ? {namespaceId:this.namespaceId} : {}); }
  async listMCPTools() {
    return this.mcpRequest('tools/list', 'tools');
  }
  async listReleaseTools() {
    const snapshot = this.namespaceSnapshot('release.management');
    if (!snapshot?.id) throw new Error('Choose an available namespace before managing releases.');
    const endpoint = this.identity ? `${this.config.mcpBaseURL}/mcp` : `${this.config.apiBaseURL}/v1/studio/sdk-mcp/mcp`;
    const headers = {'Content-Type':'application/json',Accept:'application/json','Mcp-Method':'tools/list','Mcp-Protocol-Version':'2026-07-28','X-Studio-Namespace':snapshot.id};
    if (this.config.mode === 'development') headers['X-Studio-Development-Subject'] = this.config.development.subject;
    const response = await this.send(endpoint,{method:'POST',credentials:this.identity?'omit':'include',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'tools/list',params:{_meta:{'io.modelcontextprotocol/protocolVersion':'2026-07-28','io.modelcontextprotocol/clientCapabilities':{}}}})},snapshot);
    const payload = await response.json();
    this.assertNamespace(snapshot);
    if (!response.ok || payload?.error || !Array.isArray(payload?.result?.tools)) throw new Error('Release tool catalog is unavailable.');
    return payload.result.tools;
  }
  async supportsReleases(namespaceName) {
    const snapshot = this.namespaceSnapshot('release.management');
    if (!snapshot?.id || !namespaceName) return false;
    const key = `${snapshot.id}:${snapshot.revision}:${namespaceName}`;
    if (this.releaseToolAvailability?.key !== key) {
      const promise = this.getRuntimeStatus().then(async status => {
        this.assertNamespace(snapshot);
        if (status?.releasesAvailable !== true || (status.releaseNamespace && status.releaseNamespace !== namespaceName)) return false;
        return supportsReleaseTools(await this.listReleaseTools());
      }).catch(() => false);
      this.releaseToolAvailability = {key,promise};
    }
    const value = await this.releaseToolAvailability.promise;
    this.assertNamespace(snapshot);
    return value;
  }
  async callReleaseTool(method, input) {
    if (!releaseMethods.includes(method)) throw new Error('Unsupported release operation.');
    const snapshot = this.namespaceSnapshot('release.management');
    if (!snapshot?.id) throw new Error('Choose an available namespace before managing releases.');
    const headers = {'Content-Type':'application/json',Accept:'application/json','Mcp-Method':'tools/call','Mcp-Name':method,'Mcp-Protocol-Version':'2026-07-28','X-Studio-Namespace':snapshot.id};
    if (this.config.mode === 'development') headers['X-Studio-Development-Subject'] = this.config.development.subject;
    if (this.nativeCallTool) {
      let result;
      try { result = await this.nativeCallTool(method,input,this.nativeToolOptions(snapshot,headers)); }
      catch (error) { this.assertNamespace(snapshot); if (error?.status === 401) this.onUnauthorized(); throw error; }
      this.assertNamespace(snapshot);
      return result;
    }
    const endpoint = this.identity ? `${this.config.mcpBaseURL}/mcp` : `${this.config.apiBaseURL}/v1/studio/sdk-mcp/mcp`;
    const response = await this.send(endpoint,{method:'POST',credentials:this.identity?'omit':'include',headers,body:JSON.stringify({jsonrpc:'2.0',id:1,method:'tools/call',params:{name:method,arguments:input,_meta:{'io.modelcontextprotocol/protocolVersion':'2026-07-28','io.modelcontextprotocol/clientCapabilities':{}}}})},snapshot);
    const payload = await response.json();
    this.assertNamespace(snapshot);
    const result = payload?.result;
    if (!response.ok || payload?.error || result?.isError) {
      if (response.status === 401) this.onUnauthorized?.();
      const message = result?.structuredContent?.message || payload?.error?.message || result?.content?.find(item=>item.type==='text')?.text || `Release operation failed (${response.status}).`;
      const error = new Error(message);
      error.code = result?.structuredContent?.code || nativeErrorCode[response.status];
      error.status = result?.structuredContent?.status || response.status;
      throw error;
    }
    if (result?.structuredContent !== undefined) return result.structuredContent;
    const text = result?.content?.find(item=>item.type==='text')?.text;
    if (text) return JSON.parse(text);
    throw new Error('Release operation returned no result.');
  }
  async listMCPSkills() {
    return this.mcpRequest('skills/list', 'skills');
  }
  async mcpRequest(method, resultKey) {
    const snapshot = this.namespaceSnapshot('mcp.catalog');
    let mcpURL = this.identity ? `${this.config.mcpBaseURL}/mcp` : `${this.config.apiBaseURL}/v1/studio/mcp/mcp`;
    if (snapshot.id && !this.identity && this.config.mode !== 'authenticated') {
      throw new Error('Namespace MCP discovery requires an authenticated session or identity token.');
    }
    if (snapshot.id && this.identity) {
      const runtime = await this.getRuntimeStatus();
      this.assertNamespace(snapshot);
      if (runtime?.host?.namespaceId !== snapshot.id || !runtime?.host?.mcpUrl) {
        throw new Error('The selected namespace has no ready MCP endpoint. Enable MCP in namespace settings, then retry.');
      }
      mcpURL = runtime.host.mcpUrl;
    }
    const headers = { Accept: 'application/json', 'Content-Type': 'application/json', 'Mcp-Protocol-Version': '2026-07-28', 'Mcp-Method': method };
    if (snapshot.id) headers['X-Studio-Namespace'] = snapshot.id;
    const response = await this.send(mcpURL, {
      method: 'POST',
      headers,
      credentials: this.identity ? 'omit' : this.config.mode === 'authenticated' ? 'include' : 'same-origin',
      body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params: { _meta: { 'io.modelcontextprotocol/protocolVersion': '2026-07-28', 'io.modelcontextprotocol/clientCapabilities': {} } } }),
    }, snapshot);
    const body = await response.text();
    this.assertNamespace(snapshot);
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
  listAccessResources(input = {}) { return this.nativeRequest(postV1StudioSdkAccessList, 'access.list', input); }
  listNamedPolicies() { return this.resourceAccess.listNamedPolicies(); }
  async listReleasePolicies() {
    const result = await this.nativeRequest(postNamedPolicies, 'authz.sdk.namedPolicies.manage', {operation:'releaseChoices'});
    // The shared named-policy contract omits an empty optional collection.
    if (result && !Array.isArray(result) && typeof result === 'object' && result.policies === undefined) return [];
    if (!Array.isArray(result?.policies)) throw new Error('Release policy choices are unavailable.');
    return result.policies;
  }
  createNamedPolicy(id, definition) { return this.resourceAccess.createNamedPolicy(id, definition); }
  updateNamedPolicy(reference, definition) { return this.resourceAccess.updateNamedPolicy(reference, definition); }
  namedPolicyImpact(reference) { return this.resourceAccess.namedPolicyImpact(reference); }
  assignNamedPolicy(resource, revision, actions, reference) { return this.resourceAccess.assignNamedPolicy(resource, revision, actions, reference); }
  getResourceAccess(resource) { return this.resourceAccess.getResourceAccess(resource); }
  getResourceAccessContext(resource) { return this.resourceAccess.getResourceAccessContext(resource); }
  createResourceAccess(document) { return this.resourceAccess.createResourceAccess(document); }
  replaceResourceAccess(document) { return this.resourceAccess.replaceResourceAccess(document); }
  checkCurrentAccess(resource, action) { return this.resourceAccess.checkCurrentAccess(resource, action); }
  async nativeRequest(call, operation, body) {
    const snapshot = this.namespaceSnapshot(operation);
    const headers = { Accept: 'application/json' };
    if (this.config.mode === 'development') headers['X-Studio-Development-Subject'] = this.config.development.subject;
    if (snapshot?.id) headers['X-Studio-Namespace'] = snapshot.id;
    if (this.nativeCallTool) {
      const name = operation.startsWith('authz.sdk.') ? operation : `studio.sdk.${operation}`;
      try {
        const value = await this.nativeCallTool(name, body ?? {}, this.nativeToolOptions(snapshot,headers));
        this.assertNamespace(snapshot);
        return value;
      } catch (error) {
        this.assertNamespace(snapshot);
        if (error?.status === 401) this.onUnauthorized?.();
        throw error;
      }
    }
    const { data, error, response } = await call({
      body, baseUrl: this.config.apiBaseURL, fetch: (value, init) => this.send(value, init, snapshot),
      credentials: this.identity ? 'omit' : this.config.mode === 'authenticated' ? 'include' : 'same-origin', headers,
      parseAs: 'json',
    });
    this.assertNamespace(snapshot);
    if (error) {
      if (response?.status === 401) this.onUnauthorized?.();
      const requestId = response?.headers?.get?.('X-Request-ID') || '';
      const baseMessage = error?.message || `Studio SDK operation ${operation} failed (${response?.status ?? 'network'})`;
      const failure = new Error(requestId ? `${baseMessage} · request ${requestId}` : baseMessage);
      failure.code = error?.code || nativeErrorCode[response?.status] || 'internal';
      failure.status = response?.status;
      failure.field = error?.field || '';
      failure.violations = error?.violations ?? [];
      failure.providerError = error?.providerError;
      failure.diagnostics = error?.diagnostics;
      failure.expectedEtag = error?.expectedEtag;
      failure.currentEtag = error?.currentEtag;
      failure.requestId = requestId;
      throw failure;
    }
    return data;
  }
  async listACL(reportId) { return (await this.nativeRequest(postV1StudioSdkAclList, 'acl.list', { reportId }))?.items ?? []; }
  upsertACL(input) { return this.nativeRequest(postV1StudioSdkAclUpsert, 'acl.upsert', input); }
  async deleteACL(reportId, subjectType, subjectId, etag) { await this.nativeRequest(postV1StudioSdkAclDelete, 'acl.delete', { reportId, subjectType, subjectId, etag }); }
  getResources(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkResourcesGet, 'resources.get', { reportId, versionNo }); }
	upsertResourceFile(input) { return this.nativeRequest(postV1StudioSdkResourcesUpsertFile, 'resources.upsert_file', input); }
	deleteResourceFile(reportId, versionNo, resourceId, expectedSourceRevision) { return this.nativeRequest(postV1StudioSdkResourcesDeleteFile, 'resources.delete_file', { reportId, versionNo, resourceId, expectedSourceRevision }); }
	upsertResourceFolder(input) { return this.nativeRequest(postV1StudioSdkResourcesUpsertFolder, 'resources.upsert_folder', input); }
	deleteResourceFolder(reportId, versionNo, folderId, expectedSourceRevision) { return this.nativeRequest(postV1StudioSdkResourcesDeleteFolder, 'resources.delete_folder', { reportId, versionNo, folderId, expectedSourceRevision }); }
	upsertSkillRoot(input) { return this.nativeRequest(postV1StudioSdkResourcesUpsertSkill, 'resources.upsert_skill', input); }
	deleteSkillRoot(reportId, versionNo, skillId, expectedSourceRevision) { return this.nativeRequest(postV1StudioSdkResourcesDeleteSkill, 'resources.delete_skill', { reportId, versionNo, skillId, expectedSourceRevision }); }
	applyReaderCommand(reportId, versionNo, command) { return this.nativeRequest(postV1StudioSdkVersionsBuilder, 'versions.builder', { reportId, versionNo, command }); }
  applyVersionEdit(reportId, versionNo, command) { return this.nativeRequest(postV1StudioSdkVersionsApply, 'versions.apply', { reportId, versionNo, command }); }
  testReaderView(reportId, versionNo, view, input = {}, limit = 50) { return this.nativeRequest(postV1StudioSdkVersionsTestView, 'versions.test_view', { reportId, versionNo, view, input: { input, limit } }); }
  testReaderRelation(reportId, versionNo, relation, input = {}, limit = 50) { return this.nativeRequest(postV1StudioSdkVersionsTestRelation, 'versions.test_relation', { reportId, versionNo, relation, input: { input, limit } }); }
  testCubeCompose(reportId, versionNo, input) { return this.nativeRequest(postV1StudioSdkVersionsTestCompose, 'versions.test_compose', { reportId, versionNo, input }); }
  warmupReader(reportId, versionNo) { return this.nativeRequest(postV1StudioSdkVersionsWarmup, 'versions.warmup', { reportId, versionNo }); }
  getWarmupRun(reportId, runId) { return this.nativeRequest(postV1StudioSdkVersionsWarmupGet, 'versions.warmup_get', { reportId, runId }); }
  listWarmupRuns(reportId, versionNo, input = {}) { return this.nativeRequest(postV1StudioSdkVersionsWarmupList, 'versions.warmup_list', { reportId, versionNo, input }); }
  previewReader(reportId, versionNo, input = {}, limit = 50, routePath = undefined, cube = false) { return this.nativeRequest(postV1StudioSdkPreviewExecute, 'preview.execute', { reportId, versionNo, input: { input, limit, cube } }); }
  listConnectors(input = {}) { return this.nativeRequest(postV1StudioSdkConnectorsList, 'connectors.list', input); }
  getConnector(name) { return this.nativeRequest(postV1StudioSdkConnectorsGet, 'connectors.get', { name }); }
  createConnector(input) { return this.nativeRequest(postV1StudioSdkConnectorsCreate, 'connectors.create', input); }
  updateConnector(name, input) { return this.nativeRequest(postV1StudioSdkConnectorsUpdate, 'connectors.update', { name, input }); }
  testConnector(name) { return this.nativeRequest(postV1StudioSdkConnectorsTest, 'connectors.test', { name }); }
  discoverConnector(name, input = {}) { return this.nativeRequest(postV1StudioSdkConnectorsDiscover, 'connectors.discover', { name, input }); }
  listSchemas(name, input = {}) { return this.nativeRequest(postV1StudioSdkConnectorsSchemas, 'connectors.schemas', { name, input }); }
  listTables(name, input = {}) { return this.nativeRequest(postV1StudioSdkConnectorsTables, 'connectors.tables', { name, input }); }
  getTable(name, input) { return this.nativeRequest(postV1StudioSdkConnectorsTable, 'connectors.table', { name, input }); }
  testSQL(name, input) { return this.nativeRequest(postV1StudioSdkConnectorsTestSql, 'connectors.test_sql', { name, input }); }
  activateConnector(name, etag) { return this.nativeRequest(postV1StudioSdkConnectorsActivate, 'connectors.activate', { name, etag }); }
  disableConnector(name, etag) { return this.nativeRequest(postV1StudioSdkConnectorsDisable, 'connectors.disable', { name, etag }); }
  async deleteConnector(name, etag) { await this.nativeRequest(postV1StudioSdkConnectorsDelete, 'connectors.delete', { name, etag }); }
  listNamespaceSkills(input = {}) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsList, 'namespace_skills.list', input); }
  getNamespaceSkill(skillId, revision = 0) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsGet, 'namespace_skills.get', { skillId, revision }); }
  createNamespaceSkill(bundle) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsCreate, 'namespace_skills.create', { bundle }); }
  updateNamespaceSkill(skillId, etag, bundle) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsUpdate, 'namespace_skills.update', { skillId, etag, bundle }); }
  deleteNamespaceSkill(skillId, etag) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsDelete, 'namespace_skills.delete', { skillId, etag }); }
  listNamespaceSkillRevisions(skillId, input = {}) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsRevisions, 'namespace_skills.revisions', { ...input, skillId }); }
  publishNamespaceSkill(skillId, etag, revision) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsPublish, 'namespace_skills.publish', { skillId, etag, revision }); }
  unpublishNamespaceSkill(skillId, etag) { return this.nativeRequest(postV1StudioSdkNamespaceSkillsUnpublish, 'namespace_skills.unpublish', { skillId, etag }); }
  listNamespaces(input = {}) { return this.nativeRequest(postV1StudioSdkNamespacesList, 'namespaces.list', input); }
  getNamespace(name) { return this.nativeRequest(postV1StudioSdkNamespacesGet, 'namespaces.get', { name }); }
  createNamespace(input) { return this.nativeRequest(postV1StudioSdkNamespacesCreate, 'namespaces.create', input); }
  updateNamespace(name, input) { return this.nativeRequest(postV1StudioSdkNamespacesUpdate, 'namespaces.update', { name, input }); }
  async deleteNamespace(name, etag) { await this.nativeRequest(postV1StudioSdkNamespacesDelete, 'namespaces.delete', { name, etag }); }
  listAuthorizationPredicates(input = {}) { return this.nativeRequest(postV1StudioSdkAuthorizationPredicatesList, 'authorization_predicates.list', input); }
  getAuthorizationPredicate(name) { return this.nativeRequest(postV1StudioSdkAuthorizationPredicatesGet, 'authorization_predicates.get', { name }); }
  async listAuthorizationPredicateTypes() { return (await this.nativeRequest(postV1StudioSdkAuthorizationPredicatesTypes, 'authorization_predicates.types', {}))?.items ?? []; }
  createAuthorizationPredicate(input) { return this.nativeRequest(postV1StudioSdkAuthorizationPredicatesCreate, 'authorization_predicates.create', input); }
  updateAuthorizationPredicate(name, input) { return this.nativeRequest(postV1StudioSdkAuthorizationPredicatesUpdate, 'authorization_predicates.update', { name, input }); }
  async deleteAuthorizationPredicate(name, etag) { await this.nativeRequest(postV1StudioSdkAuthorizationPredicatesDelete, 'authorization_predicates.delete', { name, etag }); }
}

import { defaultActionsByKind } from './resourceAccessActions.js';

export function liveACLReviewURL(resource, baseURL = '/') {
  if (!resource || !defaultActionsByKind[resource.kind]) return null;
  const params = new URLSearchParams({ mode: 'live', kind: resource.kind, id: resource.id, tenant: resource.tenant, version: resource.version });
  return `${baseURL}acl-review.html?${params}`;
}

export function liveACLReviewResource(params) {
  if (params.get('mode') !== 'live') return null;
  const resource = Object.fromEntries(['kind', 'id', 'tenant', 'version'].map(key => [key, params.get(key)?.trim() || '']));
  if (!defaultActionsByKind[resource.kind] || ['id', 'tenant', 'version'].some(key => !resource[key] || resource[key].length > 256 || /[\u0000-\u001f\u007f]/.test(resource[key]))) {
    throw new Error('A valid resource type, ID, tenant, and version are required for live ACL review.');
  }
  return resource;
}

export const reviewScenarios = {
  editable: 'Editable policy',
  readonly: 'Read-only access',
  denied: 'Access denied',
  unavailable: 'Provider unavailable',
  loading: 'Loading permissions',
  empty: 'No action policies',
  conflict: 'Revision conflict on save',
};

export function createReviewFixture(scenario = 'editable', kind = 'component') {
  const resource = { kind, id: 'operations-example', tenant: 'preview-tenant', version: '3' };
  const consume = kind === 'skill' ? 'retrieve' : 'execute';
  let document = { resource, revision: 7, policies: scenario === 'empty' ? {} : {
    discover: { mode: 'public' },
    describe: { mode: 'public' },
    [consume]: { mode: 'protected', entityType: 'project', rule: { kind: 'all', rules: [
      { kind: 'role', value: 'analyst' }, { kind: 'exposure', value: 'analytics' },
    ] } },
    viewAccess: { mode: 'protected', rule: { kind: 'role', value: 'access-admin' } },
    manageAccess: { mode: 'protected', rule: { kind: 'role', value: 'access-admin' } },
  } };
  const clone = value => JSON.parse(JSON.stringify(value));
  const choices = {
    subject: [{ id: 'example-alice', label: 'Alice Example' }, { id: 'example-sam', label: 'Sam Example' }],
    role: [{ id: 'analyst', label: 'Analyst' }, { id: 'reader', label: 'Reader' }, { id: 'access-admin', label: 'Access administrator' }],
    exposure: [{ id: 'analytics', label: 'Analytics' }, { id: 'exports', label: 'Exports' }],
    entity: [{ entity: { type: 'project', id: '101' }, label: 'Operations example' }, { entity: { type: 'organization', id: 'north' }, label: 'North example' }],
    entityTypes: ['project', 'organization'],
  };
  return {
    resource,
    actions: ['discover', 'describe', consume, 'export', 'edit', 'publish', 'viewAccess', 'manageAccess'],
    api: {
      async getResourceAccess() {
        if (scenario === 'loading') return new Promise(() => {});
        if (scenario === 'denied') throw new Error('Your identity cannot view this resource’s permissions. Ask its access administrator for access.');
        if (scenario === 'unavailable') throw new Error('The access provider is unavailable. No permission changes were applied. Retry when it is available.');
        return clone(document);
      },
      async getResourceAccessContext() {
        return { choices, canManage: scenario !== 'readonly', source: 'provider-directory' };
      },
      async replaceResourceAccess(next) {
        if (scenario === 'readonly') throw new Error('Permission changes are not allowed.');
        if (scenario === 'conflict') throw Object.assign(new Error('The saved revision changed.'), { code: 'conflict' });
        document = clone({ ...next, revision: document.revision + 1 });
        return clone(document);
      },
    },
  };
}

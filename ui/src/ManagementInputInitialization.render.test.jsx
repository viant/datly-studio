import React from 'react';
import {test, expect, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ManagementWindowRuntime from './ManagementWindowRuntime.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';

test('management input window retains its first editable name after delayed inspection and catalogs', async () => {
 let inspectReady, catalogsReady;
 const inspectionPending = new Promise(resolve => { inspectReady = resolve; });
 const catalogsPending = new Promise(resolve => { catalogsReady = resolve; });
 const inspection = {
  version: {versionNo: 1, sourceRevision: 9, state: 'draft'},
  capabilities: {canEdit: true},
  structure: {declarations: [], availablePredicates: [], component: {rootView: {name: 'reader', namespace: 'records', columns: [{name: 'ID', source: 'c.id'}], relations: []}}},
 };
 const complete = vi.fn();
 const api = {
  readManagementResource: async uri => managementResourceFixture(uri.split('/').at(-1)),
  fetchMCPDataSource: vi.fn(async source => {
   if (source.backend.method === 'studio.sdk.versions.inspect') { await inspectionPending; return inspection; }
   if (source.backend.method === 'studio.sdk.versions.builder') return {applied: true, inspection: {...inspection, version: {...inspection.version, sourceRevision: 10}}};
   await catalogsPending; return {items: []};
  }),
 };
 const user = userEvent.setup();
 render(<ManagementWindowRuntime api={api} uri="window://datastudio/reader-input" instanceKey="first-edit" parameters={{reportId: 'reader', versionNo: 1}} onComplete={complete}/>);
 const name = await screen.findByLabelText('Input name', {exact: true});
 expect(name.matches(':disabled')).toBe(true);
 await user.type(name, 'BeforeReady');
 expect(name.value).toBe('');
 inspectReady();
 await waitFor(() => expect(screen.getByLabelText('Input name', {exact: true}).matches(':disabled')).toBe(false));
 await user.type(screen.getByLabelText('Input name', {exact: true}), 'SearchName');
 catalogsReady();
 await user.type(screen.getByLabelText('Kind source', {exact: true}), 'searchName');
 expect(screen.getByLabelText('Input name', {exact: true}).value).toBe('SearchName');
 await user.click(screen.getByRole('button', {name: 'Save input', exact: true}));
 await waitFor(() => expect(complete).toHaveBeenCalled());
 const request = api.fetchMCPDataSource.mock.calls.find(([source]) => source.backend.method === 'studio.sdk.versions.builder')[1];
 expect(request.command.operation.field).toMatchObject({name: 'SearchName', sourceKind: 'query', sourceName: 'searchName'});
});

import React from 'react';
import {test, expect, vi} from 'vitest';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {StudioApp} from './StudioApp.jsx';
import {managementResourceFixture} from './managementWindowFixture.js';

test('authored workbench forwards graph, preview, and warmup metadata selectors for the selected version', async () => {
  const report = {id:'metadata-reader', title:'Metadata reader', ownerId:'owner', currentDraftVersion:2};
  const source = number => `SELECT ${number} AS id`;
  const versions = [1, 2].map(versionNo => ({reportId:report.id, versionNo, sourceRevision:versionNo+10, state:'draft', compileStatus:'valid'}));
  const api = {
    readManagementResource:vi.fn(async uri => managementResourceFixture(uri.split('/').at(-1))),
    invoke:vi.fn(async (operation, input) => {
      if (operation === 'components.list') return {items:[report]};
      if (operation === 'namespaces.list' || operation === 'connectors.list' || operation === 'versions.warmup.list') return {items:[]};
      if (operation === 'versions.list') return {items:versions};
      if (operation === 'publications.get') return {status:'unpublished'};
      if (operation === 'runtime.status') return {};
      if (operation === 'versions.inspect') {
        const version = versions.find(item => item.versionNo === input.versionNo);
        return {
          version, dql:source(version.versionNo), capabilities:{canRun:true, canPublish:true, canUseDql:true},
          structure:{component:{rootView:{name:'records', source:{sql:source(version.versionNo)}, columns:[{name:'id'}]}}},
          ...(input.includePreviewInput ? {previewInput:{...version, schema:{type:'object', properties:{id:{type:'integer'}}}}} : {}),
          ...(input.includeEffectiveWarmup ? {effectiveWarmup:{...version, status:'resolved', targets:[]}} : {}),
        };
      }
      throw new Error('Unexpected operation '+operation);
    }),
  };
  const inspections = () => api.invoke.mock.calls.filter(([operation]) => operation === 'versions.inspect').map(([, input]) => input);
  const graph = {discoverColumns:false, includePreviewInput:false, includeEffectiveWarmup:false};
  const user = userEvent.setup();
  window.history.replaceState({}, '', '/?section=components');
  try {
    render(<StudioApp api={api} subject="owner" mode="development"/>);
    await user.click(await screen.findByRole('button', {name:'Open Reader Builder', exact:true}));
    await screen.findByRole('heading', {name:'Component graph'});
    expect(inspections()).toEqual([{reportId:report.id, versionNo:2, ...graph}]);
    await user.selectOptions(screen.getByRole('combobox', {name:'Component version'}), '1');
    await waitFor(() => expect(inspections().at(-1)).toEqual({reportId:report.id, versionNo:1, ...graph}));
    await user.click(screen.getByRole('button', {name:'Run preview', exact:true}));
    await screen.findByRole('button', {name:'Run with parameters'});
    expect(inspections().at(-1)).toEqual({reportId:report.id, versionNo:1, ...graph, includePreviewInput:true});
    await user.click(screen.getByRole('button', {name:'Close preview parameters'}));
    await user.click(screen.getByRole('button', {name:'More actions'}));
    await user.click(await screen.findByRole('menuitem', {name:'Cache & warmup', exact:true}));
    await screen.findByText('No effective warmup plans for this saved version.', {exact:true});
    expect(inspections().at(-1)).toEqual({reportId:report.id, versionNo:1, ...graph, includeEffectiveWarmup:true});
    expect(api.invoke.mock.calls.some(([operation]) => ['preview.execute', 'versions.warmup', 'versions.builder', 'versions.apply'].includes(operation))).toBe(false);
    expect(versions.map(version => version.sourceRevision)).toEqual([11,12]);
  } finally {
    window.history.replaceState({}, '', '/');
  }
});

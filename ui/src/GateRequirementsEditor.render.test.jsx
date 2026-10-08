import React from 'react';
import {test, expect, vi} from 'vitest';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {GateRequirementsEditor} from './GateRequirementsEditor.jsx';

const resource = {kind: 'window', id: 'orders', version: '1', tenant: 'tenant'};
const document = {revision: 'r1', requirements: {schemaVersion: 1, requiredExposures: [], allowedRoles: []}};
const choices = {exposures: [{id: 'FEATURE_X', label: 'Feature X'}], roles: [{id: 'reader', label: 'Reader'}]};

test('reviews provider-owned feature and role choices before a CAS write', async () => {
  const user = userEvent.setup();
  const api = {
    getGateRequirements: vi.fn().mockResolvedValue(document),
    getGateRequirementsContext: vi.fn().mockResolvedValue({canManage: true, choices}),
    replaceGateRequirements: vi.fn().mockResolvedValue({revision: 'r2', requirements: {schemaVersion: 1, requiredExposures: ['FEATURE_X'], allowedRoles: ['reader']}}),
  };
  render(<GateRequirementsEditor api={api} resource={resource} action="execute"/>);
  await screen.findByText('Gate revision r1');
  await user.click(screen.getByLabelText('Feature X'));
  await user.click(screen.getByLabelText('Reader'));
  await user.click(screen.getByRole('button', {name: 'Review requirement changes'}));
  expect(api.replaceGateRequirements).not.toHaveBeenCalled();
  await user.click(screen.getByRole('button', {name: 'Save requirements'}));
  await screen.findByText('Requirements saved.');
  expect(api.replaceGateRequirements).toHaveBeenCalledWith({resource, action: 'execute', expectedRevision: 'r1', requirements: {schemaVersion: 1, requiredExposures: ['FEATURE_X'], allowedRoles: ['reader']}});
});

test('keeps a draft after revision conflict and respects read-only authority', async () => {
  const user = userEvent.setup();
  const api = {
    getGateRequirements: vi.fn().mockResolvedValue(document),
    getGateRequirementsContext: vi.fn().mockResolvedValue({canManage: true, choices}),
    replaceGateRequirements: vi.fn().mockRejectedValue(Object.assign(new Error('conflict'), {code: 'conflict'})),
  };
  const view = render(<GateRequirementsEditor api={api} resource={resource} action="execute"/>);
  await screen.findByText('Gate revision r1');
  await user.click(screen.getByLabelText('Feature X'));
  await user.click(screen.getByRole('button', {name: 'Review requirement changes'}));
  await user.click(screen.getByRole('button', {name: 'Save requirements'}));
  await screen.findByText('Requirements changed elsewhere');
  expect(screen.getByLabelText('Feature X').checked).toBe(true);
  view.unmount();
  const readOnlyAPI = {...api, getGateRequirementsContext: vi.fn().mockResolvedValue({canManage: false, choices})};
  render(<GateRequirementsEditor api={readOnlyAPI} resource={resource} action="execute"/>);
  await screen.findByText('You have read-only access.');
  expect(screen.queryByRole('button', {name: 'Review requirement changes'})).toBeNull();
});

test('shows a server-computed current-principal check for the saved revision', async () => {
  const user = userEvent.setup();
  const api = {
    getGateRequirements: vi.fn().mockResolvedValue(document),
    getGateRequirementsContext: vi.fn().mockResolvedValue({canManage: false, choices}),
    checkCurrentGate: vi.fn().mockResolvedValue({effect: 'deny', reasonCode: 'featureDisabled', requirementsRevision: 'r1', validUntil: new Date(Date.now() + 60_000).toISOString()}),
  };
  render(<GateRequirementsEditor api={api} resource={resource} action="execute" previewSelection={[{type: 'customer', id: '42'}]}/>);
  await screen.findByText('Gate revision r1');
  await user.click(screen.getByRole('button', {name: 'Check my current access'}));
  await screen.findByText('Denied at last check');
  expect(screen.getByText(/featureDisabled/)).toBeTruthy();
  expect(api.checkCurrentGate).toHaveBeenCalledWith(resource, 'execute', [{type: 'customer', id: '42'}]);
  expect(screen.queryByRole('button', {name: 'Save requirements'})).toBeNull();
});

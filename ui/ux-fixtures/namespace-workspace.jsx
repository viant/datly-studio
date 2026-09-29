import React from 'react';
import { createRoot } from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import '../src/studio.css';
import { StudioApp } from '../src/StudioApp.jsx';

const items = [
  { namespaceId: 'a'.repeat(64), name: 'forecasting', title: 'Forecasting', ownerId: 'fixture-owner', status: 'active', canManage: true },
  { namespaceId: 'b'.repeat(64), name: 'inventory', title: 'Inventory', ownerId: 'fixture-owner', status: 'active', canManage: true },
];
const api = {
  config: { apiBaseURL: 'ux-fixture:namespace-workspace' },
  setNamespace(id) { this.namespaceId = id; },
  setNamespaceBlocked() {},
  async listNamespaces() { return { items }; },
  async listComponents() { return { items: [] }; },
  async listConnectors() { return { items: [] }; },
  async getRuntimeStatus() { return { status: 'idle', readers: [] }; },
};
createRoot(document.getElementById('root')).render(<StudioApp api={api} mode="authenticated" subject="fixture-owner"/>);

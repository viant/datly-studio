import React from 'react';
import { createRoot } from 'react-dom/client';
import { StudioShell } from './StudioShell.jsx';
export { StudioAPI } from './studioApi.js';
export { StudioApp } from './StudioApp.jsx';
export { StudioShell } from './StudioShell.jsx';
export { createStudioSDK, defineStudioExtension } from './extensions.js';
export { ResourceAccessEditor } from './ResourceAccessEditor.jsx';
export { PermissionsWorkspace } from './SecurityCenter.jsx';
export { defaultActionsByKind } from './resourceAccessActions.js';

export function mountStudio(element, { config, sdk, extensions } = {}) {
  if (!element) throw new TypeError('mountStudio requires a DOM element');
  const root = createRoot(element);
  root.render(<StudioShell config={config} extensions={extensions ?? sdk?.extensions?.() ?? []}/>);
  return { unmount: () => root.unmount() };
}

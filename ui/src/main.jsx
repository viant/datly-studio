import React from 'react';
import { createRoot } from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import './studio.css';
import { loadConfig } from './config.js';
import { StudioShell } from './StudioShell.jsx';

async function start() {
  const config = await loadConfig();
  createRoot(document.getElementById('root')).render(<StudioShell config={config} />);
}

start().catch((error) => {
  document.getElementById('root').textContent = `Datly Studio could not start: ${error.message}`;
});

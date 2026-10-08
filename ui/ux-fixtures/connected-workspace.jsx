// Local acceptance surface. Configure only a loopback test SDK host; this page
// is not part of the production Vite entry points.
import React from 'react';
import { createRoot } from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import '../src/studio.css';
import { StudioShell } from '../src/StudioShell.jsx';
import { validateConfig } from '../src/config.js';

const params = new URLSearchParams(location.search);
const config = validateConfig({ mode: 'development', apiBaseURL: params.get('api') || 'http://127.0.0.1:8188', development: { subject: params.get('subject') || 'awitas' } });
createRoot(document.getElementById('root')).render(<StudioShell config={config}/>);

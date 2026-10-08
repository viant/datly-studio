// Lightweight public permissions entry point for embedding applications.
// It does not load the full Studio shell or its editor dependencies.
export {ResourceAccessEditor} from './ResourceAccessEditor.jsx';
export {GateRequirementsEditor} from './GateRequirementsEditor.jsx';
export {accessMCPAdapter} from './accessMCPAdapter.js';
export {gateMCPAdapter} from './gateMCPAdapter.js';
export {authzHTTPCall, accessHTTPAdapter, gateHTTPAdapter} from './authzHTTPAdapter.js';

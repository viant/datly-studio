import {gateContractAPI} from 'forge/gate-requirements-editor/contract';
export function gateMCPAdapter(callTool) {
 if(typeof callTool!=='function')throw new TypeError('trusted MCP call function is required');
 return gateContractAPI((operation,input)=>callTool('authz.sdk.gates.'+operation,input));
}

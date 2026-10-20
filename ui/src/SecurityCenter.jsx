import React from 'react';
import {ManagementWindow} from './ManagementWindow.jsx';

// Stable embedding entry points; the canonical window owns all rendering.
export function SecurityCenter({api,...parameters}) {return <ManagementWindow api={api} uri="window://datastudio/security" parameters={parameters}/>;}
export function PermissionsWorkspace({api,...parameters}) {return <ManagementWindow api={api} uri="window://datastudio/security" parameters={{...parameters,permissionsOnly:true}}/>;}

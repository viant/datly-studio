import React from 'react';
import {createRoot} from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import '../src/studio.css';
import {NamespaceDialog} from '../src/NamespaceDialog.jsx';
const api={createNamespace:async()=>{throw new Error('UX fixture: changes are not persisted.');},updateNamespace:async()=>{throw new Error('UX fixture: changes are not persisted.');}};
createRoot(document.getElementById('root')).render(<NamespaceDialog api={api} namespace={{name:'forecasting',title:'Forecasting',ownerId:'fixture-owner',status:'active',visibility:'private',allowedRoles:['forecast_reader'],mcpEnabled:true,mcpPort:8591,etag:1}} isOpen onClose={()=>{}} onSaved={()=>{}}/>);

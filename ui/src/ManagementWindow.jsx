import React, {useEffect, useId, useMemo, useState} from 'react';
import {Button, Callout, Spinner} from '@blueprintjs/core';
import {WindowContent} from 'forge/components';
import {SettingProvider} from 'forge/core';
import 'forge';
import {managementWindowMetadata, managementWindowTransport} from './managementWindow.js';

// The shell owns loading and connection context only. Layout and all screen
// actions are supplied by the same resource returned to remote Forge hosts.
export function ManagementWindow({api, uri, instanceKey = uri, onOperationCompleted}) {
  const mountId = useId();
  const windowId = `${instanceKey}:${mountId}`;
  const [state, setState] = useState({});
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    setState({});
    api.readManagementResource(uri).then(resource => {
      const metadata = managementWindowMetadata(resource);
      if (active) setState({metadata});
    }).catch(error => {if (active) setState({error});});
    return () => {active = false;};
  }, [api, uri, attempt]);
  const services = useMemo(() => state.metadata ? {dataConnectorTransport: managementWindowTransport(api, state.metadata, onOperationCompleted)} : {}, [api, state.metadata, onOperationCompleted]);
  if (state.error) return <Callout intent="danger" title="Unable to open this window">{state.error.message} <Button onClick={() => setAttempt(value => value + 1)}>Retry</Button></Callout>;
  if (!state.metadata) return <Spinner aria-label="Loading window" size={24}/>;
  return <SettingProvider connectorConfig={{window: {service: {uri}}}} services={services}><WindowContent key={windowId} window={{windowId, windowKey: state.metadata.windowKey}} selectedMetadata={state.metadata}/></SettingProvider>;
}

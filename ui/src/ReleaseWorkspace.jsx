import React, {useMemo} from 'react';
import ReleaseManager from 'forge/release-manager';
import {createReleaseMCPAdapter} from 'forge/release-manager/adapter';

export default function ReleaseWorkspace({api, namespace}) {
  const adapter = useMemo(() => createReleaseMCPAdapter({
    namespace,
    callTool: (method,input) => api.callReleaseTool(method,input),
    listPolicies: () => api.listReleasePolicies(),
  }),[api,namespace]);
  return <ReleaseManager api={adapter} channelKey={{namespace}}/>;
}

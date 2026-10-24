import React from 'react';
import {expect,test,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ReleaseWorkspace from './ReleaseWorkspace.jsx';

vi.mock('forge/release-manager',()=>({default:({api})=><button onClick={()=>api.policies()}>Load release policies</button>}));

test('Releases uses server-filtered choices and leaves Security policy inventory untouched',async()=>{
  const api={callReleaseTool:vi.fn(),listReleasePolicies:vi.fn(async()=>[]),listNamedPolicies:vi.fn(async()=>[{displayName:'Legacy'}])};
  render(<ReleaseWorkspace api={api} namespace="sales"/>);
  await userEvent.setup().click(screen.getByRole('button',{name:'Load release policies'}));
  await waitFor(()=>expect(api.listReleasePolicies).toHaveBeenCalledOnce());
  expect(api.listNamedPolicies).not.toHaveBeenCalled();
});

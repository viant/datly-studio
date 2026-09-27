import React from 'react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./StudioApp.jsx',()=>({StudioApp:({api,subject,brand,onSignOut})=><div><span>Ready as {subject}</span><span>Brand {brand}</span><button onClick={()=>api.listComponents().catch(()=>{})}>Trigger expired API</button>{onSignOut&&<button onClick={()=>onSignOut().catch(()=>{})}>Sign out</button>}</div>}));

import { StudioShell } from './StudioShell.jsx';

const config={mode:'authenticated',apiBaseURL:'https://studio.example.com',authentication:{mePath:'/v1/studio/auth/me',loginPath:'/v1/studio/auth/login'}};
const response=(payload,status=200)=>({ok:status>=200&&status<300,status,headers:{get:()=>''},json:async()=>payload});
const directConfig={mode:'authenticated',apiBaseURL:'https://datly-static.example.com',mcpBaseURL:'https://datly-dynamic.example.com',
  authentication:{mode:'identity-token',authBaseURL:window.location.origin,loginPath:'/v1/studio/auth/login',tokenPath:'/v1/studio/auth/token'}};

afterEach(()=>vi.unstubAllGlobals());

describe('StudioShell authentication boundary',()=>{
  test('restores an in-memory ID token and signs in through the same-origin auth service',async()=>{
    const fetcher=vi.fn().mockResolvedValue(response({id_token:'signed-id',subject:'owner'}));
    vi.stubGlobal('fetch',fetcher);
    render(<StudioShell config={directConfig}/>);
    expect(await screen.findByText('Ready as owner')).toBeTruthy();
    expect(fetcher.mock.calls[0][0]).toBe(`${window.location.origin}/v1/studio/auth/token`);
    expect(fetcher.mock.calls[0][1].credentials).toBe('include');
  });
  test('offers same-origin sign-in when the auth cookie is absent',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(response({},401)));
    render(<StudioShell config={directConfig}/>);
    const link=await screen.findByRole('link',{name:'Continue to sign in'});
    expect(link.getAttribute('href')).toBe(`${window.location.origin}/v1/studio/auth/login`);
  });
  test('returns to sign-in only after direct-auth logout succeeds',async()=>{
    const user=userEvent.setup();
    const fetcher=vi.fn().mockResolvedValueOnce(response({id_token:'signed-id',subject:'owner'}))
      .mockResolvedValueOnce(response(null,204));
    vi.stubGlobal('fetch',fetcher);
    render(<StudioShell config={directConfig}/>);
    await user.click(await screen.findByRole('button',{name:'Sign out'}));
    expect(await screen.findByRole('heading',{name:'Sign in to Datly Studio'})).toBeTruthy();
    expect(fetcher.mock.calls[1][1]).toEqual(expect.objectContaining({method:'DELETE',credentials:'include'}));
  });
  test('passes optional host branding through the public embedding seam',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(response({authenticated:true,subject:'owner'})));
    render(<StudioShell config={{...config,brand:'Acme Portal'}}/>);
    expect(await screen.findByText('Brand Acme Portal')).toBeTruthy();
  });
  test('shows an intentional loading state while restoring the HttpOnly session',async()=>{
    vi.stubGlobal('fetch',vi.fn(()=>new Promise(()=>{})));
    render(<StudioShell config={config}/>);
    expect(screen.getByRole('heading',{name:'Checking Studio session'})).toBeTruthy();
    expect(screen.getByText(/secure server-side session/i)).toBeTruthy();
  });

  test('routes a missing session to deployment sign-in without browser token controls',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(response({authenticated:false},401)));
    render(<StudioShell config={config}/>);
    const link=await screen.findByRole('link',{name:'Continue to sign in'});
    expect(link.getAttribute('href')).toBe('https://studio.example.com/v1/studio/auth/login');
    expect(screen.getByText(/tokens are never stored in this browser/i)).toBeTruthy();
    expect(document.body.textContent).not.toMatch(/bearer|access token|refresh token/i);
  });

  test('recovers an unavailable session check and returns expired SDK calls to sign-in',async()=>{
    const user=userEvent.setup();
    const fetcher=vi.fn()
      .mockRejectedValueOnce(new Error('identity provider unavailable'))
      .mockResolvedValueOnce(response({authenticated:true,subject:'alice'}))
      .mockResolvedValueOnce(response({message:'expired'},401));
    vi.stubGlobal('fetch',fetcher);
    render(<StudioShell config={config}/>);
    expect((await screen.findByRole('alert')).textContent).toContain('identity provider unavailable');
    await user.click(screen.getByRole('button',{name:'Try again'}));
    expect(await screen.findByText('Ready as alice')).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Trigger expired API'}));
    expect(await screen.findByRole('heading',{name:'Sign in to Datly Studio'})).toBeTruthy();
  });
});

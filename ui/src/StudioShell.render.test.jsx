import React from 'react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./StudioApp.jsx',()=>({StudioApp:({api,subject,brand})=><div><span>Ready as {subject}</span><span>Brand {brand}</span><button onClick={()=>api.listReports().catch(()=>{})}>Trigger expired API</button></div>}));

import { StudioShell } from './StudioShell.jsx';

const config={mode:'authenticated',apiBaseURL:'https://studio.example.com',authentication:{mePath:'/v1/studio/auth/me',loginPath:'/v1/studio/auth/login'}};
const response=(payload,status=200)=>({ok:status>=200&&status<300,status,headers:{get:()=>''},json:async()=>payload});

afterEach(()=>vi.unstubAllGlobals());

describe('StudioShell authentication boundary',()=>{
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

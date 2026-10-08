// Local-only synthetic identity acceptance. Never included in production entry points.
import React, {useState} from 'react';
import {createRoot} from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import '../src/studio.css';
import {StudioShell} from '../src/StudioShell.jsx';
import {validateConfig} from '../src/config.js';
const config=validateConfig({mode:'authenticated',apiBaseURL:'http://127.0.0.1:8190',authentication:{mode:'bff'}});
function Fixture(){
 const [ready,setReady]=useState(false),[token,setToken]=useState(''),[error,setError]=useState('');
 async function signIn(event){event.preventDefault();setError('');try{
  const response=await fetch(config.apiBaseURL+'/v1/studio/auth/session',{method:'POST',headers:{Authorization:'Bearer '+token},credentials:'include'});
  setToken('');if(!response.ok)throw new Error('Synthetic identity verification failed');setReady(true);
 }catch(cause){setToken('');setError(cause.message)}}
 if(ready)return <StudioShell config={config}/>;
 return <main className="studio-auth-shell"><form className="studio-card" onSubmit={signIn}><h1>Synthetic forecasting identity</h1><p>Isolated test catalog and fabricated forecasting data.</p><label>Fixture token<input type="password" autoComplete="off" value={token} onChange={event=>setToken(event.target.value)}/></label><button type="submit">Use synthetic identity</button>{error&&<p role="alert">{error}</p>}</form></main>;
}
createRoot(document.getElementById('root')).render(<Fixture/>);

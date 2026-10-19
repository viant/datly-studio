import React from 'react';
import {expect,test,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {ConnectorDialog} from './ConnectorDialog.jsx';

test('shows saved templates and preserves them when saving an unrelated edit',async()=>{
 const user=userEvent.setup();
 const api={updateConnector:vi.fn().mockResolvedValue({name:'warehouse'})};
 const connector={name:'warehouse',driver:'mysql',etag:4,status:'active',dsnConfigured:true,secretConfigured:true,dsnDisplay:'$Username:$Password@tcp(db:3306)/reporting',secretRefDisplay:'/secrets/mysql.enc|blowfish://default'};
 render(<ConnectorDialog api={api} connector={connector} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 expect(screen.getByLabelText('Current DSN template').value).toBe(connector.dsnDisplay);
 expect(screen.getByLabelText('Current DSN template').readOnly).toBe(true);
 expect(screen.getByLabelText('Current secret reference').value).toBe(connector.secretRefDisplay);
 await user.type(screen.getByLabelText('Description'),'Updated description');
 await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledOnce());
 const [name,input]=api.updateConnector.mock.calls[0];
 expect(name).toBe('warehouse');expect(input.etag).toBe(4);
 expect(input).not.toHaveProperty('dsnTemplate');expect(input).not.toHaveProperty('secretRef');
});

test('replacement DSN is submitted only when explicitly entered',async()=>{
 const user=userEvent.setup();const api={updateConnector:vi.fn().mockResolvedValue({name:'warehouse'})};
 render(<ConnectorDialog api={api} connector={{name:'warehouse',driver:'bigquery',etag:2,status:'draft',dsnConfigured:true,dsnDisplay:'bigquery://old/data'}} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 await user.type(screen.getByLabelText('New DSN template'),'bigquery://new/data');
 await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledOnce());
 expect(api.updateConnector.mock.calls[0][1].dsnTemplate).toBe('bigquery://new/data');
});

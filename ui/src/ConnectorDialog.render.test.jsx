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
 expect(screen.getByLabelText('Current secret URL').value).toBe('/secrets/mysql.enc');
 expect(screen.getByLabelText('Current secret key').value).toBe('blowfish://default');
 expect(screen.getByLabelText('Current secret URL').readOnly).toBe(true);
 expect(screen.getByLabelText('Current secret key').readOnly).toBe(true);
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


test('creates a connector with separate SCY secret URL and key fields',async()=>{
 const user=userEvent.setup();const api={createConnector:vi.fn().mockResolvedValue({name:'reporting_mysql'})};
 render(<ConnectorDialog api={api} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 await user.type(screen.getByLabelText('Name'),'reporting_mysql');
 await user.selectOptions(screen.getByLabelText('Provider'),'mysql');
 await user.click(screen.getByRole('button',{name:'Advanced provider settings'}));
 await user.type(screen.getByLabelText('Secret URL'),'file:///Users/awitas/.secret/mysql-studio.enc');
 await user.type(screen.getByLabelText('Secret key'),'blowfish://default');
 await user.click(screen.getByRole('button',{name:'Create connector'}));
 await waitFor(()=>expect(api.createConnector).toHaveBeenCalledOnce());
 expect(api.createConnector.mock.calls[0][0].secretRef).toBe('file:///Users/awitas/.secret/mysql-studio.enc|blowfish://default');
});

test('replaces and explicitly removes a saved secret without reusing display values',async()=>{
 const user=userEvent.setup();const api={updateConnector:vi.fn().mockResolvedValue({name:'warehouse'})};
 render(<ConnectorDialog api={api} connector={{name:'warehouse',driver:'mysql',etag:4,status:'draft',secretConfigured:true,secretRefDisplay:'file:///config/old.enc|blowfish://env/OLD'}} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 expect(screen.getByLabelText('New secret URL').value).toBe('');expect(screen.getByLabelText('New secret key').value).toBe('');
 await user.type(screen.getByLabelText('New secret URL'),'file:///config/new.enc');
 await user.type(screen.getByLabelText('New secret key'),'blowfish://default');
 await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledOnce());
 expect(api.updateConnector.mock.calls[0][1].secretRef).toBe('file:///config/new.enc|blowfish://default');
 await user.click(screen.getByLabelText('Remove the existing secret reference'));
 expect(screen.getByLabelText('New secret URL').disabled).toBe(true);expect(screen.getByLabelText('New secret key').disabled).toBe(true);
 await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledTimes(2));
 expect(api.updateConnector.mock.calls[1][1].secretRef).toBe('');
});

test('shows URL-only saved refs and rejects a replacement key without a URL',async()=>{
 const user=userEvent.setup();const api={updateConnector:vi.fn()};
 render(<ConnectorDialog api={api} connector={{name:'warehouse',driver:'mysql',etag:4,status:'draft',secretConfigured:true,secretRefDisplay:'file:///config/client.json'}} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 expect(screen.getByLabelText('Current secret URL').value).toBe('file:///config/client.json');expect(screen.getByLabelText('Current secret key').value).toBe('');
 await user.type(screen.getByLabelText('New secret key'),'blowfish://default');
 await user.click(screen.getByRole('button',{name:'Save connector'}));
 expect(await screen.findByRole('alert')).toHaveProperty('textContent','Enter the secret URL together with its key reference.');
 expect(api.updateConnector).not.toHaveBeenCalled();
});


test('imports full connector YAML without changing DSN placeholders',async()=>{
 const user=userEvent.setup();const api={createConnector:vi.fn().mockResolvedValue({name:'reporting_mysql'})};
 render(<ConnectorDialog api={api} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 await user.type(screen.getByLabelText('Name'),'reporting_mysql');
 await user.click(screen.getByRole('button',{name:'Paste YAML configuration'}));
 const yaml=["Driver: mysql","DSN: '${Username}:${Password}@tcp(127.0.0.1:3306)/aistudio?parseTime=true&charset=utf8mb4'","Secret:","  URL: file:///Users/awitas/.secret/mysql-studio.enc","  Key: blowfish://default"].join('\n');
 await user.click(screen.getByLabelText('Connector YAML'));await user.paste(yaml);
 await user.click(screen.getByRole('button',{name:'Apply YAML'}));
 expect(screen.getByLabelText('Provider').value).toBe('mysql');
 expect(screen.getByLabelText('DSN template').value).toBe('${Username}:${Password}@tcp(127.0.0.1:3306)/aistudio?parseTime=true&charset=utf8mb4');
 expect(api.createConnector).not.toHaveBeenCalled();
 await user.click(screen.getByRole('button',{name:'Create connector'}));
 await waitFor(()=>expect(api.createConnector).toHaveBeenCalledOnce());
 expect(api.createConnector.mock.calls[0][0]).toMatchObject({driver:'mysql',dsnTemplate:'${Username}:${Password}@tcp(127.0.0.1:3306)/aistudio?parseTime=true&charset=utf8mb4',secretRef:'file:///Users/awitas/.secret/mysql-studio.enc|blowfish://default'});
});

test('YAML edits preserve omitted secrets and require explicit null to remove',async()=>{
 const user=userEvent.setup();const api={updateConnector:vi.fn().mockResolvedValue({name:'warehouse'})};
 render(<ConnectorDialog api={api} connector={{name:'warehouse',driver:'mysql',etag:4,status:'draft',secretConfigured:true}} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 await user.click(screen.getByRole('button',{name:'Paste YAML configuration'}));
 await user.click(screen.getByLabelText('Connector YAML'));await user.paste('Driver: mysql');
 await user.click(screen.getByRole('button',{name:'Apply YAML'}));await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledOnce());expect(api.updateConnector.mock.calls[0][1]).not.toHaveProperty('secretRef');
 await user.clear(screen.getByLabelText('Connector YAML'));await user.click(screen.getByLabelText('Connector YAML'));await user.paste('Secret: null');
 await user.click(screen.getByRole('button',{name:'Apply YAML'}));await user.click(screen.getByRole('button',{name:'Save connector'}));
 await waitFor(()=>expect(api.updateConnector).toHaveBeenCalledTimes(2));expect(api.updateConnector.mock.calls[1][1].secretRef).toBe('');
});

test.each(['Driver: mysql\nDriver: postgres','Driver: mysql\nSecret: {Key: blowfish://default}','Driver: mysql\nUnrecognized: true'])('rejects invalid YAML without modifying the draft: %s',async yaml=>{
 const user=userEvent.setup();const api={createConnector:vi.fn()};
 render(<ConnectorDialog api={api} isOpen onClose={vi.fn()} onCreated={vi.fn()}/>);
 await user.click(screen.getByRole('button',{name:'Paste YAML configuration'}));await user.click(screen.getByLabelText('Connector YAML'));await user.paste(yaml);await user.click(screen.getByRole('button',{name:'Apply YAML'}));
 expect(await screen.findByRole('alert')).toBeTruthy();expect(screen.getByLabelText('Provider').value).toBe('sqlite');expect(api.createConnector).not.toHaveBeenCalled();
});

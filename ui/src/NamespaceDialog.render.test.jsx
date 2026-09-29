import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NamespaceDialog } from './NamespaceDialog.jsx';

const namespace={name:'inventory.forecasting',title:'Forecasting',description:'Planning readers',status:'active',etag:3};

describe('NamespaceDialog',()=>{
  test('preserves a stale edit and reloads the authoritative namespace',async()=>{
    const user=userEvent.setup();
    const conflict=Object.assign(new Error('namespace etag does not match'),{code:'conflict'});
    const api={updateNamespace:vi.fn().mockRejectedValue(conflict),getNamespace:vi.fn().mockResolvedValue({...namespace,title:'Forecasting production',etag:4})};
    const onSaved=vi.fn();
    render(<NamespaceDialog api={api} namespace={namespace} isOpen onClose={vi.fn()} onSaved={onSaved}/>);

    const title=screen.getByLabelText('Display title');
    await user.clear(title);await user.type(title,'My unsaved title');
    await user.click(screen.getByRole('button',{name:'Save namespace'}));
    expect((await screen.findByRole('alert')).textContent).toContain('Your change was not applied');
    expect(title.value).toBe('My unsaved title');
    expect(onSaved).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button',{name:'Reload namespace'}));
    expect(await screen.findByDisplayValue('Forecasting production')).toBeTruthy();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(api.getNamespace).toHaveBeenCalledWith('inventory.forecasting');
  });

  test('keeps entered create values after server rejection',async()=>{
    const user=userEvent.setup();
    const api={createNamespace:vi.fn().mockRejectedValue(new Error('namespace already exists'))};
    render(<NamespaceDialog api={api} namespace={null} isOpen onClose={vi.fn()} onSaved={vi.fn()}/>);
    await user.type(screen.getByLabelText('Namespace'),'finance.reporting');
    await user.type(screen.getByLabelText('Display title'),'Finance reporting');
    await user.click(screen.getByRole('button',{name:'Create namespace'}));
    expect((await screen.findByRole('alert')).textContent).toContain('namespace already exists');
    expect(screen.getByLabelText('Namespace').value).toBe('finance.reporting');
    expect(screen.getByLabelText('Display title').value).toBe('Finance reporting');
  });
  test('creates private namespace with exact viewer roles and optional MCP port',async()=>{
    const user=userEvent.setup();
    const api={createNamespace:vi.fn().mockResolvedValue({name:'forecasting'})};
    render(<NamespaceDialog api={api} isOpen onClose={vi.fn()} onSaved={vi.fn()}/>);
    expect(screen.getByLabelText('Visibility').value).toBe('private');
    expect(screen.getByLabelText('Expose MCP').checked).toBe(false);
    await user.type(screen.getByLabelText('Namespace'),'forecasting');
    await user.type(screen.getByLabelText('Display title'),'Forecasting');
    await user.type(screen.getByLabelText('Viewer roles'),'forecast_reader{Enter}');
    await user.click(screen.getByLabelText('Expose MCP'));
    await user.clear(screen.getByLabelText('MCP port'));await user.type(screen.getByLabelText('MCP port'),'8591');
    await user.click(screen.getByRole('button',{name:'Create namespace'}));
    expect(api.createNamespace).toHaveBeenCalledWith({name:'forecasting',title:'Forecasting',description:'',visibility:'private',allowedRoles:['forecast_reader'],mcpEnabled:true,mcpPort:8591});
  });

  test('blocks invalid enabled MCP port without losing values',async()=>{
    const user=userEvent.setup();const api={createNamespace:vi.fn()};
    render(<NamespaceDialog api={api} isOpen onClose={vi.fn()} onSaved={vi.fn()}/>);
    await user.type(screen.getByLabelText('Namespace'),'forecasting');await user.type(screen.getByLabelText('Display title'),'Forecasting');
    await user.click(screen.getByLabelText('Expose MCP'));
    await user.clear(screen.getByLabelText('MCP port'));await user.type(screen.getByLabelText('MCP port'),'70000');
    await user.click(screen.getByRole('button',{name:'Create namespace'}));
    expect((await screen.findByRole('alert')).textContent).toContain('0 to 65535');
    expect(api.createNamespace).not.toHaveBeenCalled();expect(screen.getByLabelText('MCP port').value).toBe('70000');
  });

  test('viewer can inspect settings and close without management controls',async()=>{
    const user=userEvent.setup();const onClose=vi.fn();const api={updateNamespace:vi.fn()};
    render(<NamespaceDialog api={api} namespace={{...namespace,ownerId:'owner',canManage:false}} isOpen onClose={onClose} onSaved={vi.fn()}/>);
    expect(screen.getByLabelText('Display title').disabled).toBe(true);
    expect(screen.getByLabelText('Visibility').disabled).toBe(true);
    expect(screen.queryByRole('button',{name:'Save namespace'})).toBeNull();
    await user.click(screen.getAllByRole('button',{name:'Close',exact:true}).at(-1));
    expect(onClose).toHaveBeenCalled();expect(api.updateNamespace).not.toHaveBeenCalled();
  });

  test('save includes a typed role without requiring Enter',async()=>{
    const user=userEvent.setup();const api={createNamespace:vi.fn().mockResolvedValue({name:'forecasting'})};
    render(<NamespaceDialog api={api} isOpen onClose={vi.fn()} onSaved={vi.fn()}/>);
    await user.type(screen.getByLabelText('Namespace'),'forecasting');await user.type(screen.getByLabelText('Display title'),'Forecasting');
    await user.type(screen.getByLabelText('Viewer roles'),'forecast_reader');
    await user.click(screen.getByRole('button',{name:'Create namespace'}));
    expect(api.createNamespace.mock.calls[0][0].allowedRoles).toEqual(['forecast_reader']);
  });

  test('saving blocks Escape, header close and cancel',async()=>{
    const user=userEvent.setup();let finish;const pending=new Promise((resolve)=>{finish=resolve;});
    const api={updateNamespace:vi.fn().mockReturnValue(pending)};const onClose=vi.fn();
    render(<NamespaceDialog api={api} namespace={namespace} isOpen onClose={onClose} onSaved={vi.fn()}/>);
    await user.click(screen.getByRole('button',{name:'Save namespace'}));
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('button',{name:'Close',exact:true})).toBeNull();
    expect(screen.getByRole('button',{name:'Cancel'}).disabled).toBe(true);
    expect(onClose).not.toHaveBeenCalled();
    finish(namespace);await waitFor(()=>expect(onClose).toHaveBeenCalledOnce());
  });

  test('conflict reload updates owner and management capability',async()=>{
    const user=userEvent.setup();const conflict=Object.assign(new Error('stale'),{code:'conflict'});
    const api={updateNamespace:vi.fn().mockRejectedValue(conflict),getNamespace:vi.fn().mockResolvedValue({...namespace,ownerId:'new-owner',canManage:false,etag:4})};
    render(<NamespaceDialog api={api} namespace={{...namespace,ownerId:'owner',canManage:true}} isOpen onClose={vi.fn()} onSaved={vi.fn()}/>);
    await user.click(screen.getByRole('button',{name:'Save namespace'}));await screen.findByRole('alert');
    await user.click(screen.getByRole('button',{name:'Reload namespace'}));
    await waitFor(()=>expect(screen.queryByRole('button',{name:'Save namespace'})).toBeNull());
    expect(screen.getByLabelText('Display title').disabled).toBe(true);expect(screen.getByText('Owner: new-owner')).toBeTruthy();
  });

});

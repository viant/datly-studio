import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
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
});

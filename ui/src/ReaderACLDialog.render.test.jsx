import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderACLDialog } from './ReaderACLDialog.jsx';

describe('ReaderACLDialog',()=>{
  test('applies least-privilege dependencies and recovers a stale grant',async()=>{
    const user=userEvent.setup();
    const stale=Object.assign(new Error('grant etag does not match'),{code:'conflict'});
    const api={listACL:vi.fn().mockResolvedValueOnce([]).mockResolvedValueOnce([{subjectType:'user',subjectId:'analyst',canView:true,canRun:true,canEdit:false,canPublish:false,canUseDql:false,etag:4}]),upsertACL:vi.fn().mockRejectedValue(stale)};
    render(<ReaderACLDialog isOpen api={api} report={{id:'vendor'}} onClose={vi.fn()}/>);

    expect(await screen.findByText(/No delegated access/)).toBeTruthy();
    await user.type(screen.getByLabelText('JWT subject'),'analyst');
    await user.click(screen.getByRole('button',{name:'Advanced author'}));
    expect(screen.getByRole('checkbox',{name:'View component and catalog'}).checked).toBe(true);
    expect(screen.getByRole('checkbox',{name:'Run reader and previews'}).checked).toBe(true);
    expect(screen.getByRole('checkbox',{name:'Edit SQL and structured settings'}).checked).toBe(true);
    expect(screen.getByRole('checkbox',{name:'Use advanced structural DQL'}).checked).toBe(true);
    await user.click(screen.getByRole('button',{name:'Create grant'}));
    expect((await screen.findByRole('alert')).textContent).toContain('No access change was applied');
    expect(screen.getByLabelText('JWT subject').value).toBe('analyst');

    await user.click(screen.getByRole('button',{name:'Reload permissions'}));
    expect(await screen.findByText('analyst')).toBeTruthy();
    expect(api.listACL).toHaveBeenCalledTimes(2);
  });
});

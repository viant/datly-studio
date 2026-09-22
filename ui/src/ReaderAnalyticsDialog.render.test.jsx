import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('./LazyEditor.jsx',()=>({LazyEditor:({value,onChange,ariaLabel})=><textarea aria-label={ariaLabel} value={value} onChange={(event)=>onChange(event.target.value)}/>}));
import { ReaderAnalyticsDialog } from './ReaderAnalyticsDialog.jsx';

const structure={component:{rootView:{columns:[{name:'STATUS',source:'STATUS',groupable:true},{name:'TOTAL',source:'TOTAL'}]},settings:{report:{enabled:true,compose:{enabled:true,maxCubes:2,maxLimit:50,timeoutMs:12000}}}}};

describe('ReaderAnalyticsDialog',()=>{
  test('builds ordered typed frames and sends the exact composition request',async()=>{
    const user=userEvent.setup();
    const onTestCompose=vi.fn().mockResolvedValue({data:[{STATUS:'active',TOTAL:3}]});
    render(<ReaderAnalyticsDialog isOpen structure={structure} onClose={vi.fn()} onTestCompose={onTestCompose}/>);
    expect(screen.getByLabelText('Frame 1 dimensions').value).toBe('status');
    expect(screen.getByLabelText('Frame 1 measures').value).toBe('total');

    await user.click(screen.getByRole('button',{name:'Add cube frame'}));
    await user.type(screen.getByLabelText('Frame 2 measures'),'total');
    await user.selectOptions(screen.getByLabelText('Frame 2 inherited filters'),'1');
    fireEvent.change(screen.getByLabelText('Frame 2 explicit filters'),{target:{value:'{}'}});
    await user.click(screen.getByRole('button',{name:'Run composition'}));
    expect(onTestCompose).toHaveBeenCalledWith({
      cubes:[
        {dimensions:{status:true},measures:{total:true},filters:{}},
        {dimensions:{},measures:{total:true},filters:{},inheritFrom:1},
      ],
      sql:'SELECT t1.STATUS, t1.TOTAL\nFROM $CubeSQL1 AS t1',
    });
    expect(await screen.findByText('active')).toBeTruthy();
  });

  test('rejects malformed frame filters before invoking Datly',async()=>{
    const user=userEvent.setup();
    const onTestCompose=vi.fn();
    render(<ReaderAnalyticsDialog isOpen structure={structure} onClose={vi.fn()} onTestCompose={onTestCompose}/>);
    fireEvent.change(screen.getByLabelText('Frame 1 explicit filters'),{target:{value:'[]'}});
    await user.click(screen.getByRole('button',{name:'Run composition'}));
    expect((await screen.findByRole('alert')).textContent).toContain('Every frame filter must be a JSON object');
    expect(onTestCompose).not.toHaveBeenCalled();
  });
});

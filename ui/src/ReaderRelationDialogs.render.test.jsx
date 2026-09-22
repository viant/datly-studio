import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderRelationDialog } from './ReaderRelationDialog.jsx';
import { ReaderRemoveViewDialog } from './ReaderRemoveViewDialog.jsx';

const products={name:'products',namespace:'products',relations:[]};
const node={parentName:'vendor',child:{name:'products',label:'Products'},relation:{on:[{childNamespace:'products',childColumn:'VENDOR_ID',parentNamespace:'vendor',parentColumn:'ID'}]}};
const structure={views:[{name:'vendor'},{name:'products'}],component:{rootView:{name:'reader',namespace:'vendor',relations:[{name:'products',view:products}]}}};

describe('relation authoring dialogs',()=>{
  test('submits the complete compiled key expression as one typed operation',async()=>{
    const user=userEvent.setup();
    const onApply=vi.fn().mockResolvedValue({});
    const onClose=vi.fn();
    render(<ReaderRelationDialog isOpen node={node} structure={structure} onClose={onClose} onApply={onApply}/>);
    expect(screen.getByLabelText('Relation keys').value).toBe('products.VENDOR_ID=vendor.ID');
    await user.click(screen.getByRole('button',{name:'Apply relation'}));
    expect(onApply).toHaveBeenCalledWith({type:'updateRelation',relation:{name:'products',parent:'vendor',on:'products.VENDOR_ID=vendor.ID'}});
    expect(onClose).toHaveBeenCalledOnce();
  });

  test('preserves a rejected relation edit for correction',async()=>{
    const user=userEvent.setup();
    const onApply=vi.fn().mockRejectedValue(new Error('ambiguous key ownership'));
    render(<ReaderRelationDialog isOpen node={node} structure={structure} onClose={vi.fn()} onApply={onApply}/>);
    const keys=screen.getByLabelText('Relation keys');
    await user.clear(keys);await user.type(keys,'products.ACCOUNT_ID=vendor.ACCOUNT_ID');
    await user.click(screen.getByRole('button',{name:'Apply relation'}));
    expect((await screen.findByRole('alert')).textContent).toContain('ambiguous key ownership');
    expect(keys.value).toBe('products.ACCOUNT_ID=vendor.ACCOUNT_ID');
  });

  test('blocks removal while a related view still has descendants',async()=>{
    const child={name:'items',namespace:'items',relations:[]};
    const nested={...structure,views:[{name:'vendor'},{name:'products'},{name:'items'}],component:{rootView:{name:'reader',namespace:'vendor',relations:[{name:'products',view:{...products,relations:[{name:'items',view:child}]}}]}}};
    const onApply=vi.fn();
    render(<ReaderRemoveViewDialog isOpen structure={nested} initialView="products" onClose={vi.fn()} onApply={onApply}/>);
    expect(screen.getByText(/dependent subviews: items/i)).toBeTruthy();
    expect(screen.getByRole('button',{name:'Remove view'}).disabled).toBe(true);
    expect(onApply).not.toHaveBeenCalled();
  });
});

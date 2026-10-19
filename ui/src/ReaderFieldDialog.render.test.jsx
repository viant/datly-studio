import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderFieldDialog } from './ReaderFieldDialog.jsx';

function fieldNode() {
  return {
    name: 'vendor', label: 'Vendor',
    view: { columns: [{ name: 'NAME', source: 'NAME', databaseType: 'TEXT', type: { name: 'string' } }] },
  };
}

describe('ReaderFieldDialog', () => {
  test('sends one normalized Datly column-contract operation', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    const onClose = vi.fn();
    const node=fieldNode();node.column=node.view.columns[0];
    render(<ReaderFieldDialog isOpen node={node} columnContracts={[]} onClose={onClose} onApply={onApply} />);

    expect(screen.queryByRole('textbox', { name: 'Search columns' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Choose another column' })).toBeTruthy();

    await user.selectOptions(screen.getByLabelText('Output visibility'), 'internal');
    await user.selectOptions(screen.getByLabelText('Cube role'), 'dimension');
    await user.type(screen.getByLabelText('Go cast'), 'model.Name');
    await user.type(screen.getByLabelText('Public field name'), 'VendorName');
    await user.click(screen.getByRole('button', { name: 'Save column contract' }));

    expect(onApply).toHaveBeenCalledWith({
      type: 'setColumnContract',
      column: {
        view: 'vendor', column: 'NAME', castType: 'model.Name',
        tags: { groupable: 'true', internal: 'true', format: 'name=VendorName', json: 'VendorName' },
        removeTags: ['groupable', 'internal', 'json', 'format'],
      },
    });
    expect(onClose).toHaveBeenCalledOnce();
  });

  test('validates advanced tag names before invoking Reader Builder', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    render(<ReaderFieldDialog isOpen node={fieldNode()} columnContracts={[]} onClose={vi.fn()} onApply={onApply} />);

    await user.type(screen.getByLabelText('Advanced tag name'), 'bad tag');
    await user.click(screen.getByRole('button', { name: 'Stage tag' }));
    expect((await screen.findByRole('alert')).textContent).toContain('must be an identifier');
    expect(onApply).not.toHaveBeenCalled();
  });

  test('discards staged tags on cancel and reopen', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    const onClose = vi.fn();
    const node=fieldNode();node.column=node.view.columns[0];
    const props={node,columnContracts:[{view:'vendor',column:'NAME',tags:{codec:'CSV'}}],onClose,onApply};
    const view=render(<ReaderFieldDialog isOpen {...props}/>);

    expect(screen.getByText('codec: CSV')).toBeTruthy();
    await user.type(screen.getByLabelText('Advanced tag name'),'trim');
    await user.type(screen.getByLabelText('Advanced tag value'),'true');
    await user.click(screen.getByRole('button',{name:'Stage tag'}));
    expect(screen.getByText('trim: true')).toBeTruthy();
    await user.click(screen.getByRole('button',{name:'Cancel'}));
    expect(onApply).not.toHaveBeenCalled();

    view.rerender(<ReaderFieldDialog isOpen={false} {...props}/>);
    view.rerender(<ReaderFieldDialog isOpen {...props}/>);
    expect(await screen.findByText('codec: CSV')).toBeTruthy();
    expect(screen.queryByText('trim: true')).toBeNull();
  });
  test('declares an aggregate output type when driver discovery has no columns', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    const onClose = vi.fn();
    render(<ReaderFieldDialog isOpen node={{name:'record',view:{columns:[]}}} columnContracts={[]} onApply={onApply} onClose={onClose}/>);
    expect(screen.getByRole('button',{name:'Save column contract'}).disabled).toBe(true);
    await user.type(screen.getByLabelText('SQL output alias'),'total');
    expect(screen.getByRole('button',{name:'Save column contract'}).disabled).toBe(true);
    await user.type(screen.getByLabelText('Go cast'),'int64');
    await user.click(screen.getByRole('button',{name:'Save column contract'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({type:'setColumnContract',column:expect.objectContaining({view:'record',column:'total',castType:'int64'})}));
    expect(onClose).toHaveBeenCalledOnce();
  });

  test('declares a missing aggregate alias while preserving known columns', async () => {
    const user=userEvent.setup(),onApply=vi.fn().mockResolvedValue({});
    const node={name:'sale',view:{columns:[{name:'total',source:'total',type:{name:'int64'}}]}};
    render(<ReaderFieldDialog isOpen node={node} columnContracts={[{view:'sale',column:'total',castType:'int64',tags:{groupable:'false'}}]} onApply={onApply} onClose={vi.fn()}/>);
    await user.click(screen.getByRole('button',{name:'Declare SQL output'}));
    await user.type(screen.getByLabelText('SQL output alias'),'quantity');
    expect(screen.getByRole('button',{name:'Save column contract'}).disabled).toBe(true);
    await user.type(screen.getByLabelText('Go cast'),'int64');
    await user.click(screen.getByRole('button',{name:'Save column contract'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({type:'setColumnContract',column:expect.objectContaining({view:'sale',column:'quantity',castType:'int64'})}));
    expect(node.view.columns.map(column=>column.name)).toEqual(['total']);
  });

  test('keeps a declared alias and type while metadata objects refresh', async () => {
    const user=userEvent.setup(),onApply=vi.fn();
    const props={isOpen:true,columnContracts:[],onApply,onClose:vi.fn()};
    const node={name:'sale',view:{columns:[{name:'total',source:'total',type:{name:'int64'}}]}};
    const view=render(<ReaderFieldDialog {...props} node={node}/>);
    await user.click(screen.getByRole('button',{name:'Declare SQL output'}));
    await user.type(screen.getByLabelText('SQL output alias'),'quantity');
    await user.type(screen.getByLabelText('Go cast'),'int64');
    view.rerender(<ReaderFieldDialog {...props} columnContracts={[]} node={{...node,view:{columns:node.view.columns.map(column=>({...column}))}}}/>);
    expect(screen.getByLabelText('SQL output alias').value).toBe('quantity');
    expect(screen.getByLabelText('Go cast').value).toBe('int64');
  });

});

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
        tags: { groupable: 'true', internal: 'true', format: 'name=VendorName' },
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
});

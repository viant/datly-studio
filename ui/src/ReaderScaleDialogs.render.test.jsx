import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderParameterDialog } from './ReaderParameterDialog.jsx';
import { ReaderPredicateDialog } from './ReaderPredicateDialog.jsx';

function parameterStructure() {
  const request = Array.from({ length: 120 }, (_, index) => ({ parameter: {
    name: `Input${index}`, typeExpr: 'string', source: { kind: 'query', name: `input_${index}` },
  } }));
  return { declarations: [...request, { parameter: { name: 'TenantID', typeExpr: 'int', source: { kind: 'const', name: 'TenantID' }, value: '7' } }] };
}

function predicateStructure() {
  const declarations = Array.from({ length: 240 }, (_, index) => ({
    parameter: { name: `Input${index}`, typeExpr: '[]int', source: { kind: 'query', name: `field_${index}` } },
    predicates: [{ ordinal: 0, predicate: { group: index % 2 ? 1 : 0, name: 'in', args: ['v', `field_${index}`] } }],
  }));
  return {
    declarations,
    component: { rootView: { name: 'reader', namespace: 'records', relations: [] } },
    views: [{ name: 'records' }],
    predicateExpansions: [
      { group: 0, view: 'records', operator: 'AND' },
      { group: 1, view: 'records', operator: 'AND' },
    ],
    predicateCompositions: [{
      view: 'records', occurrence: 0, editable: true, buildKeyword: 'WHERE',
      terms: [{ operator: 'AND', connector: 'AND', groups: [0] }, { operator: 'AND', connector: 'AND', groups: [1] }],
    }],
  };
}

describe('large Reader Builder authoring dialogs', () => {
  test('separates 120 request inputs from trusted constants and preserves zero defaults', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    render(<ReaderParameterDialog isOpen structure={parameterStructure()} onClose={vi.fn()} onApply={onApply} />);

    expect(screen.getByText('120 request inputs')).toBeTruthy();
    expect(screen.getByText('1–50 of 120')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Next input page' }));
    expect(screen.getByText('51–100 of 120')).toBeTruthy();
    await user.type(screen.getByRole('textbox', { name: 'Search inputs' }), 'input_119');
    expect(await screen.findByText('Input119')).toBeTruthy();

    await user.click(screen.getByRole('tab', { name: /Trusted constants/ }));
    expect(screen.getByText('1 constants')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: /TenantID/ }));
    const authoredDefault = screen.getByLabelText('Authored default');
    await user.clear(authoredDefault);
    await user.type(authoredDefault, '0');
    await user.click(screen.getByRole('button', { name: 'Save constant' }));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({
      type: 'updateField',
      field: expect.objectContaining({ existingName: 'TenantID', sourceKind: 'const', value: '0', updateValue: true }),
    }));
  });

  test('pages 240 predicates and edits outer Boolean composition independently', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    render(<ReaderPredicateDialog isOpen structure={predicateStructure()} onClose={vi.fn()} onApply={onApply} />);

    expect(screen.getByText('240 predicates')).toBeTruthy();
    expect(screen.getByText('1–25 of 240')).toBeTruthy();
    expect(screen.getByText('Page 1 of 10')).toBeTruthy();
    expect(screen.getByLabelText('Within-group operator')).toBeTruthy();
    expect(screen.getByLabelText('When input is absent')).toBeTruthy();
    await user.type(screen.getByPlaceholderText('Find parameter, view, source, handler, or column'), 'field_239');
    expect(await screen.findByText('Input239')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Clear predicate search' }));
    await user.click(screen.getByRole('tab', { name: 'Groups' }));

    const composition = screen.getByRole('region', { name: 'records predicate composition 1' });
    await user.selectOptions(within(composition).getByLabelText('Combine operator for stage 2'), 'OR');
    await user.selectOptions(within(composition).getByLabelText('SQL attachment'), 'AND');
    await user.click(within(composition).getByRole('button', { name: 'Save composition' }));
    expect(onApply).toHaveBeenCalledWith({
      type: 'updatePredicateComposition',
      predicateComposition: {
        view: 'records', occurrence: 0, buildKeyword: 'AND',
        terms: [{ operator: 'AND', connector: 'AND', groups: [0] }, { operator: 'OR', connector: 'AND', groups: [1] }],
      },
    });
  });

  test('assigns a Security predicate handler to a component parameter', async()=>{
    const user=userEvent.setup();
    const onApply=vi.fn().mockResolvedValue({});
    const api={listAuthorizationPredicates:vi.fn().mockResolvedValue({items:[{name:'iam.vendor.read',title:'Vendor access',packagePath:'example.com/iam',typeName:'VendorRead',alias:'v',columns:['vendor_id'],linked:true,status:'active'}]})};
    render(<ReaderPredicateDialog api={api} isOpen structure={predicateStructure()} onClose={vi.fn()} onApply={onApply}/>);
    await user.type(screen.getByPlaceholderText('Search parameter'),'Input1');
    await user.selectOptions(await screen.findByLabelText('Security predicate'),'iam.vendor.read');
    await user.click(screen.getByRole('button',{name:'Add predicate'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({type:'addFieldPredicate',predicate:expect.objectContaining({field:'Input1',name:'handler',args:['example.com/iam.VendorRead'],group:99,applyWhenAbsent:true})}));
  });
});

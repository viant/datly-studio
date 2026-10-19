import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderParameterDialog } from './ReaderParameterDialog.jsx';
import { inputKinds } from './parameterKinds.js';
import { ReaderOutputDialog } from './ReaderOutputDialog.jsx';
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
    availablePredicates: ['equal','in','not_in','expr','handler'],
    component: { rootView: { name: 'reader', namespace: 'records', columns: [{name:'ID',source:'c.id'}], relations: [] } },
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

  test('offers built-in input kinds and permits an empty whole-body source', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    render(<ReaderParameterDialog isOpen embedded formOnly structure={parameterStructure()} onClose={vi.fn()} onApply={onApply}/>);
    const kind = screen.getByLabelText('Kind');
    expect([...kind.options].map((option)=>option.value)).toEqual(inputKinds);
    await user.type(screen.getByLabelText('Input name'),'Payload');
    await user.selectOptions(kind,'body');
    await user.click(screen.getByRole('button',{name:'Add input'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({type:'addField',field:expect.objectContaining({name:'Payload',sourceKind:'body',sourceName:''})}));
  });

  test('suggests the native metrics and status output types', async () => {
    const user = userEvent.setup();
    render(<ReaderOutputDialog structure={{declarations:[]}} onClose={vi.fn()} onApply={vi.fn()}/>);
    const source = screen.getByLabelText('Kind source');
    expect([...source.options].map((option)=>option.value)).toEqual(['view','body','status','metrics','derived','summary']);
    await user.selectOptions(source,'metrics');
    expect(screen.getByLabelText('Type expression').value).toBe('github.com/viant/xdatly/response.Metrics');
    await user.selectOptions(source,'status');
    expect(screen.getByLabelText('Type expression').value).toBe('string');
  });

  test('edits output/view as a parameter without navigating to SQL', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn().mockResolvedValue({});
    render(<ReaderOutputDialog structure={{declarations:[{parameter:{name:'Records',typeExpr:'[]*Record',source:{kind:'output',name:'view'}}}]}} initialName="Records" onClose={vi.fn()} onApply={onApply}/>);
    expect(screen.getByLabelText('Kind').value).toBe('output');
    expect(screen.getByLabelText('Kind source').value).toBe('view');
    await user.clear(screen.getByLabelText('Output name'));
    await user.type(screen.getByLabelText('Output name'),'Items');
    await user.click(screen.getByRole('button',{name:'Save output'}));
    expect(onApply).toHaveBeenCalledWith({type:'updateOutputField',field:{existingName:'Records',name:'Items',type:'[]*Record',sourceKind:'output',sourceName:'view'}});
  });

  test('creates a selector-bound input and built-in predicate in one compiled operation', async () => {
    const user=userEvent.setup();const onApply=vi.fn().mockResolvedValue({});
    render(<ReaderParameterDialog isOpen embedded formOnly structure={predicateStructure()} onClose={vi.fn()} onApply={onApply}/>);
    await user.type(screen.getByLabelText('Input name'),'IDs');
    await user.clear(screen.getByLabelText('Type expression'));
    await user.paste('[]int');
    await user.type(screen.getByLabelText('Kind source'),'ids');
    expect([...screen.getByLabelText('Query selector view').options].map((item)=>item.value)).toEqual(['','records']);
    await user.selectOptions(screen.getByLabelText('Predicate'),'in');
    await user.selectOptions(screen.getByLabelText('Filter dimension'),'c.id');
    await user.click(screen.getByRole('button',{name:'Add input'}));
    expect(onApply).toHaveBeenCalledWith({type:'batch',operations:[expect.objectContaining({type:'addField',field:expect.objectContaining({name:'IDs',type:'[]int'})}),expect.objectContaining({type:'addFieldPredicate',predicate:expect.objectContaining({field:'IDs',name:'in',args:['c','id'],view:'records',group:1})})]});
  });

  test('discovers a custom handler without a saved Security entry', async () => {
    const user=userEvent.setup();const onApply=vi.fn().mockResolvedValue({});
    const api={fetchMCPDataSource:vi.fn().mockImplementation(source=>Promise.resolve(source.id==='reader_predicate_types'?[{packagePath:'example.com/filter',typeName:'CustomerScope',description:'Tenant filter'}]:[]))};
    render(<ReaderParameterDialog api={api} isOpen embedded formOnly structure={predicateStructure()} onClose={vi.fn()} onApply={onApply}/>);
    await user.type(screen.getByLabelText('Input name'),'Tenant');
    await user.type(screen.getByLabelText('Kind source'),'tenant');
    await user.selectOptions(screen.getByLabelText('Predicate'),'handler');
    await screen.findByRole('option',{name:'CustomerScope — example.com/filter'});
    await user.type(screen.getByLabelText('Find custom predicate'),'CustomerScope');
    await user.selectOptions(screen.getByLabelText('Custom predicate'),'example.com/filter.CustomerScope');
    await user.click(screen.getByRole('button',{name:'Add input'}));
    expect(onApply.mock.calls[0][0].operations[1]).toEqual(expect.objectContaining({type:'addFieldPredicate',predicate:expect.objectContaining({name:'handler',args:['example.com/filter.CustomerScope'],group:1,applyWhenAbsent:false})}));
  });

  test('keeps expression commas inside one predicate argument', async () => {
    const user=userEvent.setup();const onApply=vi.fn().mockResolvedValue({});
    render(<ReaderPredicateDialog isOpen formOnly structure={predicateStructure()} initialField="Input1" onClose={vi.fn()} onApply={onApply}/>);
    await user.selectOptions(screen.getByLabelText('Predicate'),'expr');
    await user.type(screen.getByLabelText('Predicate arguments'),'COALESCE(c.active, 0) = ?');
    await user.click(screen.getByRole('button',{name:'Add predicate'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({predicate:expect.objectContaining({field:'Input1',name:'expr',args:['COALESCE(c.active, 0) = ?']})}));
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
    const api={fetchMCPDataSource:vi.fn().mockResolvedValue([{name:'iam.vendor.read',title:'Vendor access',packagePath:'example.com/iam',typeName:'VendorRead',alias:'v',columns:['vendor_id'],linked:true,status:'active'}])};
    render(<ReaderPredicateDialog api={api} isOpen structure={predicateStructure()} onClose={vi.fn()} onApply={onApply}/>);
    await user.type(screen.getByPlaceholderText('Search parameter'),'Input1');
    await user.click(screen.getByText('Authorization options'));
    await user.selectOptions(await screen.findByLabelText('Security predicate'),'iam.vendor.read');
    await user.click(screen.getByRole('button',{name:'Add predicate'}));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({type:'addFieldPredicate',predicate:expect.objectContaining({field:'Input1',name:'handler',args:['example.com/iam.VendorRead'],group:99,applyWhenAbsent:true})}));
  });
});

test('uses MCP codec identities and authors required JWT error metadata',async()=>{
 const user=userEvent.setup(),onApply=vi.fn().mockResolvedValue({});
 const api={fetchMCPDataSource:vi.fn().mockResolvedValue([{name:'JwtClaim',inputType:'string',outputType:'*github.com/viant/scy/auth/jwt.Claims',acceptsArguments:false}])};
 render(<ReaderParameterDialog api={api} isOpen embedded formOnly structure={predicateStructure()} onClose={vi.fn()} onApply={onApply}/>);
 await user.type(screen.getByLabelText('Input name'),'JWT');
 await user.click(screen.getByText('Binding options'));
 await screen.findByRole('option',{name:'JwtClaim'});
 await user.selectOptions(screen.getByLabelText('Codec'),'JwtClaim');
 expect(screen.getByLabelText('Kind').value).toBe('header');
 expect(screen.getByLabelText('Kind source').value).toBe('Authorization');
 expect(screen.getByLabelText('Required').value).toBe('true');
 expect(screen.getByLabelText('Type expression').value).toBe('string,*github.com/viant/scy/auth/jwt.Claims');
 expect(screen.getByLabelText('Codec arguments').disabled).toBe(true);
 await user.click(screen.getByText('Error response'));
 expect(screen.getByLabelText('Error code').value).toBe('401');
 await user.clear(screen.getByLabelText('Error message'));await user.type(screen.getByLabelText('Error message'),'Sign in required');
 await user.click(screen.getByRole('button',{name:'Add input'}));
 expect(onApply).toHaveBeenCalledWith(expect.objectContaining({field:expect.objectContaining({type:'string,*github.com/viant/scy/auth/jwt.Claims',required:true,errorStatusCode:401,errorMessage:'Sign in required',codec:{name:'JwtClaim',args:[]}})}));
});

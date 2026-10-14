import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {ReaderEntityScopeDialog} from './ReaderEntityScopeDialog.jsx';

test('binds the current component and mandatory row filter in one atomic operation',async()=>{
 const onApply=vi.fn().mockResolvedValue({}),onClose=vi.fn();
 render(<ReaderEntityScopeDialog isOpen report={{id:'owned-reader'}} root={{name:'records',source:{table:'main.customers'}}} structure={{declarations:[]}} onApply={onApply} onClose={onClose}/>);
 await userEvent.click(screen.getByRole('button',{name:'Bind entity scope'}));
 expect(onApply).toHaveBeenCalledTimes(1);
 const operation=onApply.mock.calls[0][0];
 expect(operation.type).toBe('batch');
 expect(operation.operations[0].field).toMatchObject({type:'*github.com/viant/datly-studio/runtime/accesscontext.Output',sourceKind:'component',sourceName:'GET:/_studio/access/context/owned-reader/customer',required:true});
 expect(operation.operations[1].field).toMatchObject({sourceKind:'param',sourceName:'Auth.Scope.IDs',required:true,codec:{name:'EntityIDs'}});
 expect(operation.operations[2].predicate).toMatchObject({group:99,groupOperator:'AND',applyWhenAbsent:true,args:['customers','id']});
 expect(onClose).toHaveBeenCalled();
});

test('preserves an existing Auth declaration',async()=>{
 const onApply=vi.fn();
 render(<ReaderEntityScopeDialog isOpen report={{id:'reader'}} root={{name:'records'}} structure={{declarations:[{parameter:{name:'Auth'}}]}} onApply={onApply} onClose={()=>{}}/>);
 await userEvent.click(screen.getByRole('button',{name:'Bind entity scope'}));
 expect((await screen.findByRole('alert')).textContent).toContain('already defines');
 expect(onApply).not.toHaveBeenCalled();
});

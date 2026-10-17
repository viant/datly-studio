import React from 'react';
import {describe,expect,test,vi} from 'vitest';
import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {ReaderCubePreview} from './ReaderCubePreview.jsx';
const structure={component:{rootView:{columns:[{name:'country_code',groupable:true},{name:'ChannelV2',groupable:true},{name:'Avails',groupable:false}]}},declarations:[{parameter:{name:'IncludePublisherId',source:{kind:'query'},typeExpr:'[]int'}}]};
describe('Cube preview field selection',()=>{
 test('requires an explicit selection and preserves wire names and typed filters',async()=>{
  const user=userEvent.setup(),onRun=vi.fn();
  render(<ReaderCubePreview structure={structure} onRun={onRun} onClose={()=>{}}/>);
  expect(screen.getByRole('button',{name:'Run cube preview'}).disabled).toBe(true);
  await user.click(screen.getByRole('checkbox',{name:'country_code'}));
  await user.click(screen.getByRole('checkbox',{name:'Avails'}));
  await user.click(screen.getByText('Filters'));
  await user.type(screen.getByLabelText('IncludePublisherId'),'127,1471');
  await user.click(screen.getByRole('button',{name:'Run cube preview'}));
  await waitFor(()=>expect(onRun).toHaveBeenCalledWith({dimensions:{country_code:true,channelV2:false},measures:{avails:true},filters:{includePublisherId:[127,1471]}}));
  await user.type(screen.getByRole('textbox',{name:'Find cube fields'}),'Avails');
  expect(screen.queryByRole('checkbox',{name:'country_code'})).toBeNull();
  expect(screen.getByRole('checkbox',{name:'Avails'}).checked).toBe(true);
 });
 test('rejects invalid numeric filters and blocks a contract without measures',async()=>{
  const user=userEvent.setup(),onRun=vi.fn();
  const view=render(<ReaderCubePreview structure={structure} onRun={onRun} onClose={()=>{}}/>);
  await user.click(screen.getByRole('checkbox',{name:'Avails'}));
  await user.click(screen.getByText('Filters'));
  await user.type(screen.getByLabelText('IncludePublisherId'),'wrong');
  await user.click(screen.getByRole('button',{name:'Run cube preview'}));
  expect(await screen.findByRole('alert')).toBeTruthy();expect(onRun).not.toHaveBeenCalled();
  view.unmount();
  render(<ReaderCubePreview structure={{component:{rootView:{columns:[{name:'Country',groupable:true}]}}}} onRun={onRun} onClose={()=>{}}/>);
  await user.click(screen.getByRole('checkbox',{name:'Country'}));
  expect(screen.getByRole('button',{name:'Run cube preview'}).disabled).toBe(true);
 });
});
test('form inputs and calendar dates reach the native cube filter contract',async()=>{
 const user=userEvent.setup(),onRun=vi.fn();
 render(<ReaderCubePreview structure={{...structure,declarations:[{parameter:{name:'AdvertiserId',source:{kind:'form'},typeExpr:'int'}},{parameter:{name:'From',source:{kind:'form'},typeExpr:'*time.Time'}},{parameter:{name:'Auth',source:{kind:'component'},typeExpr:'*auth.Output'}}]}} onRun={onRun} onClose={()=>{}}/>);
 await user.click(screen.getByRole('checkbox',{name:'Avails'}));await user.click(screen.getByText('Filters'));
 expect(screen.queryByLabelText('Auth')).toBeNull();
 await user.type(screen.getByLabelText('AdvertiserId'),'53');await user.type(screen.getByLabelText('From'),'2026-09-01');
 await user.click(screen.getByRole('button',{name:'Run cube preview'}));
 expect(onRun.mock.calls[0][0].filters).toEqual({advertiserId:53,from:'2026-09-01'});
 await user.clear(screen.getByLabelText('From'));await user.type(screen.getByLabelText('From'),'2026-02-30');await user.click(screen.getByRole('button',{name:'Run cube preview'}));
 expect(onRun).toHaveBeenCalledTimes(1);expect(await screen.findByRole('alert')).toBeTruthy();
});
test('uses declared JSON column names for cube selectors',async()=>{
 const user=userEvent.setup(),onRun=vi.fn();
 render(<ReaderCubePreview structure={{component:{rootView:{columns:[{name:'campaign_id',tag:'json:"campaignId"',groupable:true},{name:'total_spend',tag:'json:"totalSpend"',groupable:false}]}}}} onRun={onRun} onClose={()=>{}}/>);
 await user.click(screen.getByLabelText('campaign_id'));await user.click(screen.getByLabelText('total_spend'));await user.click(screen.getByRole('button',{name:'Run cube preview'}));
 expect(onRun).toHaveBeenCalledWith({dimensions:{campaignId:true},measures:{totalSpend:true},filters:{}});
});

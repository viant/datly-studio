import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderCacheDialog } from './ReaderCacheDialog.jsx';

const structure={availableConnectors:['main'],component:{rootView:{columns:[{name:'ID',source:'ID'}]},settings:{cache:{enabled:true,name:'vendors',ttl:'60s',location:'/cache/vendors',warmup:{indexColumn:'ID',maxCases:4,limit:50,cases:[{set:[{name:'Region',values:['west','east']}]}]}}}}};
const report={id:'vendor'};
const version={versionNo:2,sourceRevision:4};

describe('ReaderCacheDialog',()=>{
  test('restores durable history and blocks an invalid case product',async()=>{
    const user=userEvent.setup();
    const run={runId:'run-1',status:'completed',versionNo:2,sourceRevision:4,plannedCases:2,completedCases:2,maxCases:4,rowLimit:50,entries:2,duration:1_000_000,target:{connectorName:'main',cacheName:'vendors',cacheProvider:'afs',indexColumn:'ID'},requestedAt:'2026-09-20T12:00:00Z',requestedBy:'alice'};
    const api={listWarmupRuns:vi.fn().mockResolvedValue({items:[run]})};
    const onApply=vi.fn();
    render(<ReaderCacheDialog api={api} report={report} version={version} canWarmup isOpen structure={structure} onClose={vi.fn()} onApply={onApply} onWarmup={vi.fn()}/>);

    expect(await screen.findByText('run-1')).toBeTruthy();
    expect(screen.getByText('2 / 2 cases')).toBeTruthy();
    const cases=screen.getByLabelText('Case dimensions');
    await user.clear(cases);await user.type(cases,'Region=west,east\nregion=north,south');
    expect(screen.getByText(/Case dimension region is duplicated/i)).toBeTruthy();
    expect(screen.getByRole('button',{name:'Save warmup plan'}).disabled).toBe(true);
    expect(onApply).not.toHaveBeenCalled();
  });

  test('surfaces a failed durable warmup without replacing prior history',async()=>{
    const user=userEvent.setup();
    const run={runId:'run-1',status:'completed',versionNo:2,sourceRevision:4,plannedCases:2,completedCases:2,maxCases:4,rowLimit:50,entries:2,duration:1_000_000,requestedAt:'2026-09-20T12:00:00Z',requestedBy:'alice'};
    const api={listWarmupRuns:vi.fn().mockResolvedValue({items:[run]})};
    const onWarmup=vi.fn().mockRejectedValue(new Error('warmup worker unavailable'));
    render(<ReaderCacheDialog api={api} report={report} version={version} canWarmup isOpen structure={structure} onClose={vi.fn()} onApply={vi.fn()} onWarmup={onWarmup}/>);
    await screen.findByText('run-1');
    await user.click(screen.getByRole('button',{name:'Run again'}));
    expect((await screen.findByRole('alert')).textContent).toContain('warmup worker unavailable');
    expect(screen.getByText('run-1')).toBeTruthy();
  });
});

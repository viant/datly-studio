import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderPublicationDialog, releaseChangeSummary } from './ReaderPublicationDialog.jsx';

describe('publication contract comparison', () => {
  test('summarizes consumer-facing serving to proposed changes', () => {
    const serving={structure:{views:[{name:'vendor',sql:'SELECT ID FROM VENDOR'}],declarations:[{parameter:{name:'ID'}}],component:{rootView:{name:'vendor',columns:[{name:'ID'}]},routes:[{path:'/vendors'}],settings:{}},columnContracts:[]}};
    const proposed={structure:{views:[{name:'vendor',sql:'SELECT ID, NAME FROM VENDOR'}],declarations:[{parameter:{name:'ID'}},{parameter:{name:'Name'}}],component:{rootView:{name:'vendor',columns:[{name:'ID'},{name:'NAME'}]},routes:[{path:'/vendors',mcp:[{name:'alice.vendor.read'}]}],settings:{}},columnContracts:[]}};
    expect(releaseChangeSummary(serving,proposed,{resourceManifest:{files:[]}},{resourceManifest:{files:['guide.md']}})).toEqual([
      'SQL / views','Inputs','Output contract','HTTP / MCP','Resources / skills',
    ]);
  });

  test('does not claim equality for a mutable same-version serving snapshot', async () => {
    const api={
      getPublication:vi.fn().mockResolvedValue({activeVersionNo:2,desiredVersionNo:2,desiredGeneration:4,activeGeneration:4,status:'active',specHash:'serving-hash'}),
      listVersions:vi.fn().mockResolvedValue({items:[{versionNo:2,specHash:'draft-hash',compileStatus:'valid'}]}),
      listPublicationEvents:vi.fn().mockResolvedValue({items:[]}),
      getRuntimeStatus:vi.fn().mockResolvedValue({status:'active',activeGeneration:4,reportCount:1}),
      inspectVersion:vi.fn(),
    };
    render(<ReaderPublicationDialog isOpen api={api} report={{id:'vendor'}} version={{versionNo:2,sourceRevision:7,specHash:'draft-hash',compileStatus:'valid'}} inspection={{structure:{}}} onClose={vi.fn()} onPublish={vi.fn()} onUnpublish={vi.fn()} onRollback={vi.fn()}/>);
    expect(await screen.findByText('Detailed comparison unavailable')).toBeTruthy();
    expect(screen.getByText(/will not claim the mutable draft matches/i)).toBeTruthy();
    expect(api.inspectVersion).not.toHaveBeenCalled();
    expect(screen.queryByText(/match the serving version/i)).toBeNull();
  });

  test('shows publication failure while preserving the currently serving generation', async () => {
    const user = userEvent.setup();
    const onPublish = vi.fn().mockRejectedValue(new Error('runtime reload failed; serving generation was preserved'));
    const api = {
      getPublication: vi.fn().mockResolvedValue({
        activeVersionNo: 1,
        desiredVersionNo: 1,
        activeGeneration: 9,
        desiredGeneration: 9,
        status: 'active',
        specHash: 'serving-hash',
      }),
      listVersions: vi.fn().mockResolvedValue({ items: [
        { versionNo: 1, sourceRevision: 4, specHash: 'serving-hash', compileStatus: 'valid' },
        { versionNo: 2, sourceRevision: 5, specHash: 'draft-hash', compileStatus: 'valid' },
      ] }),
      listPublicationEvents: vi.fn().mockResolvedValue({ items: [] }),
      getRuntimeStatus: vi.fn().mockResolvedValue({ status: 'active', activeGeneration: 9, reportCount: 1 }),
      inspectVersion: vi.fn().mockResolvedValue({ structure: {} }),
    };

    render(<ReaderPublicationDialog
      isOpen
      api={api}
      report={{ id: 'vendor' }}
      version={{ versionNo: 2, sourceRevision: 5, specHash: 'draft-hash', compileStatus: 'valid' }}
      inspection={{ structure: {} }}
      onClose={vi.fn()}
      onPublish={onPublish}
      onUnpublish={vi.fn()}
      onRollback={vi.fn()}
    />);

    expect((await screen.findAllByText('v1 · generation 9')).length).toBeGreaterThan(0);
    expect(screen.getByText('generation 9 · 1 readers')).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Publish validated revision' }));

    expect(onPublish).toHaveBeenCalledWith('');
    expect((await screen.findByRole('alert')).textContent).toContain('runtime reload failed; serving generation was preserved');
    expect(screen.getByText('Publication did not activate')).toBeTruthy();
    expect(screen.getAllByText('v1 · generation 9').length).toBeGreaterThan(0);
    expect(screen.getByText('generation 9 · 1 readers')).toBeTruthy();
    expect(screen.queryByText('Runtime generation active')).toBeNull();
    expect(api.getRuntimeStatus).toHaveBeenCalledOnce();
  });

  test('distinguishes a publication record from a stopped live host', async () => {
    const api = {
      getPublication: vi.fn().mockResolvedValue(null),
      listVersions: vi.fn().mockResolvedValue({ items: [] }),
      listPublicationEvents: vi.fn().mockResolvedValue({ items: [] }),
      getRuntimeStatus: vi.fn().mockResolvedValue({ status: 'active', activeGeneration: 9, reportCount: 1, host: { status: 'unavailable' } }),
    };
    render(<ReaderPublicationDialog isOpen api={api} report={{id:'vendor'}} version={{versionNo:2,sourceRevision:5,compileStatus:'valid'}} inspection={{structure:{}}} onClose={vi.fn()} onPublish={vi.fn()} onUnpublish={vi.fn()} onRollback={vi.fn()}/>);
    expect(await screen.findByText('Generation record')).toBeTruthy();
    expect(screen.getByText('Live runtime check failed')).toBeTruthy();
    expect(screen.getByText(/Publishing reloads their routes and tools; it does not start the host process/)).toBeTruthy();
  });

  test('unpublish pins the generation shown when the dialog opened', async () => {
    const user = userEvent.setup();
    const onUnpublish = vi.fn().mockResolvedValue({status:'unpublished'});
    const api = {
      getPublication: vi.fn().mockResolvedValue({activeVersionNo:1,activeGeneration:9,desiredGeneration:9,status:'active'}),
      listVersions: vi.fn().mockResolvedValue({items:[]}),
      listPublicationEvents: vi.fn().mockResolvedValue({items:[]}),
      getRuntimeStatus: vi.fn().mockResolvedValue({status:'active',activeGeneration:9,reportCount:1}),
    };
    render(<ReaderPublicationDialog isOpen api={api} report={{id:'vendor'}} version={{versionNo:1,sourceRevision:4,compileStatus:'valid'}} inspection={{structure:{}}} onClose={vi.fn()} onPublish={vi.fn()} onUnpublish={onUnpublish} onRollback={vi.fn()}/>);
    await screen.findByText('v1 · generation 9');
    await user.click(screen.getByRole('button',{name:'Review unpublish'}));
    await user.type(screen.getByPlaceholderText('Describe this release operation'),'retire reader');
    await user.click(screen.getByRole('button',{name:'Confirm unpublish'}));
    expect(onUnpublish).toHaveBeenCalledWith(9,'retire reader');
  });
});

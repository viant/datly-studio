import React from 'react';
import {test,expect,vi} from 'vitest';
import {render,screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {ImportComponentButton,DownloadComponentButton} from './ComponentTransfer.jsx';

test('icon-only import creates a draft and opens the imported version',async()=>{
 const user=userEvent.setup();const onImported=vi.fn();
 const api={listReports:vi.fn().mockResolvedValue({items:[{id:'reader',title:'Reader'}]}),loadDQL:vi.fn().mockResolvedValue({version:{versionNo:3}})};
 render(<ImportComponentButton api={api} onImported={onImported}/>);
 const button=screen.getByRole('button',{name:'Import component'});expect(button.textContent).toBe('');await user.click(button);
 await screen.findByRole('option',{name:'Reader'});await user.selectOptions(screen.getByLabelText('Component'),'reader');
 const file=new File(['SELECT 1'],'reader.dql',{type:'text/plain'});file.text=async()=> 'SELECT 1';
 await user.upload(screen.getByLabelText('DQL or archive'),file);
 await user.click(screen.getByRole('button',{name:'Import as new draft'}));
 expect(api.loadDQL).toHaveBeenCalledWith('reader',{dql:'SELECT 1'});
 expect(onImported).toHaveBeenCalledWith(expect.objectContaining({id:'reader',versionNo:3}));
});

test('download errors are visible and selected version is preserved',async()=>{
 const user=userEvent.setup();const api={downloadComponent:vi.fn().mockRejectedValue(new Error('DQL permission required'))};
 render(<DownloadComponentButton api={api} report={{id:'reader',title:'Reader'}} versionNo={7}/>);
 await user.click(screen.getByRole('button',{name:'Download component v7'}));
 expect(api.downloadComponent).toHaveBeenCalledWith('reader',7);
 expect((await screen.findByRole('alert')).textContent).toContain('DQL permission required');
});

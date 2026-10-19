import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { ModuleLoadBoundary } from './ModuleLoadBoundary.jsx';
test('failed module keeps shell and offers explicit recovery',()=>{
 const log=vi.spyOn(console,'error').mockImplementation(()=>{});
 const back=vi.fn();
 function Failed(){throw new TypeError('Failed to fetch dynamically imported module');}
 render(<><nav>Studio navigation</nav><ModuleLoadBoundary onBack={back}><Failed/></ModuleLoadBoundary></>);
 expect(screen.getByText('Studio navigation')).toBeTruthy();
 expect(screen.getByRole('alert').textContent).toContain('Save any work');
 expect(screen.getByRole('button',{name:'Reload application'})).toBeTruthy();
 fireEvent.click(screen.getByRole('button',{name:'Back to components'}));expect(back).toHaveBeenCalledOnce();
 log.mockRestore();
});

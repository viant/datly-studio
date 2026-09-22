import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderConflictDialog } from './ReaderConflictDialog.jsx';

describe('ReaderConflictDialog', () => {
  test('preserves the draft decision and explains reload impact', async () => {
    const user = userEvent.setup();
    const onReview = vi.fn();
    const onReload = vi.fn();
    render(<ReaderConflictDialog
      conflict={{ message: 'source revision 4 is stale; current revision is 5' }}
      reloading={false}
      onReview={onReview}
      onReload={onReload}
    />);

    expect(screen.getByRole('dialog', { name: 'Reader changed elsewhere' })).toBeTruthy();
    expect(screen.getByRole('alert').textContent).toContain('source revision 4 is stale');
    expect(screen.getByText(/discards its unsaved form values/i)).toBeTruthy();
    expect(screen.getByText(/Published runtime state is unchanged/i)).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Review my draft' }));
    expect(onReview).toHaveBeenCalledOnce();
    expect(onReload).not.toHaveBeenCalled();
  });

  test('locks review and exposes progress while the latest revision reloads', () => {
    render(<ReaderConflictDialog
      conflict={{ message: 'conflict' }}
      reloading
      onReview={vi.fn()}
      onReload={vi.fn()}
    />);

    expect(screen.getByRole('button', { name: 'Review my draft' }).disabled).toBe(true);
    expect(screen.getByRole('button', { name: /Reload latest/ }).disabled).toBe(true);
  });
});

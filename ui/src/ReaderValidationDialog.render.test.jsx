import React from 'react';
import { describe, expect, test, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReaderValidationDialog } from './ReaderValidationDialog.jsx';

const version = {
  versionNo: 3,
  sourceRevision: 12,
  compileStatus: 'pending',
};

describe('ReaderValidationDialog', () => {
  test('renders validation diagnostics and keeps the draft available for correction', async () => {
    const user = userEvent.setup();
    const onOpenSource = vi.fn();
    const onValidate = vi.fn().mockResolvedValue({
      valid: false,
      diagnostics: [
        { severity: 'error', code: 'DQL001', line: 8, column: 14, message: 'Unknown column NAME', hint: 'Select a column exposed by the source view.' },
        { severity: 'warn', code: 'DQL002', message: 'The route has no description' },
      ],
    });

    render(<ReaderValidationDialog
      isOpen
      version={version}
      canUseDQL
      onClose={vi.fn()}
      onValidate={onValidate}
      onOpenSource={onOpenSource}
    />);

    await user.click(screen.getByRole('button', { name: 'Validate revision' }));

    expect(onValidate).toHaveBeenCalledOnce();
    expect((await screen.findByRole('status')).textContent).toContain('current runtime generation is unchanged');
    expect(screen.getByText('2 diagnostics')).toBeTruthy();
    expect(screen.getByText('DQL001')).toBeTruthy();
    expect(screen.getByText('Unknown column NAME')).toBeTruthy();
    expect(screen.getByText('Line 8, column 14')).toBeTruthy();
    expect(screen.getByText('Select a column exposed by the source view.')).toBeTruthy();
    expect(screen.getByText('DQL002')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Open component source' })).toBeTruthy();

    await user.click(screen.getByRole('button', { name: 'Open component source' }));
    expect(onOpenSource).toHaveBeenCalledOnce();
  });

  test('surfaces a failed validation request and re-enables retry', async () => {
    const user = userEvent.setup();
    const onValidate = vi.fn().mockRejectedValue(new Error('validation service unavailable'));
    render(<ReaderValidationDialog
      isOpen
      version={version}
      canUseDQL={false}
      onClose={vi.fn()}
      onValidate={onValidate}
    />);

    await user.click(screen.getByRole('button', { name: 'Validate revision' }));

    expect((await screen.findByRole('alert')).textContent).toContain('validation service unavailable');
    expect(screen.getByRole('button', { name: 'Validate revision' }).disabled).toBe(false);
    expect(screen.queryByRole('button', { name: 'Open component source' })).toBeNull();
  });
});

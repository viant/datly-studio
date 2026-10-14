import React from 'react';
import { test, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SecurityWorkspace } from './SecurityWorkspace.jsx';

test('lists complete linked Go identities even before a security entry is created', async () => {
  const types = [
    {alias: 'one', packagePath: 'example.com/first/predicates', typeName: 'CustomerScope'},
    {alias: 'two', packagePath: 'example.com/second/predicates', typeName: 'CustomerScope'},
  ];
  render(<SecurityWorkspace api={{
    listAuthorizationPredicates: vi.fn().mockResolvedValue({items: []}),
    listAuthorizationPredicateTypes: vi.fn().mockResolvedValue(types),
  }}/>);
  const table = await screen.findByRole('table', {name: 'Linked Go predicates'});
  expect(within(table).getByText(types[0].packagePath)).toBeTruthy();
  expect(within(table).getByText(types[1].packagePath)).toBeTruthy();
  expect(within(table).getAllByText('CustomerScope')).toHaveLength(2);
  await userEvent.click(screen.getByRole('button', {name: 'New predicate'}));
  const chooser = screen.getByLabelText('Linked handler');
  expect(within(chooser).getByRole('option', {name: 'example.com/first/predicates · CustomerScope'})).toBeTruthy();
  expect(within(chooser).getByRole('option', {name: 'example.com/second/predicates · CustomerScope'})).toBeTruthy();
  await userEvent.click(screen.getByRole('button', {name: 'Cancel'}));
  await userEvent.type(screen.getByLabelText('Search authorization predicates'), 'example.com/first');
  await userEvent.click(screen.getByRole('button', {name: 'Search', exact: true}));
  const filtered = await screen.findByRole('table', {name: 'Linked Go predicates'});
  expect(within(filtered).getByText(types[0].packagePath)).toBeTruthy();
  expect(within(filtered).queryByText(types[1].packagePath)).toBeNull();
});

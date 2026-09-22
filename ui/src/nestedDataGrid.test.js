import assert from 'node:assert/strict';
import test from 'node:test';
import { formatCell, nestedFields, previewCollections, rowIdentity, scalarColumns } from './nestedDataGrid.js';

test('nested Datly results retain scalar columns and typed subview collections', () => {
  const data = { Vendors: [
    { Id: 1, VendorName: 'One', Domains: [{ Id: 10, VendorId: 1, Domain: 'one.example' }] },
    { Id: 2, VendorName: 'Two', Domains: [] },
  ] };
  const [collection] = previewCollections(data);
  assert.equal(collection.name, 'Vendors');
  assert.deepEqual(scalarColumns(collection.rows), ['Id', 'VendorName']);
  assert.deepEqual(nestedFields(collection.rows[0]).map(({ name, count }) => ({ name, count })), [{ name: 'Domains', count: 1 }]);
  assert.equal(rowIdentity(collection.rows[0], 0), 'Id:1');
});

test('top-level arrays and scalar-only rows remain tabular', () => {
  const [collection] = previewCollections([{ ID: 7, Active: true, Note: null }]);
  assert.equal(collection.name, 'Rows');
  assert.deepEqual(scalarColumns(collection.rows), ['ID', 'Active', 'Note']);
  assert.deepEqual(nestedFields(collection.rows[0]), []);
});

test('actual Datly subview arrays never become scalar object cells', () => {
  const rows = [{ Id: 1, Name: 'Vendor 1', Products: [{ Id: 1, Name: 'Product 1' }] }];
  assert.deepEqual(scalarColumns(rows), ['Id', 'Name']);
  assert.deepEqual(nestedFields(rows[0]).map((field) => field.name), ['Products']);
  assert.equal(formatCell(rows[0].Products), '—');
});

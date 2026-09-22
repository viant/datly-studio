export function isStructured(value) {
  return value !== null && typeof value === 'object';
}

export function previewCollections(data) {
  if (Array.isArray(data)) return [{ name: 'Rows', rows: data }];
  if (!isStructured(data)) return [];
  return Object.entries(data)
    .filter(([, value]) => Array.isArray(value))
    .map(([name, rows]) => ({ name, rows }));
}

export function scalarColumns(rows) {
  const columns = [];
  const seen = new Set();
  const nested = new Set();
  for (const row of rows ?? []) {
    for (const field of nestedFields(row)) nested.add(field.name);
  }
  for (const row of rows ?? []) {
    if (!isStructured(row) || Array.isArray(row)) continue;
    for (const [name, value] of Object.entries(row)) {
      if (nested.has(name) || isStructured(value) || seen.has(name)) continue;
      seen.add(name);
      columns.push(name);
    }
  }
  return columns;
}

export function nestedFields(row) {
  if (!isStructured(row) || Array.isArray(row)) return [];
  return Object.entries(row)
    .filter(([, value]) => isStructured(value))
    .map(([name, value]) => ({ name, value, count: Array.isArray(value) ? value.length : 1 }));
}

export function rowIdentity(row, index) {
  if (isStructured(row) && !Array.isArray(row)) {
    for (const name of ['Id', 'ID', 'id', 'Name', 'name']) {
      if (row[name] !== undefined && row[name] !== null) return `${name}:${row[name]}`;
    }
  }
  return `row:${index}`;
}

export function formatCell(value) {
  if (value === null || value === undefined) return '—';
  if (isStructured(value)) return '—';
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  return String(value);
}

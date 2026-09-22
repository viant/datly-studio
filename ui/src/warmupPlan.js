export function parseWarmupCases(value) {
  return String(value || '').split(/\r?\n/).map((line) => line.trim()).filter(Boolean).map((line) => {
    const index = line.indexOf('=');
    return index > 0
      ? { name: line.slice(0, index).trim(), values: line.slice(index + 1).split(',').map((item) => item.trim()).filter(Boolean), valid: true }
      : { name: line, values: [], valid: false };
  });
}

export function validateWarmupCases(cases) {
  const seen = new Set();
  for (const item of cases) {
    if (!item.valid || !item.name) return ['Every case dimension must use Name=value1,value2.'];
    if (seen.has(item.name.toLowerCase())) return [`Case dimension ${item.name} is duplicated.`];
    seen.add(item.name.toLowerCase());
    if (item.values.length === 0) return [`Case dimension ${item.name} needs at least one value.`];
  }
  return [];
}

export function warmupCaseCount(cases) {
  if (validateWarmupCases(cases).length) return 0;
  return cases.reduce((count, item) => count * item.values.length, 1);
}

export function warmupCaseExpression(cases) {
  return cases.length ? cases.map((item) => `${item.values.length} ${item.name}`).join(' × ') : '1 default case';
}

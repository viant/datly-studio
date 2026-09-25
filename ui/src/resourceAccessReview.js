function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.keys(value).sort().map(key => [key, canonical(value[key])]));
  }
  return value;
}

// A structural draft diff only. The server alone computes effective access.
export function changedPolicies(current, draft) {
  const before = current?.policies || {};
  const after = draft?.policies || {};
  return [...new Set([...Object.keys(before), ...Object.keys(after)])].sort().filter(action =>
    JSON.stringify(canonical(before[action])) !== JSON.stringify(canonical(after[action]))
  ).map(action => ({ action, before: before[action], after: after[action] }));
}

function describeRule(rule) {
  if (!rule) return 'No rule';
  switch (rule.kind) {
    case 'subject': return `Principal ${rule.value || 'unset'}`;
    case 'role': return `Role ${rule.value || 'unset'}`;
    case 'exposure': return `Feature exposure ${rule.value || 'unset'}`;
    case 'entity': return `Allowed entity ${rule.entity?.type || 'unset'}: ${rule.entity?.id || 'unset'}`;
    case 'all': return `Match all: ${(rule.rules || []).map(describeRule).join('; ')}`;
    case 'any': return `Match any: ${(rule.rules || []).map(describeRule).join('; ')}`;
    default: return 'Unrecognized rule';
  }
}

export function describePolicy(policy) {
  if (!policy) return 'Denied · no policy';
  if (policy.mode === 'public') return 'Public consumption';
  if (policy.mode !== 'protected') return 'Invalid mode';
  return `Protected · ${describeRule(policy.rule)}${policy.entityType ? ` · required ${policy.entityType} scope` : ''}`;
}

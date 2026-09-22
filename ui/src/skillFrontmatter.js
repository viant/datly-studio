export function skillToolNames(content) {
  const match = String(content ?? '').match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) return [];
  const portable = match[1].split(/\r?\n/).find((line) => /^allowed-tools:\s*/.test(line));
  if (!portable) return [];
  const raw = portable.replace(/^allowed-tools:\s*/, '').trim().replace(/^['"]|['"]$/g, '');
  return raw.split(/\s+/).filter(Boolean);
}

export function withSkillTools(content, tools) {
  const source = String(content ?? '');
  const frontmatter = source.match(/^(---\r?\n)([\s\S]*?)(\r?\n---[\s\S]*)$/);
  if (!frontmatter) return source;
  const lines = frontmatter[2].split(/\r?\n/);
  const index = lines.findIndex((line) => /^allowed-tools:\s*/.test(line));
  const portable = `allowed-tools: ${JSON.stringify(tools.join(' '))}`;
  if (index < 0) lines.push(portable); else lines.splice(index, 1, portable);
  return `${frontmatter[1]}${lines.join('\n')}${frontmatter[3]}`;
}

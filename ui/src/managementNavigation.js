// Only existing screens and host-registered extensions can be deep linked.
const sections = ['overview', 'connectors', 'schema', 'namespaces', 'security', 'runtime', 'skills', 'components'];
export function managementSection(search = '', extensions = []) {
  const section = new URLSearchParams(search).get('section');
  return sections.includes(section) || extensions.some((item) => item.id === section) ? section : 'overview';
}

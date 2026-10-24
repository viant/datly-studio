import nativeNavigation from './nativeNavigation.json' with {type:'json'};
// Only existing screens and host-registered extensions can be deep linked.
const sections = [...nativeNavigation.menus.map(item => item.id), 'releases'];
export function managementSection(search = '', extensions = []) {
  const section = new URLSearchParams(search).get('section');
  return sections.includes(section) || extensions.some((item) => item.id === section) ? section : 'overview';
}

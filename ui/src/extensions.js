const reservedSections = new Set(['overview', 'connectors', 'schema', 'namespaces', 'security', 'runtime', 'skills', 'reports', 'builder']);

function assertExtension(extension) {
  if (!extension || typeof extension !== 'object') throw new TypeError('Studio extension must be an object');
  if (!/^[a-z][a-z0-9.-]*$/.test(extension.id || '')) throw new TypeError('Studio extension id must be canonical');
  if (reservedSections.has(extension.id)) throw new TypeError(`Studio extension id ${extension.id} is reserved by the shell`);
  if (!String(extension.label || '').trim()) throw new TypeError(`Studio extension ${extension.id} requires a label`);
  if (typeof extension.render !== 'function') throw new TypeError(`Studio extension ${extension.id} requires render(context)`);
  return Object.freeze({ icon: 'widget', order: 500, ...extension });
}

export function defineStudioExtension(extension) {
  return assertExtension(extension);
}

export function createStudioSDK(options = {}) {
  const extensions = [];
  return {
    register(extension) {
      const value = assertExtension(extension);
      if (extensions.some((item) => item.id === value.id)) throw new Error(`Duplicate Studio extension ${value.id}`);
      extensions.push(value);
      extensions.sort((left, right) => left.order - right.order || left.label.localeCompare(right.label));
      return this;
    },
    extensions() { return [...extensions]; },
    config: Object.freeze({ ...options }),
  };
}

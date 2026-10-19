// Canonical connector rules, shared by the embedded UI and published Forge actions.
export default function connectorBehavior() {
  const drivers = ['sqlite', 'mysql', 'postgres', 'bigquery', 'oracle'];
  function yamlPatch(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Connector YAML must be an object.');
    const fields = Object.keys(value);
    if (!fields.length || fields.some(field => !['Driver', 'DSN', 'Secret'].includes(field))) throw new Error('Connector YAML supports Driver, DSN, and Secret only.');
    const patch = {};
    if (Object.hasOwn(value, 'Driver')) {
      if (!drivers.includes(value.Driver)) throw new Error('Choose a supported connector Driver.');
      patch.driver = value.Driver;
    }
    if (Object.hasOwn(value, 'DSN')) {
      if (typeof value.DSN !== 'string') throw new Error('DSN must be a string.');
      patch.dsnTemplate = value.DSN;
    }
    if (Object.hasOwn(value, 'Secret')) {
      const secret = value.Secret;
      if (secret === null) Object.assign(patch, {secretURL: '', secretKey: '', clearSecret: true});
      else {
        if (!secret || typeof secret !== 'object' || Array.isArray(secret) || Object.keys(secret).some(field => !['URL', 'Key'].includes(field))) throw new Error('Secret must contain URL and optional Key.');
        if (typeof secret.URL !== 'string' || !secret.URL.trim() || (Object.hasOwn(secret, 'Key') && typeof secret.Key !== 'string')) throw new Error('Secret requires a URL string and an optional Key string.');
        Object.assign(patch, {secretURL: secret.URL, secretKey: secret.Key || '', clearSecret: false});
      }
    }
    return patch;
  }
  function saveRequest(form) {
    const name = (form.name || '').trim();
    if (!/^[a-z][a-z0-9_-]{0,127}$/i.test(name)) throw new Error('Use a connector name beginning with a letter; letters, numbers, underscores, and dashes are allowed.');
    const url = (form.secretURL || '').trim(), key = (form.secretKey || '').trim();
    if (!form.clearSecret && key && !url) throw new Error('Enter the secret URL together with its key reference.');
    let options;
    try { options = JSON.parse(form.options || '{}'); } catch { throw new Error('Options must be valid JSON.'); }
    const input = {driver: form.driver, description: (form.description || '').trim(), options};
    if (!form.editing || (form.dsnTemplate || '').trim()) input.dsnTemplate = (form.dsnTemplate || '').trim();
    if (!form.editing || url || form.clearSecret) input.secretRef = form.clearSecret ? '' : url + (key ? '|' + key : '');
    return form.editing ? {operation: 'update', parameters: {name, input: {...input, etag: form.etag}}} : {operation: 'create', parameters: {name, ...input}};
  }
  return {yamlPatch, saveRequest};
}

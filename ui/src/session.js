export function sessionURL(config, path) { return new URL(path, `${config.apiBaseURL}/`).toString(); }

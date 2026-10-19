// Datly's native output uses byte arrays; JSON SDK transports may use base64.
export function downloadBytes(archive) {
  if (Array.isArray(archive)) {
    if (!archive.every(value => Number.isInteger(value) && value >= 0 && value <= 255)) throw new Error('The downloaded archive contains invalid bytes.');
    return Uint8Array.from(archive);
  }
  if (typeof archive === 'string') {
    try { return Uint8Array.from(atob(archive), character => character.charCodeAt(0)); }
    catch { throw new Error('The downloaded archive is not valid base64.'); }
  }
  throw new Error('The server did not return a downloadable archive.');
}

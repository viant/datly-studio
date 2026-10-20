import {downloadBytes} from './downloadBytes.js';

// Browser capability supplied to metadata windows. The producer returns the
// bytes and filename; windows cannot choose a network or filesystem destination.
export function downloadFile(result) {
 const bytes=downloadBytes(result.archive);
 const url=URL.createObjectURL(new Blob([bytes],{type:result.mediaType||'application/octet-stream'}));
 const link=document.createElement('a');link.href=url;link.download=result.filename||'download';
 document.body.appendChild(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);
}

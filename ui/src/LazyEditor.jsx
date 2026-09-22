import React, { lazy, Suspense } from 'react';

const ForgeEditor = lazy(() => import('forge/editor'));

export function LazyEditor(props) {
  return <Suspense fallback={<div className="studio-editor-loading" role="status">Loading source editor…</div>}><ForgeEditor {...props}/></Suspense>;
}

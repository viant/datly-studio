import React, {lazy, Suspense} from 'react';
import {Spinner} from '@blueprintjs/core';

const Runtime = lazy(() => import('./ManagementWindowRuntime.jsx'));

// Importing the Studio shell does not initialize the Forge window runtime.
// The same canonical renderer is loaded when a management screen is opened.
export function ManagementWindow(props) {
  return <Suspense fallback={<Spinner aria-label="Loading window" size={24}/>}><Runtime {...props}/></Suspense>;
}

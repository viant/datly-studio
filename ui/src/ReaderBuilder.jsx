import React,{useMemo} from 'react';
import {ReaderBuilder as SharedReaderBuilder} from 'forge/reader-workbench';
import {readerWorkbenchAPI} from './readerWorkbenchBridge.js';
export function ReaderBuilder({api,...props}) {
 const adapted=useMemo(()=>readerWorkbenchAPI(api),[api]);
 return <SharedReaderBuilder {...props} api={adapted} defaults={{routePrefix:'/v1/studio/readers/',...props.defaults}}/>;
}

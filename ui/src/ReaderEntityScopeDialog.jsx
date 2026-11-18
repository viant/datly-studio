import React from 'react';
import {ReaderEntityScopeDialog as SharedReaderEntityScopeDialog} from 'forge/reader-workbench';
import {readerWorkbenchDefaults} from './readerWorkbenchDefaults.js';

export function ReaderEntityScopeDialog({binding=readerWorkbenchDefaults.rowAccessScope,...props}) {
 return <SharedReaderEntityScopeDialog {...props} binding={binding}/>;
}

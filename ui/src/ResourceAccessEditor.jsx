import React from 'react';
import {ResourceAccessEditor as SharedResourceAccessEditor} from 'forge/resource-policy-editor';

export function ResourceAccessEditor(props) {
 return <SharedResourceAccessEditor publicActionNames={['discover','describe','execute','retrieve']} relatedActions={['discover','describe','execute']} applyGroupLabel="Apply to exposure and execution" {...props}/>;
}

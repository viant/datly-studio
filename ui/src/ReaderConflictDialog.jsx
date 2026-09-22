import React from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter } from '@blueprintjs/core';

export function ReaderConflictDialog({ conflict, reloading, onReview, onReload }) {
  return <Dialog className="studio-connector-dialog studio-conflict-dialog" isOpen={Boolean(conflict)} onClose={onReview} title="Reader changed elsewhere" icon="warning-sign" canOutsideClickClose={!reloading} canEscapeKeyClose={!reloading}>
    <DialogBody>
      <p className="studio-dialog-lead">This editor started from an older source revision. Studio left your draft open so you can review or copy it before deciding what to do.</p>
      <Callout intent="warning" title="Latest draft was not overwritten" role="alert">{conflict?.message || 'The reader source revision no longer matches.'}</Callout>
      <p className="studio-conflict-impact"><strong>Reload latest</strong> closes the current authoring window and discards its unsaved form values. Published runtime state is unchanged.</p>
    </DialogBody>
    <DialogFooter actions={<><Button onClick={onReview} disabled={reloading}>Review my draft</Button><Button intent="warning" icon="refresh" loading={reloading} onClick={onReload}>Reload latest</Button></>}/>
  </Dialog>;
}

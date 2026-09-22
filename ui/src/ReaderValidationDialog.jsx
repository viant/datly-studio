import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, Icon, Tag } from '@blueprintjs/core';

const checks = [
  ['data-connection', 'Active connector bindings'],
  ['code', 'Datly runtime contract and linked types'],
  ['document', 'Embedded resources and typed input/output'],
  ['route', 'HTTP route and MCP tool identities'],
  ['build', 'Executable reader initialization'],
];

export function ReaderValidationDialog({ isOpen, version, canUseDQL, onClose, onValidate, onOpenSource }) {
  const [validating, setValidating] = useState(false);
  const [result, setResult] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen) return;
    setResult(null);
    setError('');
    setValidating(false);
  }, [isOpen, version?.sourceRevision]);

  const validate = async () => {
    setValidating(true);
    setResult(null);
    setError('');
    try {
      setResult(await onValidate());
    } catch (cause) {
      setError(cause.message);
    } finally {
      setValidating(false);
    }
  };

  const valid = result?.valid === true;
  const previouslyValid=!result&&version?.compileStatus==='valid'&&Boolean(version?.validatedAt);
  const diagnostics = result?.diagnostics ?? [];
  const counts=diagnostics.reduce((summary,item)=>{const severity=normalizeSeverity(item.severity);summary[severity]=(summary[severity]||0)+1;return summary;},{error:0,warning:0,info:0});
  return (
    <Dialog className="studio-connector-dialog studio-validation-dialog" isOpen={isOpen} onClose={onClose} title="Validate reader draft" icon="endorsed" canOutsideClickClose={!validating}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Validation materializes the publishable Datly runtime contract without executing the reader’s business query or changing the active generation.</p>
        <div className="studio-validation-revision">
          <span>Draft v{version?.versionNo ?? '—'} · revision {version?.sourceRevision ?? '—'}{version?.validatedAt?` · checked ${formatValidationDate(version.validatedAt)}`:''}</span>
          <Tag minimal intent={version?.compileStatus === 'valid' ? 'success' : version?.compileStatus === 'invalid' ? 'danger' : 'warning'}>{version?.compileStatus ?? 'pending'}</Tag>
        </div>
        <section className="studio-validation-checks" aria-label="Validation stages">
          {checks.map(([icon, label]) => <div key={label}><Icon icon={validating?'time':valid||previouslyValid?'small-tick':icon} intent={valid||previouslyValid?'success':validating?'primary':'none'}/><span>{label}</span><Tag minimal intent={valid||previouslyValid?'success':validating?'primary':'none'}>{validating?'checking':result?'checked':previouslyValid?'previously passed':'not run'}</Tag></div>)}
        </section>
        {error && <Callout intent="danger" title="Validation request failed" role="alert">{error}</Callout>}
        {result && <Callout intent={valid ? 'success' : 'danger'} title={valid ? 'Runtime contract is ready' : 'Runtime contract is not ready'} role="status">
          {valid ? 'This exact source revision may proceed to publication.' : 'The draft remains editable and the current runtime generation is unchanged.'}
        </Callout>}
        {diagnostics.length > 0 && <section className="studio-validation-diagnostics" aria-label="Validation diagnostics"><div className="studio-validation-diagnostic-summary"><strong>{diagnostics.length} {diagnostics.length===1?'diagnostic':'diagnostics'}</strong>{counts.error>0&&<Tag minimal intent="danger">{counts.error} errors</Tag>}{counts.warning>0&&<Tag minimal intent="warning">{counts.warning} warnings</Tag>}{counts.info>0&&<Tag minimal>{counts.info} info</Tag>}{canUseDQL&&onOpenSource&&<Button small minimal icon="code" onClick={onOpenSource}>Open component source</Button>}</div>
          {diagnostics.map((item, index) => <DiagnosticItem item={item} index={index} key={`${item.code}-${index}`}/>)}
        </section>}
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={validating}>Close</Button><Button intent="primary" icon="endorsed" loading={validating} onClick={validate}>Validate revision</Button></>} />
    </Dialog>
  );
}

function DiagnosticItem({item,index}){const severity=normalizeSeverity(item?.severity);const location=item?.line?`Line ${item.line}${item.column?`, column ${item.column}`:''}`:'';return <article className={`studio-validation-diagnostic ${severity}`}><div><Tag minimal intent={severity==='error'?'danger':severity==='warning'?'warning':'none'}>{severity}</Tag><strong>{item?.code||`Diagnostic ${index+1}`}</strong>{location&&<code>{location}</code>}</div><p>{item?.message||'Validation failed without a diagnostic message.'}</p>{item?.hint&&<div className="studio-validation-hint"><strong>How to fix</strong><span>{item.hint}</span></div>}</article>;}
function normalizeSeverity(value){const severity=String(value||'error').toLowerCase();return severity==='warn'?'warning':['error','warning','info'].includes(severity)?severity:'error';}
function formatValidationDate(value){const date=new Date(value);return Number.isNaN(date.getTime())?'unknown time':date.toLocaleString();}

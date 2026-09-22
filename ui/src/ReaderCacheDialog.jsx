import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, NumericInput, Tag, TextArea } from '@blueprintjs/core';
import { parseWarmupCases, validateWarmupCases, warmupCaseCount, warmupCaseExpression } from './warmupPlan.js';

export function ReaderCacheDialog({ api, report, version, canWarmup, isOpen, structure, onClose, onApply, onWarmup }) {
  const cache = structure?.component?.settings?.cache;
  const warmupConfig = cache?.warmup;
  const columns = structure?.component?.rootView?.columns ?? [];
  const connectors = structure?.availableConnectors ?? [];
  const [name, setName] = useState('');
  const [ttl, setTTL] = useState('60s');
  const [location, setLocation] = useState('');
  const [backend, setBackend] = useState('afs');
  const [provider, setProvider] = useState('');
  const [indexColumn, setIndexColumn] = useState('');
  const [indexParameter, setIndexParameter] = useState('');
  const [connector, setConnector] = useState('');
  const [maxCases, setMaxCases] = useState(30);
  const [limit, setLimit] = useState(100);
  const [fieldNames, setFieldNames] = useState('');
  const [caseLines, setCaseLines] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [warmup, setWarmup] = useState(null);
  const [history, setHistory] = useState([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [warming, setWarming] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setName(cache?.name ?? ''); setTTL(cache?.ttl ?? '60s'); setLocation(cache?.location ?? '');
    setBackend(cache?.provider ? 'aerospike' : 'afs'); setProvider(cache?.provider ?? '');
    setIndexColumn(warmupConfig?.indexColumn ?? columns[0]?.source ?? columns[0]?.name ?? '');
    setIndexParameter(warmupConfig?.indexParameter ?? ''); setConnector(warmupConfig?.connector ?? '');
    setMaxCases(warmupConfig?.maxCases ?? 30); setLimit(warmupConfig?.limit ?? 100);
    setFieldNames((warmupConfig?.fieldNames ?? []).join(', ')); setCaseLines(renderCases(warmupConfig));
    setSaving(false); setError(''); setWarmup(null); setHistory([]); setWarming(false);
  }, [isOpen, cache, warmupConfig, columns]);

  useEffect(() => {
    if (!isOpen || !api || !report?.id || !version?.versionNo) return undefined;
    let cancelled = false;
    setHistoryLoading(true);
    api.listWarmupRuns(report.id,version.versionNo,{limit:10}).then((page) => { if (!cancelled) { const items=page?.items??[]; setHistory(items); if(items[0]) setWarmup(items[0]); } }).catch((cause) => !cancelled && setError(cause.message)).finally(() => !cancelled && setHistoryLoading(false));
    return () => { cancelled = true; };
  }, [api, isOpen, report?.id, version?.versionNo]);

  const cases = useMemo(() => parseWarmupCases(caseLines), [caseLines]);
  const caseErrors = useMemo(() => validateWarmupCases(cases), [cases]);
  const caseCount = useMemo(() => warmupCaseCount(cases), [cases]);
  const caseExpression = useMemo(() => warmupCaseExpression(cases), [cases]);
  const activeRun = warmup && ['accepted','running'].includes(warmup.status);
  const ready = name.trim() && /^\d+(ms|s|m|h)$/.test(ttl) && location.trim() && (backend === 'afs' || provider.trim());

  const saveCache = async () => {
    if (!ready || location.includes("'")) { setError('Provide a cache name, positive duration, location, and host-approved provider reference for Aerospike.'); return; }
    setSaving(true); setError('');
    const options = [{ name: 'WithLocation', args: [quote(location.trim())] }];
    if (backend === 'aerospike') options.unshift({ name: 'WithProvider', args: [quote(provider.trim())] });
    try { await onApply({ type: 'setSetting', setting: { name: 'cache', args: [quote(name.trim()), quote(ttl)], options } }); }
    catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  const saveWarmup = async () => {
    if (!cache?.enabled) { setError('Save an enabled cache before authoring warmup.'); return; }
    if (caseErrors.length) { setError(caseErrors[0]); return; }
    if (!indexColumn.trim() || maxCases < 1 || limit < 1 || caseCount > maxCases) { setError(`Warmup needs an index column and positive budgets; ${caseCount} cases exceed MaxCases ${maxCases}.`); return; }
    const args = [quote(indexColumn.trim())];
    if (connector) args.push(quote(`Connector=${connector}`));
    if (indexParameter.trim()) args.push(quote(`IndexParameter=${indexParameter.trim()}`));
    args.push(quote(`MaxCases=${maxCases}`), quote(`Limit=${limit}`));
    const fields = fieldNames.split(',').map((item) => item.trim()).filter(Boolean);
    if (fields.length) args.push(quote(`FieldNames=${fields.join(',')}`));
    for (const item of cases) args.push(quote(`${item.name}=${item.values.join(',')}`));
    setSaving(true); setError('');
    try { await onApply({ type: 'setSetting', setting: { name: 'cache_warmup', args } }); }
    catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  const removeWarmup = async () => {
    setSaving(true); setError('');
    try { await onApply({ type: 'setSetting', setting: { name: 'cache_warmup', remove: true } }); }
    catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  const executeWarmup = async () => {
    setWarming(true); setError('');
    try { const run=await onWarmup(); setWarmup(run); setHistory((current)=>[run,...current.filter((item)=>item.runId!==run.runId)]); }
    catch (cause) { setError(cause.message); }
    finally { setWarming(false); }
  };

  useEffect(() => {
    if (!isOpen || !api || !report?.id || !warmup?.runId || !['accepted','running'].includes(warmup.status)) return undefined;
    let cancelled=false; let timer;
    const poll=async()=>{try{const run=await api.getWarmupRun(report.id,warmup.runId);if(cancelled)return;setWarmup(run);setHistory((current)=>[run,...current.filter((item)=>item.runId!==run.runId)]);if(['accepted','running'].includes(run.status))timer=setTimeout(poll,500);}catch(cause){if(!cancelled)setError(cause.message);}};
    timer=setTimeout(poll,350);
    return()=>{cancelled=true;clearTimeout(timer);};
  },[api,isOpen,report?.id,warmup?.runId,warmup?.status]);

  return <Dialog className="studio-connector-dialog studio-cache-dialog" isOpen={isOpen} onClose={onClose} title="Cache & warmup" icon="database" canOutsideClickClose={!saving && !warming}>
    <DialogBody className="studio-connector-dialog-body">
      <p className="studio-dialog-lead">Configure Datly’s native prepared-view cache, then author a bounded server-owned warmup plan. Browser values never contain cache credentials.</p>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      <div className="studio-cache-layout"><div className="studio-cache-configuration">
      <section className="studio-cache-section"><div className="studio-panel-heading"><div><h3>Cache storage</h3><p>One explicit backend; no automatic AFS/Aerospike fallback.</p></div>{cache?.enabled && <Tag minimal intent="success">enabled</Tag>}</div>
        <div className="studio-form-grid"><FormGroup label="Backend" labelFor="cache-backend"><HTMLSelect id="cache-backend" value={backend} onChange={(event)=>setBackend(event.target.value)} fill disabled={saving}><option value="afs">AFS</option><option value="aerospike">Aerospike</option></HTMLSelect></FormGroup><FormGroup label="TTL" labelFor="cache-ttl" helperText="ms, s, m, or h"><InputGroup id="cache-ttl" value={ttl} onChange={(event)=>setTTL(event.target.value)} disabled={saving}/></FormGroup></div>
        <FormGroup label="Cache name" labelFor="cache-name"><InputGroup id="cache-name" value={name} onChange={(event)=>setName(event.target.value)} placeholder="product-status" disabled={saving}/></FormGroup>
        {backend === 'aerospike' && <FormGroup label="Approved provider" labelFor="cache-provider" helperText="Host-owned provider reference; never credentials."><InputGroup id="cache-provider" value={provider} onChange={(event)=>setProvider(event.target.value)} placeholder="aerospike://approved-provider" disabled={saving}/></FormGroup>}
        <FormGroup label="Location" labelFor="cache-location"><InputGroup id="cache-location" value={location} onChange={(event)=>setLocation(event.target.value)} placeholder={backend==='afs'?'/var/cache/datly/products':'namespace/set'} disabled={saving}/></FormGroup>
        <Button intent="primary" icon="floppy-disk" loading={saving} disabled={!ready} onClick={saveCache}>Save cache storage</Button>
      </section>

      <section className="studio-cache-section"><div className="studio-panel-heading"><div><h3>Warmup plan</h3><p>Estimate the complete case product before authoring or execution.</p></div><Tag minimal intent={caseCount <= maxCases ? 'success' : 'danger'}>{caseCount} / {maxCases} cases</Tag></div>
        <div className="studio-form-grid"><FormGroup label="Index column" labelFor="warmup-index"><HTMLSelect id="warmup-index" value={indexColumn} onChange={(event)=>setIndexColumn(event.target.value)} fill disabled={saving}><option value="">Select a column</option>{columns.map((column)=><option key={column.source||column.name} value={column.source||column.name}>{column.name}</option>)}</HTMLSelect></FormGroup><FormGroup label="Index parameter" labelFor="warmup-parameter"><InputGroup id="warmup-parameter" value={indexParameter} onChange={(event)=>setIndexParameter(event.target.value)} placeholder="TenantID" disabled={saving}/></FormGroup></div>
        <div className="studio-form-grid"><FormGroup label="Warmup connector" labelFor="warmup-connector"><HTMLSelect id="warmup-connector" value={connector} onChange={(event)=>setConnector(event.target.value)} fill disabled={saving}><option value="">Component connector</option>{connectors.map((name)=><option key={name} value={name}>{name}</option>)}</HTMLSelect></FormGroup><FormGroup label="Selected fields" labelFor="warmup-fields"><InputGroup id="warmup-fields" value={fieldNames} onChange={(event)=>setFieldNames(event.target.value)} placeholder="id, status, total" disabled={saving}/></FormGroup></div>
        <div className="studio-form-grid"><FormGroup label="Max cases" labelFor="warmup-max"><NumericInput id="warmup-max" min={1} value={maxCases} onValueChange={setMaxCases} fill disabled={saving}/></FormGroup><FormGroup label="Row limit" labelFor="warmup-limit"><NumericInput id="warmup-limit" min={1} value={limit} onValueChange={setLimit} fill disabled={saving}/></FormGroup></div>
        <FormGroup label="Case dimensions" labelFor="warmup-cases" helperText="One Name=value1,value2 dimension per line. Case count is the Cartesian product."><TextArea id="warmup-cases" value={caseLines} onChange={(event)=>setCaseLines(event.target.value)} fill rows={4} className="studio-code-input" placeholder={'Period=today,yesterday\nGranularity=hour,day'} disabled={saving}/></FormGroup>
        <div className={`studio-case-product ${caseErrors.length||caseCount>maxCases?'invalid':''}`}><strong>{caseExpression} = {caseCount} {caseCount===1?'case':'cases'}</strong><span>MaxCases {maxCases} · {Math.max(0,maxCases-caseCount)} remaining · row limit {limit}</span>{caseErrors.length>0&&<span>{caseErrors[0]}</span>}</div>
        <div className="studio-warmup-actions"><Button intent="primary" icon="heatmap" loading={saving} disabled={!cache?.enabled || caseErrors.length>0 || caseCount > maxCases} onClick={saveWarmup}>Save warmup plan</Button>{warmupConfig && <Button icon="trash" intent="danger" disabled={saving||warming||activeRun} onClick={removeWarmup}>Remove plan</Button>}<Button icon="play" loading={warming||activeRun} disabled={!canWarmup||!warmupConfig||saving||activeRun} title={!canWarmup?'Publish permission is required to run warmup':''} onClick={executeWarmup}>{history.length?'Run again':'Run warmup'}</Button></div>
      </section>
      </div><aside className="studio-warmup-evidence-panel" aria-label="Warmup operation evidence"><div className="studio-panel-heading"><div><h3>Operation evidence</h3><p>Server-owned runs survive this dialog and client disconnects.</p></div>{historyLoading&&<Tag minimal>Loading</Tag>}</div>
        {!warmup&&!historyLoading?<div className="studio-empty-compact">No warmup has been run for this version.</div>:warmup&&<WarmupEvidence run={warmup}/>}
        {history.length>0&&<div className="studio-warmup-history"><h4>Run history</h4>{history.slice(0,8).map((run)=><button type="button" key={run.runId} className={warmup?.runId===run.runId?'selected':''} onClick={()=>setWarmup(run)}><Tag minimal intent={runIntent(run.status)}>{run.status}</Tag><span><strong>{run.plannedCases||0} planned · {run.entries||0} prepared</strong><small>{formatDate(run.requestedAt)} · {run.requestedBy}</small></span></button>)}</div>}
      </aside></div>
    </DialogBody>
    <DialogFooter actions={<Button onClick={onClose} disabled={saving||warming}>Close</Button>}/>
  </Dialog>;
}

function renderCases(warmup) { return (warmup?.cases?.[0]?.set ?? []).map((item)=>`${item.name}=${(item.values??[]).join(',')}`).join('\n'); }
function quote(value) { return `'${String(value).replaceAll("'", "''")}'`; }
function formatDuration(value) { const nanoseconds=Number(value);if(!Number.isFinite(nanoseconds))return'completed';if(nanoseconds<1e6)return`${Math.round(nanoseconds/1e3)}µs`;return`${(nanoseconds/1e6).toFixed(1)}ms`; }
function formatDate(value){if(!value)return'—';const date=new Date(value);return Number.isNaN(date.valueOf())?value:new Intl.DateTimeFormat(undefined,{month:'short',day:'numeric',hour:'numeric',minute:'2-digit'}).format(date);}
function runIntent(status){return status==='completed'?'success':status==='failed'?'danger':status==='partial'||status==='canceled'?'warning':'primary';}
function WarmupEvidence({run}){const terminal=['completed','partial','failed','canceled'].includes(run.status);return <div className="studio-warmup-run" aria-live="polite"><div><Tag intent={runIntent(run.status)}>{run.status}</Tag><code>{run.runId}</code></div><dl><div><dt>Version</dt><dd>v{run.versionNo} · rev {run.sourceRevision}</dd></div><div><dt>Plan</dt><dd>{run.completedCases||0} / {run.plannedCases||0} cases</dd></div><div><dt>Plan key</dt><dd><code>{run.planKey?.slice(0,16)||'—'}</code></dd></div><div><dt>Budgets</dt><dd>MaxCases {run.maxCases??'—'} · rows {run.rowLimit??'—'}</dd></div><div><dt>Prepared</dt><dd>{run.entries||0} entries</dd></div><div><dt>Duration</dt><dd>{terminal?formatDuration(run.duration):'Running'}</dd></div><div><dt>Connector</dt><dd>{run.target?.connectorName||'Component connector'}</dd></div><div><dt>Cache</dt><dd>{run.target?.cacheName||'Resolving'} · {run.target?.cacheProvider||'—'}</dd></div><div><dt>Index</dt><dd>{run.target?.indexColumn||'—'}</dd></div><div><dt>Requested</dt><dd>{formatDate(run.requestedAt)} · {run.requestedBy}</dd></div></dl>{(run.diagnostics??[]).map((item,index)=><Callout key={`${item.code}-${index}`} intent="danger" title={item.code}>{item.message}</Callout>)}</div>;}

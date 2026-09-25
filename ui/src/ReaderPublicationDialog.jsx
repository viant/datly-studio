import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tag } from '@blueprintjs/core';

export function ReaderPublicationDialog({ isOpen, api, report, version, inspection, onClose, onPublish, onUnpublish, onRollback }) {
  const [reason, setReason] = useState('');
  const [publishing, setPublishing] = useState(false);
  const [result, setResult] = useState(null);
  const [error, setError] = useState('');
  const [publication, setPublication] = useState(null);
  const [runtime, setRuntime] = useState(null);
  const [history, setHistory] = useState([]);
  const [rollbackVersion, setRollbackVersion] = useState('');
  const [loadingContext, setLoadingContext] = useState(false);
  const [contextError, setContextError] = useState('');
  const [pendingAction, setPendingAction] = useState('');
  const [events, setEvents] = useState([]);
  const [servingInspection, setServingInspection] = useState(null);
  const ready = version?.compileStatus === 'valid';

  useEffect(() => {
    if (!isOpen) return;
    setReason('');
    setPublishing(false);
    setResult(null);
    setError(''); setContextError(''); setPublication(null); setRuntime(null); setHistory([]); setEvents([]); setServingInspection(null); setRollbackVersion(''); setPendingAction(''); setLoadingContext(true);
    const publicationRequest=report?.id?api.getPublication(report.id):Promise.resolve(null);
    const historyRequest=report?.id?api.listVersions(report.id,{limit:50}):Promise.resolve({items:[]});
    const eventsRequest=report?.id?api.listPublicationEvents(report.id,{limit:12,offset:0}):Promise.resolve({items:[]});
    Promise.allSettled([publicationRequest,historyRequest,api.getRuntimeStatus(),eventsRequest]).then(([publicationResult,historyResult,runtimeResult,eventsResult])=>{
      const failures=[];
      if(publicationResult.status==='fulfilled'){setPublication(publicationResult.value);const active=publicationResult.value?.activeVersionNo;if(active&&active!==version?.versionNo)api.inspectVersion(report.id,active).then(setServingInspection).catch((cause)=>setContextError((current)=>[current,`serving contract: ${cause.message}`].filter(Boolean).join(' · ')));}else if(publicationResult.reason?.code!=='not_found')failures.push(`publication state: ${publicationResult.reason?.message||'unavailable'}`);
      if(historyResult.status==='fulfilled')setHistory(historyResult.value?.items??[]);else failures.push(`version history: ${historyResult.reason?.message||'unavailable'}`);
      if(runtimeResult.status==='fulfilled')setRuntime(runtimeResult.value);else failures.push(`runtime status: ${runtimeResult.reason?.message||'unavailable'}`);
      if(eventsResult.status==='fulfilled')setEvents(eventsResult.value?.items??[]);else failures.push(`release activity: ${eventsResult.reason?.message||'unavailable'}`);
      setContextError(failures.join(' · '));setLoadingContext(false);
    });
  }, [isOpen, version?.sourceRevision]);
  const refreshEvents=async()=>{if(!report?.id)return;try{const page=await api.listPublicationEvents(report.id,{limit:12,offset:0});setEvents(page?.items??[]);}catch(cause){setContextError((current)=>[current,`release activity: ${cause.message}`].filter(Boolean).join(' · '));}};

  const publish = async () => {
    if (!ready) return;
    setPublishing(true);
    setError('');
    try {
      const outcome = await onPublish(reason.trim());
      setResult(outcome); setPublication(outcome); setRuntime(await api.getRuntimeStatus()); await refreshEvents();
    } catch (cause) {
      setError(cause.message);
    } finally {
      setPublishing(false);
    }
  };
  const unpublish = async () => {
    setPublishing(true); setError('');
    try { const outcome = await onUnpublish(publication?.activeGeneration, reason.trim()); setResult(outcome); setPublication(outcome); setRuntime(await api.getRuntimeStatus()); await refreshEvents(); }
    catch (cause) { setError(cause.message); }
    finally { setPublishing(false); setPendingAction(''); }
  };
  const rollback = async () => {
    const selected = history.find((item) => String(item.versionNo) === rollbackVersion);
    if (!selected || selected.compileStatus !== 'valid') return;
    setPublishing(true); setError('');
    try {
      const outcome = await onRollback(selected, reason.trim());
      setResult(outcome); setPublication(outcome); setRuntime(await api.getRuntimeStatus()); await refreshEvents();
    } catch (cause) { setError(cause.message); }
    finally { setPublishing(false); setPendingAction(''); }
  };
  const selectedRollback = history.find((item) => String(item.versionNo) === rollbackVersion);
  const servingMatchesDraft = Boolean(publication?.specHash && version?.specHash && publication.specHash === version.specHash);
  const comparisonUnavailable = Boolean(publication?.activeVersionNo === version?.versionNo && !servingMatchesDraft);
  const comparisonReady = servingMatchesDraft || Boolean(servingInspection);
  const changes = releaseChangeSummary(servingInspection, inspection, history.find((item)=>item.versionNo===publication?.activeVersionNo), version);

  return <Dialog className="studio-connector-dialog studio-publication-dialog" isOpen={isOpen} onClose={onClose} title="Publish reader" icon="rocket-slant" canOutsideClickClose={!publishing}>
    <DialogBody className="studio-connector-dialog-body">
      <p className="studio-dialog-lead">Publish stages this exact validated revision, asks the dedicated Datly runtime to atomically reload, then marks the generation active only after that reload succeeds.</p>
      <div className="studio-validation-revision"><span>Draft v{version?.versionNo ?? '—'} · revision {version?.sourceRevision ?? '—'}</span><Tag minimal intent={ready ? 'success' : 'danger'}>{version?.compileStatus ?? 'pending'}</Tag></div>
      {loadingContext&&<div className="studio-release-context-loading" role="status">Loading serving state, runtime health, and version history…</div>}
      {contextError&&<Callout intent="warning" title="Some release context is unavailable">{contextError}. Publishing remains governed by server-side validation and atomic activation.</Callout>}
      {runtime && <div className="studio-runtime-status"><span>Generation record</span><strong>{runtime.status}</strong><Tag minimal intent={runtime.status === 'active' ? 'success' : 'warning'}>generation {runtime.activeGeneration ?? '—'} · {runtime.reportCount ?? 0} readers</Tag>{runtime.host?.status === 'ready' && <Tag minimal intent="success">Live HTTP/MCP host ready</Tag>}</div>}
      {runtime?.host?.status === 'unavailable' && <Callout intent="warning" title="Live runtime check failed">The REST and MCP listeners must be running before this revision can activate. Publishing reloads their routes and tools; it does not start the host process.</Callout>}
      {publication && publication.status !== 'unpublished' && <section className="studio-publication-state" aria-label="Publication state"><div><span>Serving</span><strong>v{publication.activeVersionNo} · generation {publication.activeGeneration ?? '—'}</strong></div><div><span>Desired</span><strong>{publication.desiredVersionNo ? `v${publication.desiredVersionNo}` : 'removal'} · generation {publication.desiredGeneration ?? '—'}</strong></div><Tag minimal intent={publication.status === 'active' ? 'success' : 'warning'}>{publication.status}</Tag></section>}
      {publication?.activeVersionNo && <section className="studio-release-impact" aria-label="Serving to proposed change summary"><div className="studio-panel-heading"><div><h3>Serving → proposed</h3><p className="studio-muted">Consumer-facing contract areas changed between v{publication.activeVersionNo} and draft v{version?.versionNo}.</p></div>{comparisonReady&&<Tag minimal intent={changes.length?'warning':'success'}>{changes.length ? `${changes.length} changed` : 'No contract change detected'}</Tag>}</div>{comparisonUnavailable?<Callout intent="warning" title="Detailed comparison unavailable">The serving generation uses an earlier source snapshot of this version. Validate and publish a new immutable version to produce a field-level comparison; Studio will not claim the mutable draft matches.</Callout>:!comparisonReady?<p className="studio-empty-compact">Loading the serving contract comparison…</p>:changes.length?<div>{changes.map((item)=><Tag key={item} minimal>{item}</Tag>)}</div>:<p className="studio-empty-compact">The inspected SQL, inputs, output contract, HTTP/MCP exposure, and resources match the serving version.</p>}</section>}
      {!loadingContext&&<section className="studio-release-events" aria-label="Release activity"><div className="studio-panel-heading"><div><h3>Release activity</h3><p className="studio-muted">Append-only publication, rollback, and unpublish outcomes.</p></div><Tag minimal>{events.length} recent</Tag></div>{events.length===0?<div className="studio-empty-compact">No release operations have been recorded for this reader yet.</div>:<div>{events.map((event)=><article key={event.eventId}><span className="studio-release-event-marker" aria-hidden="true"/><div><strong>{releaseEventTitle(event)}</strong><small>{formatEventTime(event.occurredAt)} · {event.requestedBy||'unknown actor'}{event.generationNo?` · generation ${event.generationNo}`:''}</small>{event.reason&&<p>{event.reason}</p>}{event.failureMessage&&<p className="studio-release-event-failure">{event.failureMessage}</p>}</div><Tag minimal intent={event.status==='succeeded'?'success':'danger'}>{event.status}</Tag></article>)}</div>}</section>}
      {!ready && <Callout intent="warning" title="Validation required">Fix the current validation diagnostics, then validate this exact source revision before publishing.</Callout>}
      {ready && <Callout intent="primary" title="Staged runtime activation">The serving version remains active while Studio validates and reloads the complete desired generation. Database promotion happens only after the runtime confirms the swap; persistence failure compensates to the prior generation.</Callout>}
      <FormGroup label="Release note" labelFor="publication-reason" helperText={pendingAction?'Required for rollback and unpublish audit evidence.':'Optional audit context for publication.'}>
        <InputGroup id="publication-reason" value={reason} onChange={(event) => setReason(event.target.value)} disabled={publishing} placeholder="Describe this release operation" />
      </FormGroup>
      {history.length > 1 && <section className="studio-rollback-panel"><FormGroup label="Rollback to a validated historical version" labelFor="rollback-version" helperText="Rollback runs the same staged runtime activation; it does not rewrite source."><HTMLSelect id="rollback-version" value={rollbackVersion} onChange={(event) => setRollbackVersion(event.target.value)} fill disabled={publishing}><option value="">Select a historical version</option>{history.filter((item) => item.versionNo !== version?.versionNo).map((item) => <option key={item.versionNo} value={item.versionNo}>v{item.versionNo} · rev {item.sourceRevision} · {item.compileStatus}</option>)}</HTMLSelect></FormGroup>{selectedRollback && selectedRollback.compileStatus !== 'valid' && <Callout intent="warning">Selected version must validate before rollback.</Callout>}</section>}
      {publication?.status === 'active' && <Callout intent="warning" title={`Active generation ${publication.activeGeneration ?? '—'}`}>Unpublishing reloads the dynamic host without this reader, its MCP tools, or its published resources.</Callout>}
      {pendingAction==='rollback'&&selectedRollback&&<Callout intent="warning" title={`Confirm rollback to v${selectedRollback.versionNo}`}><p>This activates the historical validated source without rewriting it. Enter a release note, then confirm the staged runtime swap.</p><div className="studio-confirm-actions"><Button onClick={()=>setPendingAction('')} disabled={publishing}>Cancel</Button><Button intent="warning" icon="history" loading={publishing} disabled={!reason.trim()} onClick={rollback}>Confirm rollback</Button></div></Callout>}
      {pendingAction==='unpublish'&&<Callout intent="danger" title="Confirm unpublish"><p>The active reader, MCP tools, resources, and skills will be removed from the next runtime generation. Enter a release note to preserve the audit reason.</p><div className="studio-confirm-actions"><Button onClick={()=>setPendingAction('')} disabled={publishing}>Cancel</Button><Button intent="danger" icon="remove" loading={publishing} disabled={!reason.trim()} onClick={unpublish}>Confirm unpublish</Button></div></Callout>}
      {error && <Callout intent="danger" title="Publication did not activate" role="alert">{error}</Callout>}
      {result?.status === 'unpublished' && <Callout intent="primary" title="Reader removed from active generation" role="status">The dynamic host reloaded without this reader, its MCP tools, resources, and skills.</Callout>}
      {result && result.status !== 'unpublished' && <Callout intent="success" title="Runtime generation active" role="status">Generation {result.activeGeneration} now serves version {result.activeVersionNo}.</Callout>}
    </DialogBody>
    <DialogFooter actions={<><Button onClick={onClose} disabled={publishing}>Close</Button>{selectedRollback && pendingAction!=='rollback' && <Button icon="history" disabled={publishing||selectedRollback.compileStatus !== 'valid'} onClick={()=>setPendingAction('rollback')}>Review rollback to v{selectedRollback.versionNo}</Button>}{publication?.status === 'active'&&pendingAction!=='unpublish'&&<Button intent="danger" icon="remove" disabled={publishing} onClick={()=>setPendingAction('unpublish')}>Review unpublish</Button>}<Button intent="primary" icon="rocket-slant" loading={publishing} disabled={!ready||Boolean(pendingAction)} onClick={publish}>Publish validated revision</Button></>} />
  </Dialog>;
}

function releaseEventTitle(event){const operation=String(event?.operation||'release');const version=event?.versionNo?` v${event.versionNo}`:'';return `${operation.charAt(0).toUpperCase()+operation.slice(1)}${version}`;}
function formatEventTime(value){if(!value)return'Unknown time';const date=new Date(value);return Number.isNaN(date.getTime())?'Unknown time':date.toLocaleString();}

export function releaseChangeSummary(servingInspection,currentInspection,servingVersion,currentVersion){if(!servingInspection||!currentInspection)return[];const before=servingInspection?.structure??{};const after=currentInspection?.structure??{};const result=[];if(changed(before.views,after.views))result.push('SQL / views');if(changed(parameters(before),parameters(after)))result.push('Inputs');if(changed({root:before.component?.rootView,contracts:before.columnContracts},{root:after.component?.rootView,contracts:after.columnContracts}))result.push('Output contract');if(changed({routes:before.component?.routes,settings:before.component?.settings},{routes:after.component?.routes,settings:after.component?.settings}))result.push('HTTP / MCP');if(changed(servingVersion?.resourceManifest,currentVersion?.resourceManifest))result.push('Resources / skills');return result;}
function parameters(structure){return(structure?.declarations??[]).map((item)=>item.parameter).filter(Boolean);}
function changed(left,right){return JSON.stringify(left??null)!==JSON.stringify(right??null);}

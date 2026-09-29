import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tag } from '@blueprintjs/core';

export function ReaderPublicationDialog({ isOpen, api, report, version, inspection, onClose, onPublish, onPromote, onUnpublish, onRollback }) {
  const [reason, setReason] = useState('');
  const [promotionVersion, setPromotionVersion] = useState('');
  const [promotionInspection, setPromotionInspection] = useState(null);
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
    setReason(''); setPromotionVersion(String(version?.versionNo || '')); setPromotionInspection(null);
    setPublishing(false);
    setResult(null);
    setError(''); setContextError(''); setPublication(null); setRuntime(null); setHistory([]); setEvents([]); setServingInspection(null); setRollbackVersion(''); setPendingAction(''); setLoadingContext(true);
    const publicationRequest=report?.id?api.getPublication(report.id):Promise.resolve(null);
    const historyRequest=report?.id?api.listVersions(report.id,{limit:50}):Promise.resolve({items:[]});
    const eventsRequest=report?.id?api.listPublicationEvents(report.id,{limit:12,offset:0}):Promise.resolve({items:[]});
    Promise.allSettled([publicationRequest,historyRequest,api.getRuntimeStatus(),eventsRequest]).then(([publicationResult,historyResult,runtimeResult,eventsResult])=>{
      const failures=[];
      if(publicationResult.status==='fulfilled'){setPublication(publicationResult.value);}else if(publicationResult.reason?.code!=='not_found')failures.push(`publication state: ${publicationResult.reason?.message||'unavailable'}`);
      if(historyResult.status==='fulfilled')setHistory(historyResult.value?.items??[]);else failures.push(`version history: ${historyResult.reason?.message||'unavailable'}`);
      if(runtimeResult.status==='fulfilled')setRuntime(runtimeResult.value);else failures.push(`runtime status: ${runtimeResult.reason?.message||'unavailable'}`);
      if(eventsResult.status==='fulfilled')setEvents(eventsResult.value?.items??[]);else failures.push(`release activity: ${eventsResult.reason?.message||'unavailable'}`);
      setContextError(failures.join(' · '));setLoadingContext(false);
    });
  }, [isOpen, version?.sourceRevision]);
  useEffect(() => {
    setServingInspection(null);
    const activeVersionNo = publication?.activeVersionNo;
    const specHash = publication?.specHash;
    if (!isOpen || !report?.id || !activeVersionNo || activeVersionNo === version?.versionNo) return;
    let cancelled = false;
    api.inspectVersion(report.id, activeVersionNo).then((value) => {
      if (!cancelled) setServingInspection({ versionNo: activeVersionNo, specHash, inspection: value });
    }).catch((cause) => {
      if (!cancelled) setContextError((current) => [current, `serving contract: ${cause.message}`].filter(Boolean).join(' · '));
    });
    return () => { cancelled = true; };
  }, [api, isOpen, report?.id, publication?.activeVersionNo, publication?.specHash, version?.versionNo]);
  const refreshEvents=async()=>{if(!report?.id)return;try{const page=await api.listPublicationEvents(report.id,{limit:12,offset:0});setEvents(page?.items??[]);}catch(cause){setContextError((current)=>[current,`release activity: ${cause.message}`].filter(Boolean).join(' · '));}};

  const selectedPromotion = history.find(item => String(item.versionNo) === promotionVersion) || (String(version?.versionNo) === promotionVersion ? version : null);
  const promotionReady = selectedPromotion?.compileStatus === 'valid' && selectedPromotion?.versionNo !== publication?.activeVersionNo;
  useEffect(()=>{
    setPromotionInspection(null);
    if (!isOpen || !selectedPromotion || selectedPromotion.versionNo === version?.versionNo) return;
    let cancelled=false;
    api.inspectVersion(report.id, selectedPromotion.versionNo).then(value=>{if(!cancelled)setPromotionInspection(value);}).catch(cause=>{if(!cancelled)setContextError(cause.message);});
    return()=>{cancelled=true;};
  },[api,isOpen,report?.id,selectedPromotion?.versionNo,selectedPromotion?.specHash,version?.versionNo]);
  const publish = async () => {
    if (!promotionReady) return;
    setPublishing(true);
    setError('');
    try {
      const outcome = selectedPromotion.versionNo === version?.versionNo
        ? await onPublish(reason.trim())
        : await onPromote(selectedPromotion,reason.trim());
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
  const rollbackCandidates = history.filter((item) => item.compileStatus === 'valid' && item.versionNo !== version?.versionNo && item.versionNo !== publication?.activeVersionNo);
  const selectedRollback = rollbackCandidates.find((item) => String(item.versionNo) === rollbackVersion);
  const servingMatchesDraft = Boolean(publication?.specHash && version?.specHash && publication.specHash === version.specHash);
  const comparisonUnavailable = Boolean(publication?.activeVersionNo === version?.versionNo && !servingMatchesDraft);
  const servingContract = servingInspection && servingInspection.versionNo === publication?.activeVersionNo && servingInspection?.specHash === publication?.specHash ? servingInspection.inspection : null;

  const proposal = selectedPromotion?.versionNo === version?.versionNo ? inspection : promotionInspection;
  const publishedContract = servingContract || (servingMatchesDraft && publication?.activeVersionNo === version?.versionNo ? inspection : null);
  const selectedMatchesPublished = Boolean(selectedPromotion?.specHash && selectedPromotion.specHash === publication?.specHash);
  const comparisonReady = selectedMatchesPublished || Boolean(publishedContract && proposal);
  const changes = releaseChangeSummary(publishedContract, proposal, history.find((item)=>item.versionNo===publication?.activeVersionNo), selectedPromotion);
  const proposalReady = selectedPromotion?.specHash === publication?.specHash || Boolean(proposal);

  return <Dialog className="studio-connector-dialog studio-publication-dialog" isOpen={isOpen} onClose={onClose} title="Release reader" icon="rocket-slant" canOutsideClickClose={!publishing}>
    <DialogBody className="studio-connector-dialog-body">
      <FormGroup label="Publish version" labelFor="promotion-version"><HTMLSelect id="promotion-version" value={promotionVersion} disabled={publishing || loadingContext} onChange={event=>setPromotionVersion(event.target.value)}>{(history.length?history:version?[version]:[]).map(item=><option key={item.versionNo} value={String(item.versionNo)}>v{item.versionNo}{item.versionNo===publication?.activeVersionNo?' · Published':` · ${item.compileStatus}`}</option>)}</HTMLSelect></FormGroup>
      <p className="studio-muted">Only one version is published per component. Publishing replaces the current version after successful activation.</p>
      <div className="studio-validation-revision"><span>{selectedPromotion?.versionNo === publication?.activeVersionNo ? 'Published' : selectedPromotion?.state === 'draft' ? 'Draft' : 'Version'} v{selectedPromotion?.versionNo ?? '—'} · revision {selectedPromotion?.sourceRevision ?? '—'}</span><Tag minimal intent={selectedPromotion?.compileStatus === 'valid' ? 'success' : 'danger'}>{selectedPromotion?.compileStatus ?? 'pending'}</Tag></div>
      {loadingContext&&<div className="studio-release-context-loading" role="status">Loading serving state, runtime health, and version history…</div>}
      {contextError&&<Callout intent="warning" title="Some release context is unavailable">{contextError}. Publishing remains governed by server-side validation and atomic activation.</Callout>}
      {runtime && <div className="studio-runtime-status"><span>Generation record</span><strong>{runtime.status}</strong><Tag minimal intent={runtime.status === 'active' ? 'success' : 'warning'}>generation {runtime.activeGeneration ?? '—'} · {runtime.reportCount ?? 0} readers</Tag>{runtime.host?.status === 'ready' && <Tag minimal intent="success">Live HTTP/MCP host ready</Tag>}</div>}
      {runtime?.host?.status === 'unavailable' && <Callout intent="warning" title="Live runtime check failed">The REST and MCP listeners must be running before this revision can activate. Publishing reloads their routes and tools; it does not start the host process.</Callout>}
      {publication && publication.status !== 'unpublished' && <section className="studio-publication-state" aria-label="Publication state"><div><span>Published version</span><strong>v{publication.activeVersionNo} · generation {publication.activeGeneration ?? '—'}</strong></div><div><span>Desired</span><strong>{publication.desiredVersionNo ? `v${publication.desiredVersionNo}` : 'removal'} · generation {publication.desiredGeneration ?? '—'}</strong></div><Tag minimal intent={publication.status === 'active' ? 'success' : 'warning'}>{publication.status}</Tag></section>}
      {publication?.activeVersionNo && <section className="studio-release-impact" aria-label="Serving to proposed change summary"><div className="studio-panel-heading"><div><h3>Published → selected version</h3></div>{comparisonReady&&proposalReady&&<Tag minimal intent={changes.length?'warning':'success'}>{changes.length ? `${changes.length} changed` : 'No contract change detected'}</Tag>}</div>{comparisonUnavailable?<Callout intent="warning" title="Detailed comparison unavailable">The serving generation uses an earlier source snapshot of this version. Validate and publish a new immutable version to produce a field-level comparison; Studio will not claim the mutable draft matches.</Callout>:!comparisonReady?<p className="studio-empty-compact">Loading the serving contract comparison…</p>:changes.length?<div>{changes.map((item)=><Tag key={item} minimal>{item}</Tag>)}</div>:null}</section>}
      {!loadingContext&&<details className="studio-release-events"><summary>Release activity · {events.length}</summary>{events.length===0?<div className="studio-empty-compact">No release operations have been recorded for this reader yet.</div>:<div>{events.map((event)=><article key={event.eventId}><span className="studio-release-event-marker" aria-hidden="true"/><div><strong>{releaseEventTitle(event)}</strong><small>{formatEventTime(event.occurredAt)} · {event.requestedBy||'unknown actor'}{event.generationNo?` · generation ${event.generationNo}`:''}</small>{event.reason&&<p>{event.reason}</p>}{event.failureMessage&&<p className="studio-release-event-failure">{event.failureMessage}</p>}</div><Tag minimal intent={event.status==='succeeded'?'success':'danger'}>{event.status}</Tag></article>)}</div>}</details>}
      {!ready && <Callout intent="warning" title="Validation required">Validate this draft to publish. Rollback and unpublish remain available.</Callout>}
      <FormGroup label="Release note" labelFor="publication-reason" helperText={pendingAction?'Required for rollback and unpublish.':undefined}>
        <InputGroup id="publication-reason" value={reason} onChange={(event) => setReason(event.target.value)} disabled={publishing} placeholder="Describe this release operation" />
      </FormGroup>
      {rollbackCandidates.length > 0 && <section className="studio-rollback-panel"><FormGroup label="Rollback to a validated historical version" labelFor="rollback-version" ><HTMLSelect id="rollback-version" value={rollbackVersion} onChange={(event) => setRollbackVersion(event.target.value)} fill disabled={publishing}><option value="">Select a historical version</option>{rollbackCandidates.map((item) => <option key={item.versionNo} value={item.versionNo}>v{item.versionNo} · rev {item.sourceRevision} · {item.compileStatus}</option>)}</HTMLSelect></FormGroup></section>}
      {pendingAction==='rollback'&&selectedRollback&&<Callout intent="warning" title={`Confirm rollback to v${selectedRollback.versionNo}`}><p>This activates the historical validated source without rewriting it. Enter a release note, then confirm the staged runtime swap.</p><div className="studio-confirm-actions"><Button onClick={()=>setPendingAction('')} disabled={publishing}>Cancel</Button><Button intent="warning" icon="history" loading={publishing} disabled={!reason.trim()} onClick={rollback}>Confirm rollback</Button></div></Callout>}
      {pendingAction==='unpublish'&&<Callout intent="danger" title="Confirm unpublish"><p>The active reader, MCP tools, resources, and skills will be removed from the next runtime generation. Enter a release note to preserve the audit reason.</p><div className="studio-confirm-actions"><Button onClick={()=>setPendingAction('')} disabled={publishing}>Cancel</Button><Button intent="danger" icon="remove" loading={publishing} disabled={!reason.trim()} onClick={unpublish}>Confirm unpublish</Button></div></Callout>}
      {error && <Callout intent="danger" title="Publication did not activate" role="alert">{error}</Callout>}
      {result?.status === 'unpublished' && <Callout intent="primary" title="Reader removed from active generation" role="status">The dynamic host reloaded without this reader, its MCP tools, resources, and skills.</Callout>}
      {result && result.status !== 'unpublished' && <Callout intent="success" title="Runtime generation active" role="status">Published version is now v{result.activeVersionNo} · generation {result.activeGeneration}.</Callout>}
    </DialogBody>
    <DialogFooter actions={<><Button onClick={onClose} disabled={publishing}>Close</Button>{selectedRollback && pendingAction!=='rollback' && <Button icon="history" disabled={publishing||selectedRollback.compileStatus !== 'valid'} onClick={()=>setPendingAction('rollback')}>Review rollback to v{selectedRollback.versionNo}</Button>}{publication?.status === 'active'&&pendingAction!=='unpublish'&&<Button intent="danger" icon="remove" disabled={publishing} onClick={()=>setPendingAction('unpublish')}>Review unpublish</Button>}<Button intent="primary" icon="rocket-slant" loading={publishing} disabled={loadingContext||!promotionReady||Boolean(pendingAction)||Boolean(selectedPromotion?.versionNo !== version?.versionNo && !onPromote)} onClick={publish}>Publish selected version</Button></>} />
  </Dialog>;
}

function releaseEventTitle(event){const operation=String(event?.operation||'release');const version=event?.versionNo?` v${event.versionNo}`:'';return `${operation.charAt(0).toUpperCase()+operation.slice(1)}${version}`;}
function formatEventTime(value){if(!value)return'Unknown time';const date=new Date(value);return Number.isNaN(date.getTime())?'Unknown time':date.toLocaleString();}

export function releaseChangeSummary(servingInspection,currentInspection,servingVersion,currentVersion){if(!servingInspection||!currentInspection)return[];const before=servingInspection?.structure??{};const after=currentInspection?.structure??{};const result=[];if(changed(before.views,after.views))result.push('SQL / views');if(changed(parameters(before),parameters(after)))result.push('Inputs');if(changed({root:before.component?.rootView,contracts:before.columnContracts},{root:after.component?.rootView,contracts:after.columnContracts}))result.push('Output contract');if(changed({routes:before.component?.routes,settings:before.component?.settings},{routes:after.component?.routes,settings:after.component?.settings}))result.push('HTTP / MCP');if(changed(servingVersion?.resourceManifest,currentVersion?.resourceManifest))result.push('Resources / skills');return result;}
function parameters(structure){return(structure?.declarations??[]).map((item)=>item.parameter).filter(Boolean);}
function changed(left,right){return JSON.stringify(left??null)!==JSON.stringify(right??null);}

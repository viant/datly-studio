import React from 'react';
import { Button, Callout, Card, Spinner, Tag } from '@blueprintjs/core';
import { environmentChecks, overviewIssues } from './overviewModel.js';

export function OverviewWorkspace({ data, loading, error, onRefresh, onNavigate, onOpenComponent }) {
  const reports = data?.reports ?? [];
  const issues = overviewIssues(data);
  const checks = environmentChecks(data);
  const sectionErrors = data?.errors ?? {};
  return <main className="studio-workspace studio-overview-workspace">
    <div className="studio-overview-header"><div><h1 className="studio-page-heading">Overview</h1><p className="studio-page-description">Resume authoring and review anything blocking publication.</p></div><div><Button icon="refresh" loading={loading} onClick={onRefresh}>Refresh</Button><Button intent="primary" icon="application" onClick={() => onNavigate('reports')}>Components</Button></div></div>
    {error && <Callout intent="danger" title="Overview unavailable" role="alert">{error}<Button small minimal intent="danger" onClick={onRefresh}>Retry</Button></Callout>}
    {loading && !data ? <div className="studio-loading"><Spinner size={28}/></div> : data && <>
      <div className="studio-overview-grid">
        <section className="studio-overview-section" aria-labelledby="continue-title"><div className="studio-overview-section-heading"><h2 id="continue-title">Continue authoring</h2><Button small minimal onClick={() => onNavigate('reports')}>View catalog</Button></div><Card className="studio-card" elevation={0}>{sectionErrors.reports ? <Callout intent="warning" title="Component catalog unavailable">{sectionErrors.reports}</Callout> : reports.length === 0 ? <div className="studio-empty-compact">No components are available yet.</div> : <div className="studio-overview-component-list">{reports.slice(0, 6).map((report) => <button type="button" key={report.id} onClick={() => onOpenComponent(report)}><span><strong>{report.title}</strong><small>{report.namespace} · {report.defaultConnectorName}</small></span><Tag minimal intent={report.status === 'active' ? 'success' : 'warning'}>{report.status}</Tag><span>{formatDate(report.updatedAt)}</span></button>)}</div>}</Card></section>

        <section className="studio-overview-section" aria-labelledby="readiness-title"><div className="studio-overview-section-heading"><h2 id="readiness-title">Environment readiness</h2></div><Card className="studio-card" elevation={0}><div className="studio-readiness-list">{checks.map((check) => <button type="button" key={check.key} onClick={() => onNavigate(check.section)}><Tag minimal icon={check.ready ? 'tick-circle' : 'warning-sign'} intent={check.ready ? 'success' : 'warning'}>{check.ready ? 'Ready' : 'Review'}</Tag><span><strong>{check.label}</strong><small>{check.value}</small></span><span aria-hidden="true">→</span></button>)}</div></Card></section>
      </div>

      <section className="studio-overview-section studio-overview-attention" aria-labelledby="attention-title"><div className="studio-overview-section-heading"><h2 id="attention-title">Needs attention</h2><Tag minimal intent={issues.length ? 'warning' : 'success'}>{issues.length ? `${issues.length} open` : 'Clear'}</Tag></div>{issues.length === 0 ? <div className="studio-overview-clear">No issues require action.</div> : <div className="studio-overview-issues">{issues.map((issue) => <button type="button" key={issue.key} onClick={() => onNavigate(issue.section)}><Tag minimal intent={issue.intent}>{issue.intent === 'danger' ? 'Blocked' : 'Review'}</Tag><span><strong>{issue.title}</strong><small>{issue.detail}</small></span><span aria-hidden="true">→</span></button>)}</div>}</section>
    </>}
  </main>;
}

function formatDate(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date);
}

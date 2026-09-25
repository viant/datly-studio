import React, { useMemo, useState } from 'react';
import { Button, Checkbox, FormGroup, HTMLSelect, Tag } from '@blueprintjs/core';
import { ResourceAccessEditor } from './ResourceAccessEditor.jsx';
import { createReviewFixture, reviewScenarios } from './aclReviewFixture.js';
import './aclReview.css';

export function ACLReviewFrame({ scenario, kind }) {
  const fixture = useMemo(() => createReviewFixture(scenario, kind), [scenario, kind]);
  return <main className="acl-review-frame-content">
    <p className="acl-review-synthetic">Synthetic ACL preview · changes stay in this window</p>
    <ResourceAccessEditor api={fixture.api} resource={fixture.resource} actions={fixture.actions}/>
  </main>;
}

export function ACLReviewWindow() {
  const [viewport, setViewport] = useState(() => typeof window !== 'undefined' && window.innerWidth < 650 ? 'phone' : 'desktop');
  const [scenario, setScenario] = useState('editable');
  const [kind, setKind] = useState('component');
  const [iteration, setIteration] = useState(0);
  const widths = { desktop: 1200, tablet: 768, phone: 390 };
  const params = new URLSearchParams({ frame: '1', scenario, kind, iteration: String(iteration) });
  return <main className="acl-review-window">
    <header className="acl-review-heading">
      <div><h1>ACL UX review</h1><p>Review the shared Datly Studio permissions editor before release.</p></div>
      <Tag minimal intent="primary">Synthetic data only</Tag>
    </header>
    <section className="acl-review-controls" aria-label="Preview controls">
      <FormGroup label="Viewport" labelFor="review-viewport"><HTMLSelect id="review-viewport" value={viewport} onChange={event => setViewport(event.target.value)}>
        <option value="desktop">Desktop · 1200 px</option><option value="tablet">Tablet · 768 px</option><option value="phone">Phone · 390 px</option>
      </HTMLSelect></FormGroup>
      <FormGroup label="Resource" labelFor="review-kind"><HTMLSelect id="review-kind" value={kind} onChange={event => setKind(event.target.value)}><option value="component">Component</option><option value="skill">Skill</option><option value="report">Report</option></HTMLSelect></FormGroup>
      <FormGroup label="Scenario" labelFor="review-scenario"><HTMLSelect id="review-scenario" value={scenario} onChange={event => setScenario(event.target.value)}>{Object.entries(reviewScenarios).map(([id, label]) => <option key={id} value={id}>{label}</option>)}</HTMLSelect></FormGroup>
      <Button icon="refresh" onClick={() => setIteration(value => value + 1)}>Reset preview</Button>
    </section>
    <div className="acl-review-layout">
      <section className="acl-review-stage" aria-label="ACL preview viewport">
        <iframe key={params.toString()} src={`./acl-review.html?${params}`} title="ACL permissions preview" style={{ width: widths[viewport] }} />
      </section>
      <aside className="acl-review-checklist" aria-label="UX review checklist">
        <h2>Review checklist</h2>
        <p>These checks are local notes, not an automated approval.</p>
        {['Actions and access modes are clear', 'Provider choices are easy to find', 'Entity type and ID are visible', 'All / any rules read correctly', 'Keyboard focus is visible', 'Read-only controls cannot change', 'Errors explain how to recover', 'Conflict preserves unsaved edits', 'Phone layout remains usable'].map(item => <Checkbox key={item} label={item}/>)}
        <p>Use Tab and Shift+Tab inside the preview. In the conflict scenario, edit a rule and save to inspect recovery.</p>
      </aside>
    </div>
  </main>;
}

import React from 'react';
import { createRoot } from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import './studio.css';
import { ACLReviewFrame, ACLReviewWindow, LiveACLReview } from './ACLReviewWindow.jsx';
import { liveACLReviewResource } from './aclReviewURL.js';
import { reviewScenarios } from './aclReviewFixture.js';
import { defaultActionsByKind } from './resourceAccessActions.js';
import { loadConfig } from './config.js';
import { StudioAPI } from './studioApi.js';

const query = new URLSearchParams(window.location.search);
const scenario = Object.hasOwn(reviewScenarios, query.get('scenario')) ? query.get('scenario') : 'editable';
const kind = Object.hasOwn(defaultActionsByKind, query.get('kind')) ? query.get('kind') : 'component';
const root = createRoot(document.getElementById('root'));
try {
  const resource = liveACLReviewResource(query);
  if (resource) {
    loadConfig().then(config => root.render(<LiveACLReview api={new StudioAPI(config)} resource={resource}/>))
      .catch(error => root.render(<main className="acl-review-window"><div role="alert">Permissions review could not load: {error.message}</div></main>));
  } else {
    root.render(query.get('frame') === '1' ? <ACLReviewFrame scenario={scenario} kind={kind}/> : <ACLReviewWindow/>);
  }
} catch (error) {
  root.render(<main className="acl-review-window"><div role="alert">{error.message}</div></main>);
}

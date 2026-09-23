import React from 'react';
import { createRoot } from 'react-dom/client';
import '@blueprintjs/core/lib/css/blueprint.css';
import './studio.css';
import { ACLReviewFrame, ACLReviewWindow } from './ACLReviewWindow.jsx';
import { reviewScenarios } from './aclReviewFixture.js';

const query = new URLSearchParams(window.location.search);
const scenario = Object.hasOwn(reviewScenarios, query.get('scenario')) ? query.get('scenario') : 'editable';
const kind = query.get('kind') === 'skill' ? 'skill' : 'component';
createRoot(document.getElementById('root')).render(query.get('frame') === '1' ? <ACLReviewFrame scenario={scenario} kind={kind}/> : <ACLReviewWindow/>);

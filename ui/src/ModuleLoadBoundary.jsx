import React from 'react';
import { Button, Callout } from '@blueprintjs/core';

// Never automatically discard an editor's unsaved work after a deployment.
export class ModuleLoadBoundary extends React.Component {
  state = { error: null };
  static getDerivedStateFromError(error) { return { error }; }
  componentDidCatch(error) { console.error('Studio module could not load', error); }
  render() {
    if (!this.state.error) return this.props.children;
    return <Callout intent="warning" title="This view could not load" role="alert">
      <p>The application may have been updated. Save any work in other views before reloading.</p>
      <Button onClick={() => window.location.reload()}>Reload application</Button>
      {this.props.onBack && <Button minimal onClick={this.props.onBack}>Back to components</Button>}
    </Callout>;
  }
}

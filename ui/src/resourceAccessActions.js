// These are UI authoring contracts, not authorization decisions. An embedding
// app must explicitly supply an action list for any additional resource kind.
export const defaultActionsByKind = Object.freeze({
  component: Object.freeze(['discover', 'describe', 'execute', 'export', 'edit', 'validate', 'publish', 'unpublish', 'viewAccess', 'manageAccess']),
  skill: Object.freeze(['discover', 'describe', 'retrieve', 'export', 'edit', 'validate', 'publish', 'unpublish', 'viewAccess', 'manageAccess']),
  report: Object.freeze(['discover', 'describe', 'preview', 'execute', 'export', 'edit', 'validate', 'publish', 'unpublish', 'viewAccess', 'manageAccess']),
});

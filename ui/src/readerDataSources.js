// Reader lookup contracts use the same mcp_tool datasource model as management
// windows. They are scoped by the current component/version, never by SQL text.
export const readerDataSources = {
  codecs: {id:'reader_codecs',cardinality:'collection',backend:{kind:'mcp_tool',method:'studio.sdk.codecs.types'},selectors:{data:'items'},cache:{enabled:false}},
  definition: {id:'reader_definition',cardinality:'object',backend:{kind:'mcp_tool',method:'studio.sdk.versions.inspect'},cache:{enabled:false}},
  authorizationPredicates: {id:'reader_authorization_predicates',cardinality:'collection',backend:{kind:'mcp_tool',method:'studio.sdk.authorization_predicates.list'},selectors:{data:'items'},cache:{enabled:false}},
  predicateTypes: {id:'reader_predicate_types',cardinality:'collection',backend:{kind:'mcp_tool',method:'studio.sdk.authorization_predicates.types'},selectors:{data:'items'},cache:{enabled:false}},
};
export async function fetchReaderDataSource(api,name,input={}) {
  const source=readerDataSources[name];
  if(!source)throw new Error('Unknown reader lookup datasource.');
  return api.fetchMCPDataSource(source,input);
}

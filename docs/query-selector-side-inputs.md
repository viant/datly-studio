# Datly v1 query selectors

Datly v1 `53df1ed7` generates declared view selectors as typed side inputs.
The DQL declaration remains the source of truth: use one-argument
`QuerySelector('view')` on canonical `Fields`, `OrderBy`, `Limit`, `Offset`,
`Page`, or `Criteria` declarations. The query key remains whatever the
declaration binds. `OrderBy<[]string>` is required for a generated side
selector; the earlier `OrderBy<string>` stayed in the business `Input`.

Eight Studio readers now declare `OrderBy<[]string>` and have Datly-generated
selector companions on their component holders. Fields, OrderBy, Limit and
Offset are no longer copied into those readers' business `Input`/`Has` or
setter files. Their existing output/view and SQL projections are retained:
schema-free transcription would replace typed projections with `SELECT *`,
while schema-enabled transcription currently reports an ambiguous legacy
`Input` type. Only the selector input/router artifacts from native
transcription were accepted.

The internal authorization-predicate reader now supplies its trusted paging
and ordering selector through `ComponentRequest.Providers`. The SDK transport
uses that same reader rather than duplicating binding code. The normal Datly
reader still validates the allowed ordering and page limits. The report
version reader retains direct `SetLimit`/`SetOffset` Go callers; the
publication-event reader has separately authored component files. Both keep
their compatible direct selector fields pending a dedicated caller/source
migration.

Datly Studio pins Datly v1 `53df1ed7`. Its temporary remote-version replace
for `mcp-protocol` retains the published `extension/namespaces` API: the
version required by Datly removed that package, and the local unpublished
replacement must not be assumed available in a clean build. Remove the
replace after a compatible protocol release and a clean module build.

Validation: all `./studio/...` packages compile, the SQL SDK transport suite
passes, and focused DQL contract cases for the migrated readers pass. The
full static DQL contract suite still reports unrelated writer helper-field
projection failures on Datly v1; it is not a green migration gate.

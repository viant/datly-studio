# Cubes and cube composition

Studio exposes the Datly reader's cube contract rather than maintaining a
separate reporting engine. Authors choose dimension and measure roles, configure
permitted filters and outputs, enable Cube/Composition in component settings,
and validate the saved version. Enabling Cube also marks the root view groupable
in the same saved operation so the published cube tool is actually derived.

A named reader exposure may produce these tools when enabled:

```text
<name>
<name>Cube
<name>CubeCompose
```

Use runtime discovery to obtain the actual published names and schemas.

## Author aggregate column types

After editing view SQL, Studio refreshes the compiled column metadata. If the
SQL driver cannot report an expression's type (for example SQLite `SUM(amount)`
in an empty discovery query), open **Manage columns**, enter the **SQL output
alias** such as `total`, and declare its **Go cast**, such as `int64`. Datly owns
and validates that typed column declaration. Once columns resolve, assign their
**Cube role** and enable Cube plus field selection in component settings.

## Cube

A cube request selects allowed dimensions and measures and supplies typed filters.
The runtime produces the aggregation from the authored contract. Tests should
make multiple calls with different groupings, filters and pagination, compare
exact aggregates, and repeat a request to detect state leaking between calls.

## Cube compose

Composition operates over ordered cube frames. The wrapper SQL determines each frame’s selected dimensions and measures.
Frames supply explicit filters; filter inheritance names a prior frame. The
composition SQL refers to `$CubeSQL1` through `$CubeSQLN` and validated output
aliases. The authored configuration bounds frame counts and other limits.

Fact data and an outer dictionary can use different database connectors. Author
complete dictionary keys and relation semantics in the component graph; an MCP
caller must not supply arbitrary database handles or SQL table names. Verify
unknown/missing dictionary entries, key completeness and requested dimension
combinations, not just the presence of enriched labels.

Authorization applies to the published source component and its dependencies.
Composition must not broaden entity scope or make another user's hidden data
available through an alternate tool. Use allowed and denied callers as well as
multiple successful cube/compose requests.

Endly case `014_ui_cube_compose` authors both connectors and an optional single
lookup through UI, using separate fact and dictionary SQLite files. It verifies
lookup labels and unmatched rows on source-cube results, multiple scalar compose
queries, inherited filters, repeated-request isolation, frame budgets and denied
users. It also runs the Composition lab and native authoring MCP compose tool.
The focused run passes 105 assertions (see `e2e/results.md`). As requested, dictionary labels are attached to source-cube results only;
composed responses contain the selected scalar columns.

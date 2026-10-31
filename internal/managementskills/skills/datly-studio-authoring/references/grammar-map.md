# DQL grammar by task

Load one topic first, then the formal grammar only when syntax is uncertain.
These references explain the native contract; they do not define another parser.

| Task | Read |
| --- | --- |
| Package, connector, route, input/output holders | [DQL reader shape](dql.md) |
| Typed columns, nulls, aliases and output dates | [Column output how-to](column-output-howto.md) |
| Dimensions, measures and reusable cube grain | [Cubes and dictionaries](cubes-and-dictionaries.md) |
| Cross-connector relations and lazy labels | [Complete cube example](worked-example.md) |
| Create/edit/validate/release versions | [Workflow](workflow.md), [version-session how-to](version-session-howto.md) |
| Producer schemas and dashboard handoff | [Report consumers](report-consumers.md) |
| Structural syntax | [Native DQL EBNF](dql.ebnf.txt) |
| Builder operations | [Native operation schema](builder-operations.schema.json) |

The complete worked example is synthetic and compiler-tested. It is not a claim
that its sample database tables exist. A live session must discover and validate
the user's authorized connector, source schema and predicates before execution.
